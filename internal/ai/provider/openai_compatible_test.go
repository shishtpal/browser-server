package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestStreamParsesDeltasUsageAndToolFragments(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\",\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"get_\",\"arguments\":\"{\"}}]}}]}\n\ndata: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"current_time\",\"arguments\":\"}\"}}]}}],\"usage\":{\"total_tokens\":3}}\n\ndata: [DONE]\n\n"))
	}))
	defer s.Close()
	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second)
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
	if err != nil || resp.Content != "hi" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_current_time" || resp.ToolCalls[0].Arguments != "{}" {
		t.Fatalf("response=%+v err=%v", resp, err)
	}
}

func TestStreamRejectsPrematureEOF(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second)
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
	if err == nil || resp.Content != "partial" {
		t.Fatalf("response=%+v err=%v", resp, err)
	}
}

func TestPayloadEncodesAssistantToolCallsInOpenAIFormat(t *testing.T) {
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second, 0, time.Second)
	payload, err := json.Marshal(c.payload(ChatRequest{Model: "m", Messages: []Message{{
		Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Name: "get_current_time", Arguments: "{}"}},
	}}}, false))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages []struct {
			ToolCalls []struct {
				Type     string `json:"type"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Messages) != 1 || len(decoded.Messages[0].ToolCalls) != 1 || decoded.Messages[0].ToolCalls[0].Type != "function" || decoded.Messages[0].ToolCalls[0].Function.Name != "get_current_time" {
		t.Fatalf("payload=%s", payload)
	}
}

func TestCompleteReturnsToolCalls(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_current_time","arguments":"{}"}}]}}]}`))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second)
	resp, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "c1" || resp.ToolCalls[0].Name != "get_current_time" {
		t.Fatalf("tool calls=%+v", resp.ToolCalls)
	}
}

func TestCompleteRetriesTransientFailures(t *testing.T) {
	var attempts atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) <= 2 {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 2, time.Millisecond)
	resp, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err != nil || resp.Content != "ok" || attempts.Load() != 3 {
		t.Fatalf("attempts=%d response=%+v err=%v", attempts.Load(), resp, err)
	}
}

func TestCompleteDoesNotRetryClientErrors(t *testing.T) {
	var attempts atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(w, "invalid request", http.StatusBadRequest)
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 2, time.Millisecond)
	_, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err == nil || attempts.Load() != 1 {
		t.Fatalf("attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestStreamRetriesBeforeEmittingContent(t *testing.T) {
	var attempts atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(w, "temporarily unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 1, time.Millisecond)
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
	if err != nil || resp.Content != "ok" || attempts.Load() != 2 {
		t.Fatalf("attempts=%d response=%+v err=%v", attempts.Load(), resp, err)
	}
}

func TestCompleteRetriesProviderErrorInBody(t *testing.T) {
	var attempts atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) <= 2 {
			// HTTP 200 with error body (common with OpenRouter)
			_, _ = w.Write([]byte(`{"error":{"message":"Service temporarily overloaded","type":"server_error","code":"503"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"success"}}]}`))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 3, time.Millisecond)
	resp, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err != nil || resp.Content != "success" || attempts.Load() != 3 {
		t.Fatalf("attempts=%d response=%+v err=%v", attempts.Load(), resp, err)
	}
}

func TestErrorIncludesDiagnostic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second)
	_, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should include the diagnostic, not just the code
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if providerErr.Diagnostic == "" {
		t.Fatal("diagnostic should not be empty")
	}
	if providerErr.Error() == providerErr.Code {
		t.Fatalf("Error() should include diagnostic, got %q", providerErr.Error())
	}
}

func TestCompleteRespectsContextCancellation(t *testing.T) {
	var attempts atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(w, "overloaded", http.StatusServiceUnavailable)
	}))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after first attempt
	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 5, 50*time.Millisecond)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := c.Complete(ctx, ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected error after cancellation")
	}
	// Should have stopped retrying promptly
	if attempts.Load() > 2 {
		t.Fatalf("expected at most 2 attempts, got %d", attempts.Load())
	}
}

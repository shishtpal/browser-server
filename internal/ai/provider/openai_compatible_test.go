package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamParsesDeltasUsageAndToolFragments(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\",\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"get_\",\"arguments\":\"{\"}}]}}]}\n\ndata: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"current_time\",\"arguments\":\"}\"}}]}}],\"usage\":{\"total_tokens\":3}}\n\ndata: [DONE]\n\n"))
	}))
	defer s.Close()
	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second)
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second)
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
	if err == nil || resp.Content != "partial" {
		t.Fatalf("response=%+v err=%v", resp, err)
	}
}

func TestPayloadEncodesAssistantToolCallsInOpenAIFormat(t *testing.T) {
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second)
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second)
	resp, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "c1" || resp.ToolCalls[0].Name != "get_current_time" {
		t.Fatalf("tool calls=%+v", resp.ToolCalls)
	}
}

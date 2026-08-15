package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeminiCompleteParsesStepsAndUsage(t *testing.T) {
	var gotBody string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/interactions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"steps": [
				{"type": "thought", "content": [{"type": "text", "text": "Let me add."}]},
				{"type": "function_call", "id": "fc1", "name": "get_current_time", "arguments": "{\"tz\":\"UTC\"}"},
				{"type": "model_output", "content": [{"type": "text", "text": "It is 12:00."}]}
			],
			"usage": {"total_input_tokens": 10, "total_output_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer s.Close()

	c := NewGeminiInteractionsClient(s.URL, "secret", time.Second, 0, time.Second)
	resp, err := c.Complete(context.Background(), ChatRequest{
		Model:    "models/gemini-3-flash",
		Messages: []Message{{Role: "system", Content: "be brief"}, {Role: "user", Content: "what time is it?"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "It is 12:00." {
		t.Errorf("Content = %q", resp.Content)
	}
	if !strings.Contains(resp.Reasoning, "Let me add.") {
		t.Errorf("Reasoning = %q", resp.Reasoning)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_current_time" || resp.ToolCalls[0].Arguments != `{"tz":"UTC"}` {
		t.Errorf("ToolCalls = %+v", resp.ToolCalls)
	}
	if resp.Usage.PromptTokens == nil || *resp.Usage.PromptTokens != 10 || resp.Usage.TotalTokens == nil || *resp.Usage.TotalTokens != 15 {
		t.Errorf("Usage = %+v", resp.Usage)
	}

	var req struct {
		Model             string `json:"model"`
		SystemInstruction string `json:"system_instruction"`
		Stream            bool   `json:"stream"`
		Store             bool   `json:"store"`
		Input             []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal([]byte(gotBody), &req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if req.Model != "gemini-3-flash" {
		t.Errorf("model = %q (prefix should be stripped)", req.Model)
	}
	if req.SystemInstruction != "be brief" || req.Stream || req.Store {
		t.Errorf("request = %+v", req)
	}
	if len(req.Input) != 1 || req.Input[0].Type != "user_input" {
		t.Errorf("input = %+v", req.Input)
	}
}

func TestGeminiCompleteErrorEnvelope(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": 403, "message": "API key not valid", "status": "PERMISSION_DENIED"}}`))
	}))
	defer s.Close()

	c := NewGeminiInteractionsClient(s.URL, "secret", time.Second, 0, time.Second)
	_, err := c.Complete(context.Background(), ChatRequest{Model: "m"})
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if providerErr.Code != "provider_error" || providerErr.Status != http.StatusBadGateway || providerErr.Retryable || !strings.Contains(providerErr.Diagnostic, "API key not valid") {
		t.Errorf("error = %+v", providerErr)
	}
}

func TestNewFactoryDispatchesByType(t *testing.T) {
	if _, ok := New("gemini_interactions", "", "", time.Second, 0, time.Second).(*GeminiInteractionsClient); !ok {
		t.Error("gemini_interactions should build a GeminiInteractionsClient")
	}
	if _, ok := New("openai_compatible", "http://localhost", "k", time.Second, 0, time.Second).(*OpenAICompatibleClient); !ok {
		t.Error("openai_compatible should build an OpenAICompatibleClient")
	}
	if _, ok := New("bogus", "http://localhost", "k", time.Second, 0, time.Second).(*OpenAICompatibleClient); !ok {
		t.Error("unknown types should fall back to the OpenAI-compatible client")
	}
}

func TestGeminiStreamParsesDeltasAndTerminalEvents(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			"data: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"Hi \"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"there\"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"thought_summary\",\"content\":{\"type\":\"text\",\"text\":\"checking\"}}}\n\n" +
				"data: {\"event_type\":\"step.start\",\"index\":1,\"step\":{\"type\":\"function_call\",\"name\":\"get_current_time\"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":1,\"delta\":{\"type\":\"arguments_delta\",\"arguments\":\"{\\\"tz\\\":\"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":1,\"delta\":{\"type\":\"arguments_delta\",\"arguments\":\"\\\"UTC\\\"}\"}}\n\n" +
				"data: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"i1\",\"status\":\"completed\",\"usage\":{\"total_input_tokens\":10,\"total_output_tokens\":5,\"total_tokens\":15}}}\n\n",
		))
	}))
	defer s.Close()

	c := NewGeminiInteractionsClient(s.URL, "secret", time.Second, 0, time.Second)
	var kinds []string
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(ev Event) error {
		kinds = append(kinds, ev.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if resp.Content != "Hi there" {
		t.Errorf("Content = %q", resp.Content)
	}
	if resp.Reasoning != "checking" {
		t.Errorf("Reasoning = %q", resp.Reasoning)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_current_time" || resp.ToolCalls[0].Arguments != `{"tz":"UTC"}` {
		t.Errorf("ToolCalls = %+v", resp.ToolCalls)
	}
	if resp.Usage.TotalTokens == nil || *resp.Usage.TotalTokens != 15 {
		t.Errorf("Usage = %+v", resp.Usage)
	}
	joined := strings.Join(kinds, ",")
	for _, want := range []string{"text_delta", "reasoning_delta", "tool_call", "done"} {
		if !strings.Contains(joined, want) {
			t.Errorf("events %q missing %q", joined, want)
		}
	}
}

func TestGeminiStreamRejectsPrematureEOF(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"partial\"}}\n\n"))
	}))
	defer s.Close()

	c := NewGeminiInteractionsClient(s.URL, "secret", time.Second, 0, time.Second)
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, nil)
	var providerErr *Error
	if !errors.As(err, &providerErr) || providerErr.Code != "malformed_provider_stream" {
		t.Fatalf("err = %v", err)
	}
	if resp.Content != "partial" {
		t.Errorf("Content = %q", resp.Content)
	}
}

func TestGeminiPayloadMapsConversationAndGoogleSearch(t *testing.T) {
	c := NewGeminiInteractionsClient("http://localhost", "secret", time.Second, 0, time.Second)
	req := ChatRequest{
		Model:        "gemini-3-flash",
		GoogleSearch: true,
		Messages: []Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "sum"},
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Name: "add", Arguments: `{"a":1}`}}},
			{Role: "tool", ToolCallID: "c1", Content: "ok"},
		},
		Tools: []ToolSpec{{Name: "add", Parameters: json.RawMessage(`{"type":"object"}`)}},
	}
	p := c.payload(req, false)
	if p.Model != "gemini-3-flash" || p.Stream || p.Store {
		t.Errorf("payload base = %+v", p)
	}
	if p.SystemInstruction != "sys" {
		t.Errorf("SystemInstruction = %q", p.SystemInstruction)
	}
	var stepTypes []string
	for _, step := range p.Input {
		stepTypes = append(stepTypes, step.Type)
	}
	if strings.Join(stepTypes, ",") != "user_input,function_call,function_result" {
		t.Errorf("step types = %v", stepTypes)
	}
	var toolTypes []string
	for _, tool := range p.Tools {
		toolTypes = append(toolTypes, tool.Type)
	}
	if strings.Join(toolTypes, ",") != "function,google_search" {
		t.Errorf("tool types = %v", toolTypes)
	}
	if p.Input[1].ID != "c1" || p.Input[1].Name != "add" {
		t.Errorf("function_call step = %+v", p.Input[1])
	}
	var args string
	if err := json.Unmarshal(p.Input[1].Arguments, &args); err != nil || args != `{"a":1}` {
		t.Errorf("function_call arguments = %s (err %v)", p.Input[1].Arguments, err)
	}
	if p.Input[2].CallID != "c1" || p.Input[2].Name != "add" {
		t.Errorf("function_result step = %+v", p.Input[2])
	}
}

func TestGeminiStreamCollectsToolCallsAtNonZeroIndices(t *testing.T) {
	// A thought step occupies index 0, so the function calls land at indices
	// 1 and 3 (gap at 2). A len(calls)-bounded aggregation loop would drop
	// them and the turn would end having executed no tools.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			"data: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"thought_summary\",\"content\":{\"type\":\"text\",\"text\":\"planning\"}}}\n\n" +
				"data: {\"event_type\":\"step.start\",\"index\":1,\"step\":{\"type\":\"function_call\",\"id\":\"c1\",\"name\":\"get_current_time\"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":1,\"delta\":{\"type\":\"arguments_delta\",\"arguments\":\"{\\\"tz\\\":\\\"UTC\\\"}\"}}\n\n" +
				"data: {\"event_type\":\"step.start\",\"index\":3,\"step\":{\"type\":\"function_call\",\"id\":\"c2\",\"name\":\"search_bookmarks\"}}\n\n" +
				"data: {\"event_type\":\"step.delta\",\"index\":3,\"delta\":{\"type\":\"arguments_delta\",\"arguments\":\"{\\\"q\\\":\\\"go\\\"}\"}}\n\n" +
				"data: {\"event_type\":\"interaction.status_update\",\"status\":\"requires_action\"}\n\n",
		))
	}))
	defer s.Close()

	c := NewGeminiInteractionsClient(s.URL, "secret", time.Second, 0, time.Second)
	var names []string
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(ev Event) error {
		if ev.Type == "tool_call" && ev.ToolCall != nil {
			names = append(names, ev.ToolCall.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %+v, want the 2 calls at indices 1 and 3", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Name != "get_current_time" || resp.ToolCalls[1].Name != "search_bookmarks" {
		t.Errorf("ToolCalls = %+v", resp.ToolCalls)
	}
	if strings.Join(names, ",") != "get_current_time,search_bookmarks" {
		t.Errorf("tool_call events = %v", names)
	}
}

func TestFunctionCallArgumentsJSON(t *testing.T) {
	cases := []struct {
		raw  string
		want string // exact JSON string literal the API receives
	}{
		// Empty and "null" arguments collapse to the string "{}" (no-arg call).
		{raw: "", want: `"{}"`},
		{raw: "null", want: `"{}"`},
		{raw: "{}", want: `"{}"`},
		{raw: `{"a":1}`, want: `"{\"a\":1}"`},
		{raw: `{"tz":"UTC"}`, want: `"{\"tz\":\"UTC\"}"`},
	}
	for _, tc := range cases {
		got := functionCallArgumentsJSON(tc.raw)
		if string(got) != tc.want {
			t.Errorf("functionCallArgumentsJSON(%q) = %s, want %s", tc.raw, got, tc.want)
		}
		// The wire value must always be a valid JSON string that decodes back
		// to the arguments text the tool runner produced.
		var decoded string
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Errorf("functionCallArgumentsJSON(%q) = %s, not a JSON string: %v", tc.raw, got, err)
		}
	}
}

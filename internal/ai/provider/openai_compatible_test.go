package provider

import (
	"browser-server/internal/ai/openrouter"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
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
	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second, "", "")
	resp, err := c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
	if err == nil || resp.Content != "partial" {
		t.Fatalf("response=%+v err=%v", resp, err)
	}
}

func TestPayloadEncodesAssistantToolCallsInOpenAIFormat(t *testing.T) {
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second, 0, time.Second, "", "")
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

func TestPayloadKeepsTextOnlyContentAsAString(t *testing.T) {
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second, 0, time.Second, "", "")
	payload, err := json.Marshal(c.payload(ChatRequest{Model: "m", Messages: []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}}, false))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	// Text-only messages must stay as plain strings (not content-part arrays)
	// for provider compatibility.
	if len(decoded.Messages) != 3 {
		t.Fatalf("payload=%s", payload)
	}
	for _, m := range decoded.Messages {
		if m.Content == "" {
			t.Fatalf("text-only content should be a string, got empty for %s: %s", m.Role, payload)
		}
	}
}

func TestPayloadEncodesImageAttachmentsAsContentParts(t *testing.T) {
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second, 0, time.Second, "", "")
	payload, err := json.Marshal(c.payload(ChatRequest{Model: "m", Messages: []Message{
		{Role: "user", Content: "describe this", ImageParts: []ImagePart{
			{DataURL: "data:image/png;base64,Qk=="},
			{DataURL: "data:image/jpeg;base64,Ug=="},
		}},
	}}, false))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text,omitempty"`
				ImageURL *struct {
					URL string `json:"url"`
				} `json:"image_url,omitempty"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Messages) != 1 {
		t.Fatalf("payload=%s", payload)
	}
	parts := decoded.Messages[0].Content
	// First the text part, then the two image_url parts, in order.
	if len(parts) != 3 || parts[0].Type != "text" || parts[0].Text != "describe this" {
		t.Fatalf("text part not first: %+v payload=%s", parts, payload)
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil || parts[1].ImageURL.URL != "data:image/png;base64,Qk==" {
		t.Fatalf("first image part wrong: %+v", parts[1])
	}
	if parts[2].Type != "image_url" || parts[2].ImageURL == nil || parts[2].ImageURL.URL != "data:image/jpeg;base64,Ug==" {
		t.Fatalf("second image part wrong: %+v", parts[2])
	}
}

func TestPayloadImagePartsOmitTextWhenEmpty(t *testing.T) {
	// An image-only user message (no text) must not emit an empty text part.
	c := NewOpenAICompatibleClient("http://localhost", "secret", time.Second, 0, time.Second, "", "")
	payload, err := json.Marshal(c.payload(ChatRequest{Model: "m", Messages: []Message{
		{Role: "user", Content: "", ImageParts: []ImagePart{{DataURL: "data:image/png;base64,Qk=="}}},
	}}, false))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages []struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Messages) != 1 || len(decoded.Messages[0].Content) != 1 || decoded.Messages[0].Content[0].Type != "image_url" {
		t.Fatalf("expected exactly one image_url part, payload=%s", payload)
	}
}

func TestCompleteReturnsToolCalls(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_current_time","arguments":"{}"}}]}}]}`))
	}))
	defer s.Close()

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 2, time.Millisecond, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 2, time.Millisecond, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 1, time.Millisecond, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 3, time.Millisecond, "", "")
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

	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 0, time.Second, "", "")
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
	c := NewOpenAICompatibleClient(s.URL, "secret", time.Second, 5, 50*time.Millisecond, "", "")
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

func TestOpenRouterHeadersSentOnlyForOpenRouter(t *testing.T) {
	orClient := NewOpenAICompatibleClient("https://openrouter.ai/api/v1", "secret", time.Second, 0, time.Second, "https://example.com/app", "My App")
	h := http.Header{}
	openrouter.SetAttributionHeaders(h, orClient.baseURL, orClient.openRouterSiteURL, orClient.openRouterAppName)

	if h.Get("HTTP-Referer") != "https://example.com/app" {
		t.Errorf("HTTP-Referer = %q", h.Get("HTTP-Referer"))
	}
	if h.Get("Referer") != "https://example.com/app" {
		t.Errorf("Referer = %q", h.Get("Referer"))
	}
	if h.Get("X-Title") != "My App" {
		t.Errorf("X-Title = %q", h.Get("X-Title"))
	}

	// Detection is case-insensitive and covers subdomains.
	orCase := NewOpenAICompatibleClient("https://api.OPENROUTER.ai/v1", "secret", time.Second, 0, time.Second, "https://example.com/app", "My App")
	hCase := http.Header{}
	openrouter.SetAttributionHeaders(hCase, orCase.baseURL, orCase.openRouterSiteURL, orCase.openRouterAppName)
	if hCase.Get("X-Title") != "My App" {
		t.Errorf("case-insensitive detection failed: X-Title = %q", hCase.Get("X-Title"))
	}

	// Non-OpenRouter providers receive no attribution headers.
	orOther := NewOpenAICompatibleClient("https://api.openai.com/v1", "secret", time.Second, 0, time.Second, "https://example.com/app", "My App")
	hOther := http.Header{}
	openrouter.SetAttributionHeaders(hOther, orOther.baseURL, orOther.openRouterSiteURL, orOther.openRouterAppName)
	if hOther.Get("HTTP-Referer") != "" || hOther.Get("X-Title") != "" || hOther.Get("Referer") != "" {
		t.Errorf("non-OpenRouter provider should not receive attribution headers: %v", hOther)
	}

	// An empty app name is omitted rather than sent as an empty X-Title, while
	// the referer headers still go out.
	noTitle := NewOpenAICompatibleClient("https://openrouter.ai/api/v1", "secret", time.Second, 0, time.Second, "https://example.com/app", "")
	hNoTitle := http.Header{}
	openrouter.SetAttributionHeaders(hNoTitle, noTitle.baseURL, noTitle.openRouterSiteURL, noTitle.openRouterAppName)
	if hNoTitle.Get("X-Title") != "" {
		t.Errorf("empty X-Title should be omitted, got %q", hNoTitle.Get("X-Title"))
	}
	if hNoTitle.Get("HTTP-Referer") != "https://example.com/app" {
		t.Errorf("HTTP-Referer = %q", hNoTitle.Get("HTTP-Referer"))
	}
}

// routeToAddr returns an http.Transport that dials addr for every request
// regardless of the Host in the request URL, so a test can keep an
// "https://openrouter.ai/..." base URL while the local test server
// intercepts the traffic.
func routeToAddr(addr string) *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	}
	return tr
}

// TestCompleteAndStreamOpenRouterAttributionHeaders verifies end-to-end that
// both request paths attach the attribution headers only when the provider
// base URL actually points at OpenRouter, and that lookalike hosts receive
// nothing. This guards the setOpenRouterHeaders call sites in completeOnce and
// streamOnce, not just the helper.
func TestCompleteAndStreamOpenRouterAttributionHeaders(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		open    bool
	}{
		{"openrouter root host", "http://openrouter.ai/api/v1", true},
		{"openrouter subdomain", "http://api.openrouter.ai/v1", true},
		{"openrouter case-insensitive", "http://API.OpenRouter.ai/v1", true},
		{"openrouter with port", "http://openrouter.ai:8080/v1", true},
		{"non-openrouter host", "http://api.example.com/v1", false},
		{"lookalike host", "http://myopenrouter.ai/v1", false},
		{"suffix attack host", "http://openrouter.ai.evil.example.com/v1", false},
	}
	for _, stream := range []bool{false, true} {
		method := "Complete"
		if stream {
			method = "Stream"
		}
		for _, tc := range cases {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				var mu sync.Mutex
				var headers []http.Header
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					headers = append(headers, r.Header.Clone())
					mu.Unlock()
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))
						return
					}
					_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
				}))
				defer s.Close()

				c := NewOpenAICompatibleClient(tc.baseURL, "secret", time.Second, 0, time.Second, "https://example.com/app", "My App")
				// Route the base URL's (possibly openrouter.ai) host through the
				// local test server. Detection is host-only, so the plain-http
				// scheme is fine for exercising header attachment.
				c.httpClient.Transport = routeToAddr(s.Listener.Addr().String())

				var err error
				if stream {
					_, err = c.Stream(context.Background(), ChatRequest{Model: "m"}, func(Event) error { return nil })
				} else {
					_, err = c.Complete(context.Background(), ChatRequest{Model: "m"})
				}
				if err != nil {
					t.Fatalf("%s failed: %v", method, err)
				}

				mu.Lock()
				defer mu.Unlock()
				if len(headers) != 1 {
					t.Fatalf("expected 1 request, got %d", len(headers))
				}
				h := headers[0]
				if (h.Get("HTTP-Referer") != "") != tc.open {
					t.Errorf("HTTP-Referer = %q, want present=%v", h.Get("HTTP-Referer"), tc.open)
				}
				if (h.Get("Referer") != "") != tc.open {
					t.Errorf("Referer = %q, want present=%v", h.Get("Referer"), tc.open)
				}
				if (h.Get("X-Title") != "") != tc.open {
					t.Errorf("X-Title = %q, want present=%v", h.Get("X-Title"), tc.open)
				}
				if tc.open && h.Get("X-Title") != "My App" {
					t.Errorf("X-Title = %q, want %q", h.Get("X-Title"), "My App")
				}
			})
		}
	}
}

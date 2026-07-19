package provider

import (
	"context"
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

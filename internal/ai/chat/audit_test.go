package chat

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

func TestAuditRequestNormalizesStreamResponseAndUsesCallContextCancellation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	service := &Service{cfg: &aiconfig.Config{
		Logging:   aiconfig.LoggingConfig{Enabled: true, LogFullPayload: true, MaxPayloadBytes: 4096},
		Providers: map[string]aiconfig.ProviderConfig{"test": {BaseURL: "https://example.test/v1"}},
	}, store: st}

	id := service.auditRequest(context.Background(), "chat", "", "", "", 0, "test", "model", provider.ChatResponse{
		Content: "streamed answer",
		ToolCalls: []provider.ToolCall{{
			ID: "call-1", Name: "example", Arguments: "{\"api_key\":\"secret-value\"}",
		}},
	}, nil)
	logs, err := st.ListRequestLogs(context.Background(), store.LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ID != id || !strings.Contains(logs[0].ResponsePayload, "streamed answer") {
		t.Fatalf("stream response was not captured: %#v", logs)
	}
	if strings.Contains(logs[0].ResponsePayload, "secret-value") {
		t.Fatalf("stream response was not redacted: %s", logs[0].ResponsePayload)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	service.auditRequest(cancelled, "task_agent", "", "", "task-1", 1, "test", "model", provider.ChatResponse{}, &provider.Error{Code: "provider_error"})
	logs, err = st.ListRequestLogs(context.Background(), store.LogFilter{TaskID: "task-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Status != "cancelled" || logs[0].ErrorCode != "cancelled" {
		t.Fatalf("cancelled call audit = %#v", logs)
	}
}

func TestAuditToolOmitsDetailedErrorWhenFullPayloadLoggingIsDisabled(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	service := &Service{cfg: &aiconfig.Config{
		Logging:   aiconfig.LoggingConfig{Enabled: true, MaxPayloadBytes: 4096},
		Providers: map[string]aiconfig.ProviderConfig{"test": {BaseURL: "https://example.test/v1"}},
	}, store: st}
	requestID := service.auditRequest(context.Background(), "chat", "", "", "", 0, "test", "model", provider.ChatResponse{}, nil)
	service.AuditTool(requestID, "", "edit_file", "{\"secret\":\"argument-value\"}", []byte("result-value"), errors.New("sensitive file content"), "error", "approved", time.Millisecond)
	logs, err := st.ListRequestLogs(context.Background(), store.LogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || len(logs[0].ToolCalls) != 1 {
		t.Fatalf("tool audit = %#v", logs)
	}
	call := logs[0].ToolCalls[0]
	if call.Arguments != "" || call.Result != "" || call.ErrorMessage != "tool execution failed" {
		t.Fatalf("payload-disabled tool audit = %#v", call)
	}
}

package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRequestLogsToolsFiltersAndMonitoring(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	total := 7
	entry := RequestLog{ID: "req_test", Source: "task_agent", TaskID: "task_1", Iteration: 2, Provider: "p", Model: "m", Endpoint: "endpoint", Status: "success", TotalTokens: &total, LatencyMS: 25}
	if err := s.InsertRequestLog(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertToolCall(ctx, ToolCall{RequestID: entry.ID, ToolName: "read_file", Arguments: `{"path":"x"}`, Result: `{"ok":true}`, Status: "success", Decision: "replayed", DurationMS: 3}); err != nil {
		t.Fatal(err)
	}
	logs, err := s.ListRequestLogs(ctx, LogFilter{Source: "task_agent", TaskID: "task_1", Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Iteration != 2 || len(logs[0].ToolCalls) != 1 || logs[0].ToolCalls[0].Decision != "replayed" {
		t.Fatalf("unexpected logs: %#v", logs)
	}
	metrics, err := s.Monitoring(ctx, 24)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Requests != 1 || metrics.ToolSuccesses != 1 || metrics.TotalTokens != 7 || metrics.MaxLatencyMS != 25 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestRequestLogsEmptySliceWhenNoRows(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "empty-logs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	logs, err := s.ListRequestLogs(context.Background(), LogFilter{Source: "task_agent", Limit: 25, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if logs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs, got %d", len(logs))
	}
}

func TestRequestLogDeleteCascadesToolAudit(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cascade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.InsertRequestLog(ctx, RequestLog{ID: "req", Provider: "p", Model: "m", Endpoint: "e", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	if err = s.InsertToolCall(ctx, ToolCall{RequestID: "req", ToolName: "t", Status: "success", Decision: "approved"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DELETE FROM request_logs WHERE id='req'`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM tool_calls`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

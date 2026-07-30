package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/todo"
)

func TestManageCalendarValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing user_id",
			input:   `{"action":"add","title":"Meeting","start_date":"2026-08-01"}`,
			wantErr: "user_id",
		},
		{
			name:    "missing action",
			input:   `{"user_id":1,"title":"Meeting","start_date":"2026-08-01"}`,
			wantErr: "action",
		},
		{
			name:    "invalid action",
			input:   `{"user_id":1,"action":"delete"}`,
			wantErr: "action",
		},
		{
			name:    "add missing title",
			input:   `{"user_id":1,"action":"add","start_date":"2026-08-01"}`,
			wantErr: "title",
		},
		{
			name:    "add missing start_date",
			input:   `{"user_id":1,"action":"add","title":"Meeting"}`,
			wantErr: "start_date",
		},
		{
			name:    "add invalid priority",
			input:   `{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01","priority":"super"}`,
			wantErr: "priority",
		},
		{
			name:    "add invalid status",
			input:   `{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01","status":"unknown"}`,
			wantErr: "status",
		},
		{
			name:    "add invalid start_date",
			input:   `{"user_id":1,"action":"add","title":"Meeting","start_date":"not-a-date"}`,
			wantErr: "start_date",
		},
		{
			name:    "edit no fields provided",
			input:   `{"user_id":1,"action":"edit","id":1}`,
			wantErr: "no updatable fields",
		},
		{
			name:    "edit missing id",
			input:   `{"user_id":1,"action":"edit","title":"Updated"}`,
			wantErr: "id",
		},
		{
			name:    "remove missing id",
			input:   `{"user_id":1,"action":"remove"}`,
			wantErr: "id",
		},
		{
			name:    "get missing id",
			input:   `{"user_id":1,"action":"get"}`,
			wantErr: "id",
		},
		{
			name:    "unknown field",
			input:   `{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01","bogus":true}`,
			wantErr: "bogus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manageCalendar(context.Background(), json.RawMessage(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestManageCalendarAddAndGetRejectWrongOwner(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	// Create event for user 1
	_, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01"}`))
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// User 2 tries to get it — should fail
	_, err = manageCalendar(context.Background(), json.RawMessage(`{"user_id":2,"action":"get","id":1}`))
	if err == nil {
		t.Fatal("expected ownership error, got nil")
	}
	if !contains(err.Error(), "does not belong") {
		t.Fatalf("expected ownership error, got %v", err)
	}

	// User 2 tries to edit it — should fail
	_, err = manageCalendar(context.Background(), json.RawMessage(`{"user_id":2,"action":"edit","id":1,"title":"Hacked"}`))
	if err == nil {
		t.Fatal("expected ownership error, got nil")
	}
	if !contains(err.Error(), "does not belong") {
		t.Fatalf("expected ownership error, got %v", err)
	}

	// User 2 tries to remove it — should fail
	_, err = manageCalendar(context.Background(), json.RawMessage(`{"user_id":2,"action":"remove","id":1}`))
	if err == nil {
		t.Fatal("expected ownership error, got nil")
	}
	if !contains(err.Error(), "does not belong") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestManageCalendarRemoveAndGet(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	_, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01"}`))
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// Get it
	got, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"get","id":1}`))
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	m := got.(map[string]any)
	if m["title"] != "Meeting" {
		t.Fatalf("expected title 'Meeting', got %v", m["title"])
	}

	// Remove it
	removed, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"remove","id":1}`))
	if err != nil {
		t.Fatalf("failed to remove event: %v", err)
	}
	rm := removed.(map[string]any)
	if rm["removed"] != true {
		t.Fatalf("expected removed=true, got %v", rm["removed"])
	}

	// Get should now fail
	_, err = manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"get","id":1}`))
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestManageCalendarEdit(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	_, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"add","title":"Meeting","start_date":"2026-08-01"}`))
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	edited, err := manageCalendar(context.Background(), json.RawMessage(`{"user_id":1,"action":"edit","id":1,"title":"Updated Meeting","priority":"high"}`))
	if err != nil {
		t.Fatalf("failed to edit event: %v", err)
	}
	e := edited.(*todo.UpdateResult)
	if e.Title != "Updated Meeting" {
		t.Fatalf("expected title 'Updated Meeting', got %q", e.Title)
	}
	if e.Priority != "high" {
		t.Fatalf("expected priority 'high', got %q", e.Priority)
	}
}


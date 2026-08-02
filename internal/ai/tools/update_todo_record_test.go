package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/todo"
)

func TestUpdateTodoRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing user_id",
			input:   `{"id":1}`,
			wantErr: "user_id",
		},
		{
			name:    "missing id",
			input:   `{"user_id":1}`,
			wantErr: "id",
		},
		{
			name:    "invalid user_id",
			input:   `{"user_id":0,"id":1}`,
			wantErr: "user_id",
		},
		{
			name:    "invalid id",
			input:   `{"user_id":1,"id":0}`,
			wantErr: "id",
		},
		{
			name:    "invalid priority",
			input:   `{"user_id":1,"id":1,"priority":"super"}`,
			wantErr: "priority",
		},
		{
			name:    "invalid status",
			input:   `{"user_id":1,"id":1,"status":"unknown"}`,
			wantErr: "status",
		},
		{
			name:    "title too long",
			input:   `{"user_id":1,"id":1,"title":"` + repeat("x", 501) + `"}`,
			wantErr: "title",
		},
		{
			name:    "description too long",
			input:   `{"user_id":1,"id":1,"description":"` + repeat("x", 2001) + `"}`,
			wantErr: "description",
		},
		{
			name:    "invalid color",
			input:   `{"user_id":1,"id":1,"color":"orange"}`,
			wantErr: "color",
		},
		{
			name:    "invalid position",
			input:   `{"user_id":1,"id":1,"position":-1}`,
			wantErr: "position",
		},
		{
			name:    "invalid start_date",
			input:   `{"user_id":1,"id":1,"start_date":"not-a-date"}`,
			wantErr: "start_date",
		},
		{
			name:    "unknown field",
			input:   `{"user_id":1,"id":1,"bogus":true}`,
			wantErr: "bogus",
		},
		{
			name:    "no updatable fields",
			input:   `{"user_id":1,"id":1}`,
			wantErr: "no updatable fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := updateTodoRecord(context.Background(), json.RawMessage(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestUpdateTodoRecordReplacesAndClearsTags(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	// Create a tagged todo directly so tag state is fully controlled.
	insertTestTodo(t, 1, "Original", `["a","b"]`, "pending", "medium", "")

	// Updating tags replaces the stored set.
	res, err := updateTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"id":1,"tags":["c"]}`))
	if err != nil {
		t.Fatalf("updateTodoRecord failed: %v", err)
	}
	ur, ok := res.(*todo.UpdateResult)
	if !ok {
		t.Fatalf("expected *todo.UpdateResult, got %T", res)
	}
	if !sameStrings(ur.Tags, []string{"c"}) {
		t.Fatalf("expected tags [c], got %v", ur.Tags)
	}

	// An explicit empty array clears tags.
	res2, err := updateTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"id":1,"tags":[]}`))
	if err != nil {
		t.Fatalf("updateTodoRecord failed: %v", err)
	}
	ur2 := res2.(*todo.UpdateResult)
	if len(ur2.Tags) != 0 {
		t.Fatalf("expected cleared tags, got %v", ur2.Tags)
	}
}

func TestUpdateTodoRecordAcceptsInProgressStatus(t *testing.T) {
	// This test verifies that in_progress is accepted at validation level.
	// It will panic at the DB layer (no DB in unit test), so we recover and check.
	var validationErr error
	func() {
		defer func() { recover() }()
		_, validationErr = updateTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"id":1,"status":"in_progress"}`))
	}()
	if validationErr != nil && contains(validationErr.Error(), "status") {
		t.Fatalf("in_progress should be a valid status, got: %v", validationErr)
	}
}

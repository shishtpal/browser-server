package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/todo"
)

func TestAddTodoRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing user_id",
			input:   `{"title":"Buy milk"}`,
			wantErr: "user_id",
		},
		{
			name:    "missing title",
			input:   `{"user_id":1}`,
			wantErr: "title",
		},
		{
			name:    "invalid user_id",
			input:   `{"user_id":0,"title":"Buy milk"}`,
			wantErr: "user_id",
		},
		{
			name:    "invalid priority",
			input:   `{"user_id":1,"title":"Buy milk","priority":"super"}`,
			wantErr: "priority",
		},
		{
			name:    "invalid status",
			input:   `{"user_id":1,"title":"Buy milk","status":"unknown"}`,
			wantErr: "status",
		},
		{
			name:    "title too long",
			input:   `{"user_id":1,"title":"` + repeat("x", 501) + `"}`,
			wantErr: "title",
		},
		{
			name:    "description too long",
			input:   `{"user_id":1,"title":"OK","description":"` + repeat("x", 2001) + `"}`,
			wantErr: "description",
		},
		{
			name:    "unknown field",
			input:   `{"user_id":1,"title":"Buy milk","bogus":true}`,
			wantErr: "bogus",
		},
		{
			name:    "subtask title too long",
			input:   `{"user_id":1,"title":"Shopping","subtasks":[{"title":"` + repeat("x", 501) + `"}]}`,
			wantErr: "subtasks[0].title",
		},
		{
			name:    "subtask description too long",
			input:   `{"user_id":1,"title":"Shopping","subtasks":[{"title":"OK","description":"` + repeat("x", 2001) + `"}]}`,
			wantErr: "subtasks[0].description",
		},
		{
			name:    "subtask invalid priority",
			input:   `{"user_id":1,"title":"Shopping","subtasks":[{"title":"Milk","priority":"super"}]}`,
			wantErr: "subtasks[0].priority",
		},
		{
			name:    "subtask invalid status",
			input:   `{"user_id":1,"title":"Shopping","subtasks":[{"title":"Milk","status":"unknown"}]}`,
			wantErr: "subtasks[0].status",
		},
		{
			name:    "too many subtasks",
			input:   `{"user_id":1,"title":"Big","subtasks":[` + repeatJSON(`{"title":"st"}`, 21) + `]}`,
			wantErr: "subtasks must have 20 items or fewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := addTodoRecord(context.Background(), json.RawMessage(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestAddTodoRecordAcceptsInProgressStatus(t *testing.T) {
	// This test verifies that in_progress is accepted at validation level.
	// It will panic at the DB layer (no DB in unit test), so we recover and check.
	var validationErr error
	func() {
		defer func() { recover() }()
		_, validationErr = addTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"title":"Work","status":"in_progress"}`))
	}()
	if validationErr != nil && contains(validationErr.Error(), "status") {
		t.Fatalf("in_progress should be a valid status, got: %v", validationErr)
	}
}

func TestAddTodoRecordSubtaskTitleDefaultsWhenOmitted(t *testing.T) {
	// Subtask without a title should not produce a validation error.
	// It will panic at the DB layer, so we recover and check.
	var validationErr error
	func() {
		defer func() { recover() }()
		_, validationErr = addTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"title":"Parent","subtasks":[{"description":"no title given"}]}`))
	}()
	if validationErr != nil && contains(validationErr.Error(), "title") {
		t.Fatalf("subtask without title should default, got: %v", validationErr)
	}
}

func TestAddTodoRecordPersistsAndDedupesTags(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	res, err := addTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"title":"Tagged","tags":["work"],"subtasks":[{"title":"Sub","tags":["subtag"]}]}`))
	if err != nil {
		t.Fatalf("addTodoRecord failed: %v", err)
	}
	cr, ok := res.(*todo.CreateResult)
	if !ok {
		t.Fatalf("expected *todo.CreateResult, got %T", res)
	}
	// Explicit tags are persisted along with one browser-server-chat tag.
	if !sameStrings(cr.Tags, []string{"work", "browser-server-chat"}) {
		t.Fatalf("expected tags [work browser-server-chat], got %v", cr.Tags)
	}
	// Subtask tags survive creation.
	if len(cr.Subtasks) != 1 {
		t.Fatalf("expected 1 subtask, got %d", len(cr.Subtasks))
	}
	if !sameStrings(cr.Subtasks[0].Tags, []string{"subtag"}) {
		t.Fatalf("expected subtask tags [subtag], got %v", cr.Subtasks[0].Tags)
	}

	// An explicitly supplied browser-server-chat tag is deduplicated.
	res2, err := addTodoRecord(context.Background(), json.RawMessage(`{"user_id":1,"title":"Chat tagged","tags":["browser-server-chat"]}`))
	if err != nil {
		t.Fatalf("addTodoRecord failed: %v", err)
	}
	cr2 := res2.(*todo.CreateResult)
	if !sameStrings(cr2.Tags, []string{"browser-server-chat"}) {
		t.Fatalf("expected tags [browser-server-chat], got %v", cr2.Tags)
	}
}

// repeatJSON creates n copies of s joined by commas.
func repeatJSON(s string, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = s
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

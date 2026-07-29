package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"browser-server/internal/db"
)

func TestManagePromptRejectsAddDescriptionTooLong(t *testing.T) {
	db.InitPromptDB(t.TempDir())
	t.Cleanup(db.ClosePromptDB)

	_, err := managePrompt(context.Background(), json.RawMessage(`{"user_id":1,"action":"add","title":"Test","content":"Body","description":"`+strings.Repeat("x", 2001)+`"}`))
	if err == nil {
		t.Fatal("expected description length validation error")
	}
	if !contains(err.Error(), "description") {
		t.Fatalf("expected description-related error, got %v", err)
	}
}

package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func TestManagePromptClearsFolderOnEditWithNull(t *testing.T) {
	db.InitPromptDB(t.TempDir())
	t.Cleanup(db.ClosePromptDB)

	result, err := db.PromptDB.Exec("INSERT INTO prompt_folders (user_id, name) VALUES (?, ?)", 1, "General")
	if err != nil {
		t.Fatalf("insert folder: %v", err)
	}
	folderID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	_, err = db.PromptDB.Exec(
		"INSERT INTO prompts (user_id, folder_id, title, content, description, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		1, folderID, "Original", "Body", "", "[]",
	)
	if err != nil {
		t.Fatalf("insert prompt: %v", err)
	}

	res, err := managePrompt(context.Background(), json.RawMessage(fmt.Sprintf(`{"user_id":1,"action":"edit","id":1,"folder_id":null}`)))
	if err != nil {
		t.Fatalf("managePrompt returned error: %v", err)
	}

	payload, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", res)
	}
	if payload["folder_id"] != nil {
		t.Fatalf("expected folder_id to be cleared, got %#v", payload["folder_id"])
	}

	var existingFolderID sql.NullInt64
	err = db.PromptDB.QueryRow("SELECT folder_id FROM prompts WHERE id = 1").Scan(&existingFolderID)
	if err != nil {
		t.Fatalf("scan prompt folder: %v", err)
	}
	if existingFolderID.Valid {
		t.Fatalf("expected prompt folder_id to be NULL after clearing, got %d", existingFolderID.Int64)
	}
}

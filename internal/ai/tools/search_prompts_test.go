package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

func insertTestPrompt(t *testing.T, userID int, title, content string) int64 {
	t.Helper()
	res, err := db.PromptDB.Exec(`INSERT INTO prompts (user_id, title, content, description, tags, created_at, updated_at)
		VALUES (?, ?, ?, '', '[]', ?, ?)`, userID, title, content, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert prompt %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read id for prompt %q: %v", title, err)
	}
	return id
}

func callSearchPrompts(t *testing.T, input string) (map[string]any, error) {
	t.Helper()
	res, err := searchPrompts(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	page, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map envelope, got %T", res)
	}
	return page, nil
}

func TestSearchPromptsFiltering(t *testing.T) {
	dir := t.TempDir()
	db.InitPromptDB(dir)
	t.Cleanup(db.ClosePromptDB)

	insertTestPrompt(t, 1, "Go prompt", "This is about the Go programming language")
	insertTestPrompt(t, 1, "Python prompt", "Python tips and tricks")
	insertTestPrompt(t, 2, "Other prompt", "Go content from another user")

	t.Run("query matches title", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":1,"query":"Go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go prompt" {
			t.Fatalf("expected Go prompt, got %v", rows)
		}
	})

	t.Run("query matches content", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":1,"query":"tricks"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Python prompt" {
			t.Fatalf("expected Python prompt, got %v", rows)
		}
	})

	t.Run("user scoping", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":2,"query":"Go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Other prompt" {
			t.Fatalf("expected Other prompt, got %v", rows)
		}
	})

	t.Run("empty query returns recent first", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":1,"page_size":100}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		if rows[0]["title"] != "Python prompt" {
			t.Fatalf("expected most recent first, got %v", rows[0]["title"])
		}
	})

	t.Run("page metadata", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page"] != 1 || page["page_size"] != 10 || page["total"] != 2 || page["has_more"] != false || page["truncated"] != false {
			t.Fatalf("unexpected metadata: %v", page)
		}
	})

	t.Run("scores present", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":1,"query":"Go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		assertScoresPresent(t, pageResults(page))
	})
}

func TestSearchPromptsPaginationAndLegacyLimit(t *testing.T) {
	dir := t.TempDir()
	db.InitPromptDB(dir)
	t.Cleanup(db.ClosePromptDB)

	for i := 0; i < 15; i++ {
		insertTestPrompt(t, 3, fmt.Sprintf("Prompt %d", i), "content")
	}

	t.Run("legacy limit 10 accepted", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":3,"limit":10}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 10 || page["total"] != 15 || page["has_more"] != true || len(pageResults(page)) != 10 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 100 accepted", func(t *testing.T) {
		page, err := callSearchPrompts(t, `{"user_id":3,"page_size":100}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 100 || page["total"] != 15 || len(pageResults(page)) != 15 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 101 rejected", func(t *testing.T) {
		_, err := callSearchPrompts(t, `{"user_id":3,"page_size":101}`)
		if err == nil || !contains(err.Error(), "page_size must be between 1 and 100") {
			t.Fatalf("expected page_size error, got %v", err)
		}
	})
}

func TestSearchPromptsStrict(t *testing.T) {
	_, err := callSearchPrompts(t, `{"user_id":1,"bogus":true}`)
	if err == nil || !contains(err.Error(), "bogus") {
		t.Fatalf("expected unknown argument error, got %v", err)
	}
}

func init() {
	_ = models.Prompt{}
}

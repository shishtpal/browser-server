package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

func insertTestBookmark(t *testing.T, userID int, title, url, tagsJSON string) int64 {
	t.Helper()
	// Fully parameterized: mixing `?` placeholders with `''` literals in the
	// same VALUES clause makes the sqlite3 driver misbind later parameters.
	res, err := db.BookmarkDB.Exec(`INSERT INTO bookmarks (user_id, title, url, description, tags, folder_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, userID, title, url, "", tagsJSON, "")
	if err != nil {
		t.Fatalf("insert bookmark %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read id for bookmark %q: %v", title, err)
	}
	return id
}

func callSearchBookmarks(t *testing.T, input string) (map[string]any, error) {
	t.Helper()
	res, err := searchBookmarks(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	page, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map envelope, got %T", res)
	}
	return page, nil
}

func TestSearchBookmarksFiltering(t *testing.T) {
	dir := t.TempDir()
	db.InitBookmarkDB(dir)
	t.Cleanup(db.CloseBookmarkDB)

	insertTestBookmark(t, 1, "Go docs", "https://go.dev/doc", `["go","docs"]`)
	insertTestBookmark(t, 1, "Go modules", "https://go.dev/ref/mod", `["go"]`)
	insertTestBookmark(t, 1, "Python docs", "https://python.org/doc", `["python","docs"]`)
	insertTestBookmark(t, 2, "Other user go", "https://go.dev/other", `["go"]`)

	t.Run("query matches title", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1,"query":"modules"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go modules" {
			t.Fatalf("expected Go modules, got %v", rows)
		}
	})

	t.Run("query matches URL", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1,"query":"python.org"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Python docs" {
			t.Fatalf("expected Python docs, got %v", rows)
		}
	})

	t.Run("exact tags use AND semantics", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1,"tags":["go","docs"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go docs" {
			t.Fatalf("expected Go docs, got %v", rows)
		}
	})

	t.Run("user scoping", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":2,"tags":["go"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Other user go" {
			t.Fatalf("expected Other user go, got %v", rows)
		}
	})

	t.Run("empty query returns recent first", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1,"page_size":100}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		if rows[0]["title"] != "Python docs" {
			t.Fatalf("expected most recent first, got %v", rows[0]["title"])
		}
	})

	t.Run("page metadata", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page"] != 1 || page["page_size"] != 10 || page["total"] != 3 || page["has_more"] != false || page["truncated"] != false {
			t.Fatalf("unexpected metadata: %v", page)
		}
	})

	t.Run("scores present", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":1,"query":"go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		assertScoresPresent(t, pageResults(page))
	})
}

func TestSearchBookmarksPaginationAndLegacyLimit(t *testing.T) {
	dir := t.TempDir()
	db.InitBookmarkDB(dir)
	t.Cleanup(db.CloseBookmarkDB)

	for i := 0; i < 25; i++ {
		insertTestBookmark(t, 3, fmt.Sprintf("Bookmark %d", i), "https://example.com", `[]`)
	}

	t.Run("legacy limit 20 accepted", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":3,"limit":20}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 20 || page["total"] != 25 || page["has_more"] != true || len(pageResults(page)) != 20 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 100 accepted", func(t *testing.T) {
		page, err := callSearchBookmarks(t, `{"user_id":3,"page_size":100}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 100 || page["total"] != 25 || len(pageResults(page)) != 25 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 101 rejected", func(t *testing.T) {
		_, err := callSearchBookmarks(t, `{"user_id":3,"page_size":101}`)
		if err == nil || !contains(err.Error(), "page_size must be between 1 and 100") {
			t.Fatalf("expected page_size error, got %v", err)
		}
	})
}

func TestSearchBookmarksStrict(t *testing.T) {
	_, err := callSearchBookmarks(t, `{"user_id":1,"bogus":true}`)
	if err == nil || !contains(err.Error(), "bogus") {
		t.Fatalf("expected unknown argument error, got %v", err)
	}
}

func init() {
	_ = models.Bookmark{}
}

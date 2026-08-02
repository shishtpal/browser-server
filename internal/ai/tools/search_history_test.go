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

func insertTestHistory(t *testing.T, userID int, title, url string) int64 {
	t.Helper()
	res, err := db.HistoryDB.Exec(`INSERT INTO history (user_id, url, domain, title, visited_at, duration)
		VALUES (?, ?, ?, ?, ?, 0)`, userID, url, urlDomain(url), title, time.Now())
	if err != nil {
		t.Fatalf("insert history %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read id for history %q: %v", title, err)
	}
	return id
}

func urlDomain(u string) string {
	if len(u) < 8 {
		return u
	}
	return u[8:]
}

func callSearchHistory(t *testing.T, input string) (map[string]any, error) {
	t.Helper()
	res, err := searchHistory(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	page, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map envelope, got %T", res)
	}
	return page, nil
}

func TestSearchHistoryFiltering(t *testing.T) {
	dir := t.TempDir()
	db.InitHistoryDB(dir)
	t.Cleanup(db.CloseHistoryDB)

	insertTestHistory(t, 1, "Go packages", "https://pkg.go.dev")
	insertTestHistory(t, 1, "Go modules", "https://go.dev/ref/mod")
	insertTestHistory(t, 1, "Python docs", "https://python.org/doc")
	insertTestHistory(t, 2, "Other user go", "https://go.dev/other")

	t.Run("query matches title", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":1,"query":"modules"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go modules" {
			t.Fatalf("expected Go modules, got %v", rows)
		}
	})

	t.Run("query matches URL", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":1,"query":"pkg.go.dev"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go packages" {
			t.Fatalf("expected Go packages, got %v", rows)
		}
	})

	t.Run("domain filter", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":1,"domain":"go.dev/ref/mod"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Go modules" {
			t.Fatalf("expected Go modules, got %v", rows)
		}
	})

	t.Run("user scoping", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":2,"query":"go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		rows := pageResults(page)
		if len(rows) != 1 || rows[0]["title"] != "Other user go" {
			t.Fatalf("expected Other user go, got %v", rows)
		}
	})

	t.Run("empty query returns recent first", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":1,"page_size":100}`)
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
		page, err := callSearchHistory(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page"] != 1 || page["page_size"] != 10 || page["total"] != 3 || page["has_more"] != false || page["truncated"] != false {
			t.Fatalf("unexpected metadata: %v", page)
		}
	})

	t.Run("scores present", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":1,"query":"go"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		assertScoresPresent(t, pageResults(page))
	})
}

func TestSearchHistoryPaginationAndLegacyLimit(t *testing.T) {
	dir := t.TempDir()
	db.InitHistoryDB(dir)
	t.Cleanup(db.CloseHistoryDB)

	for i := 0; i < 60; i++ {
		insertTestHistory(t, 3, fmt.Sprintf("History %d", i), "https://example.com")
	}

	t.Run("legacy limit 50 accepted", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":3,"limit":50}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 50 || page["total"] != 60 || page["has_more"] != true || len(pageResults(page)) != 50 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 100 accepted", func(t *testing.T) {
		page, err := callSearchHistory(t, `{"user_id":3,"page_size":100}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if page["page_size"] != 100 || page["total"] != 60 || len(pageResults(page)) != 60 {
			t.Fatalf("unexpected page: %v", page)
		}
	})

	t.Run("page_size 101 rejected", func(t *testing.T) {
		_, err := callSearchHistory(t, `{"user_id":3,"page_size":101}`)
		if err == nil || !contains(err.Error(), "page_size must be between 1 and 100") {
			t.Fatalf("expected page_size error, got %v", err)
		}
	})
}

func TestSearchHistoryStrict(t *testing.T) {
	_, err := callSearchHistory(t, `{"user_id":1,"bogus":true}`)
	if err == nil || !contains(err.Error(), "bogus") {
		t.Fatalf("expected unknown argument error, got %v", err)
	}
}

func init() {
	_ = models.History{}
}

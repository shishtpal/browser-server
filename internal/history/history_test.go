package history

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

var ctx = context.Background()

func setupDB(t *testing.T) {
	t.Helper()
	db.InitHistoryDB(t.TempDir())
	t.Cleanup(func() { db.HistoryDB.Close() })
}

func insertRow(t *testing.T, userID int, url, title string, visitedAt time.Time, duration int) {
	t.Helper()
	_, err := db.HistoryDB.Exec(
		"INSERT INTO history (user_id, url, domain, title, visited_at, duration) VALUES (?, ?, ?, ?, ?, ?)",
		userID, url, helpers.URLHostname(url), title, visitedAt, duration,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestParseSQLiteTime(t *testing.T) {
	layouts := []string{
		"2026-07-28 15:04:05.999999999-07:00",
		"2026-07-28T15:04:05.999999999-07:00",
		"2026-07-28 15:04:05.999999999",
		"2026-07-28T15:04:05.999999999",
		"2026-07-28 15:04:05",
		"2026-07-28T15:04:05",
		"2026-07-28 15:04",
		"2026-07-28T15:04",
	}
	for _, value := range layouts {
		got := parseSQLiteTime(value)
		if got.IsZero() {
			t.Fatalf("parseSQLiteTime(%q) returned zero time", value)
		}
		if got.Year() != 2026 || got.Month() != time.July || got.Day() != 28 || got.Hour() != 15 {
			t.Fatalf("parseSQLiteTime(%q) = %v, want 2026-07-28T15:04:xx", value, got)
		}
	}

	if got := parseSQLiteTime("2026-07-28"); got.IsZero() {
		t.Fatal("date-only value should parse")
	}

	if got := parseSQLiteTime("not a date"); !got.IsZero() {
		t.Fatal("unparsable value should yield the zero time")
	}
}

func TestNullTime(t *testing.T) {
	if got := nullTime(sql.NullString{}); !got.IsZero() {
		t.Fatalf("invalid NullString should give zero time, got %v", got)
	}
	got := nullTime(sql.NullString{String: "2026-07-28 15:04:05", Valid: true})
	if got.IsZero() {
		t.Fatal("valid NullString should parse")
	}
}

func TestSearchTerms(t *testing.T) {
	clause, args := SearchTerms("foo bar", "title", "url")
	if clause == "" {
		t.Fatal("expected non-empty clause")
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args (2 terms × 2 columns), got %d", len(args))
	}
}

func TestCreateAndGetByID(t *testing.T) {
	setupDB(t)

	entry := models.History{
		UserID:    1,
		URL:       "https://example.com/page",
		Title:     "Example Page",
		VisitedAt: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC),
		Duration:  30,
	}
	id, err := Create(ctx, entry)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.URL != entry.URL {
		t.Fatalf("url = %q, want %q", got.URL, entry.URL)
	}
	if got.Domain != "example.com" {
		t.Fatalf("domain = %q, want %q", got.Domain, "example.com")
	}
}

func TestList(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/a", "A", time.Now(), 10)
	insertRow(t, 1, "https://example.com/b", "B", time.Now(), 20)
	insertRow(t, 2, "https://other.com/c", "C", time.Now(), 30)

	entries, err := List(ctx, ListOptions{UserID: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for user 1, got %d", len(entries))
	}
}

func TestSearch(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/hello", "Hello World", time.Now(), 10)
	insertRow(t, 1, "https://example.com/goodbye", "Goodbye World", time.Now(), 20)

	results, err := Search(ctx, 1, "hello", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'hello', got %d", len(results))
	}
}

func TestDelete(t *testing.T) {
	setupDB(t)

	id, err := Create(ctx, models.History{
		UserID: 1,
		URL:    "https://example.com/delete-me",
		Title:  "Delete Me",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	deleted, err := Delete(ctx, int(id))
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted = true")
	}

	deleted, err = Delete(ctx, 99999)
	if err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted = false for non-existent id")
	}
}

func TestExistingURLs(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/unique", "Unique", time.Now(), 0)

	urls, err := ExistingURLs(ctx, 1)
	if err != nil {
		t.Fatalf("ExistingURLs: %v", err)
	}
	if _, ok := urls["https://example.com/unique"]; !ok {
		t.Fatal("expected unique URL in existing set")
	}
}

func TestListGrouped(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/a", "A", time.Now(), 10)
	insertRow(t, 1, "https://example.com/a", "A again", time.Now(), 20)
	insertRow(t, 2, "https://other.com/c", "C", time.Now(), 30)

	resp, err := ListGrouped(ctx, GroupedOptions{UserID: 1, Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListGrouped: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 grouped entry for user 1, got %d", resp.Total)
	}
}

func TestListDomains(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/a", "A", time.Now(), 10)
	insertRow(t, 1, "https://other.com/b", "B", time.Now(), 20)

	domains, err := ListDomains(ctx, 1, "")
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(domains) < 1 {
		t.Fatal("expected at least 1 domain")
	}
}

func TestOmniboxSearch(t *testing.T) {
	setupDB(t)

	insertRow(t, 1, "https://example.com/test", "Test Page", time.Now(), 10)

	results, err := OmniboxSearch(ctx, 1, "test", 10)
	if err != nil {
		t.Fatalf("OmniboxSearch: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 omnibox result, got %d", len(results))
	}
	if results[0].Source != "history" {
		t.Fatalf("source = %q, want 'history'", results[0].Source)
	}
}

func TestSearchMap(t *testing.T) {
	h := models.History{ID: 1, UserID: 1, URL: "https://example.com", Title: "Example"}
	m := SearchMap(h)
	if m["url"] != "https://example.com" {
		t.Fatalf("SearchMap url = %v", m["url"])
	}
}

func TestSearchMaps(t *testing.T) {
	entries := []models.History{
		{ID: 1, UserID: 1, URL: "https://example.com/a", Title: "A"},
		{ID: 2, UserID: 1, URL: "https://example.com/b", Title: "B"},
	}
	maps := SearchMaps(entries)
	if len(maps) != 2 {
		t.Fatalf("expected 2 maps, got %d", len(maps))
	}
}

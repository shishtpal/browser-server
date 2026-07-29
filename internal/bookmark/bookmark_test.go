package bookmark

import (
	"context"
	"errors"
	"strings"
	"testing"

	"browser-server/internal/db"
)

func setupDB(t *testing.T) {
	t.Helper()
	db.InitBookmarkDB(t.TempDir())
	t.Cleanup(func() { db.BookmarkDB.Close() })
}

func TestCreateAndGetRoundTrip(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, inserted, err := Create(ctx, CreateInput{
		UserID: 1, Title: "Example", URL: "https://example.com",
		Description: "desc", FolderPath: "Work/Refs", Tags: []string{"go", "docs"},
	})
	if err != nil || !inserted {
		t.Fatalf("create: %v (inserted=%v)", err, inserted)
	}

	b, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if b.Title != "Example" || b.URL != "https://example.com" || b.FolderPath != "Work/Refs" {
		t.Fatalf("fields lost: %+v", b)
	}

	resp := Response(b)
	if len(resp.Tags) != 2 || resp.Tags[0] != "go" {
		t.Fatalf("tags = %v", resp.Tags)
	}
}

func TestCreateIsIdempotentForRepeatedCaptures(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, inserted, err := Create(ctx, CreateInput{
		UserID: 1, Title: "Captured", URL: "https://example.com", CaptureID: "cap-1",
	})
	if err != nil || !inserted {
		t.Fatalf("first capture: %v (inserted=%v)", err, inserted)
	}

	_, inserted, err = Create(ctx, CreateInput{
		UserID: 1, Title: "Captured", URL: "https://example.com", CaptureID: "cap-1",
	})
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}
	if inserted {
		t.Fatal("repeated capture should not insert again")
	}

	stored, err := GetByCaptureID(ctx, 1, "cap-1")
	if err != nil {
		t.Fatalf("capture lookup: %v", err)
	}
	if stored.ID != int(id) {
		t.Fatalf("capture lookup id = %d, want %d", stored.ID, id)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, _, _ := Create(ctx, CreateInput{UserID: 1, Title: "Before", URL: "https://a.test"})

	err := Update(ctx, int(id), CreateInput{
		UserID: 1, Title: "After", URL: "https://b.test", Tags: []string{"new"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	b, _ := GetByID(ctx, int(id))
	if b.Title != "After" || b.URL != "https://b.test" {
		t.Fatalf("update did not apply: %+v", b)
	}

	if err := Update(ctx, 9999, CreateInput{UserID: 1, Title: "Missing", URL: "https://missing.test"}); !errors.Is(err, ErrBookmarkNotFound) {
		t.Fatalf("updating a missing bookmark returned %v, want ErrBookmarkNotFound", err)
	}

	deleted, err := Delete(ctx, int(id))
	if err != nil || !deleted {
		t.Fatalf("delete: %v (deleted=%v)", err, deleted)
	}
	if _, err := GetByID(ctx, int(id)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if deleted, _ := Delete(ctx, 9999); deleted {
		t.Fatal("deleting a missing bookmark should report false")
	}
}

func TestSearchMatchesTitleURLAndDescription(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	Create(ctx, CreateInput{UserID: 1, Title: "Go docs", URL: "https://go.dev", Description: "language"})
	Create(ctx, CreateInput{UserID: 1, Title: "Rust book", URL: "https://rust-lang.org", Description: "systems"})
	Create(ctx, CreateInput{UserID: 2, Title: "Go docs", URL: "https://go.dev", Description: "other user"})

	byTitle, err := Search(ctx, 1, "Go docs", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(byTitle) != 1 {
		t.Fatalf("title search returned %d results, want 1 (user scoping)", len(byTitle))
	}

	byURL, _ := Search(ctx, 1, "rust-lang", 10)
	if len(byURL) != 1 || byURL[0].Title != "Rust book" {
		t.Fatalf("url search failed: %+v", byURL)
	}

	byDescription, _ := Search(ctx, 1, "systems", 10)
	if len(byDescription) != 1 {
		t.Fatalf("description search failed: %+v", byDescription)
	}

	limited, _ := Search(ctx, 1, "", 1)
	if len(limited) != 1 {
		t.Fatalf("limit ignored: got %d", len(limited))
	}
}

func TestExistingURLs(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	Create(ctx, CreateInput{UserID: 1, Title: "A", URL: "https://a.test"})
	Create(ctx, CreateInput{UserID: 2, Title: "B", URL: "https://b.test"})

	urls, err := ExistingURLs(ctx, 1)
	if err != nil {
		t.Fatalf("existing URLs: %v", err)
	}
	if _, ok := urls["https://a.test"]; !ok {
		t.Fatal("expected user 1's URL to be present")
	}
	if _, ok := urls["https://b.test"]; ok {
		t.Fatal("another user's URL leaked into the set")
	}
}

func TestListFiltersByUserFolderAndTag(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	Create(ctx, CreateInput{UserID: 1, Title: "Match", URL: "https://match.test", FolderPath: "Work/Go", Tags: []string{"Go"}})
	Create(ctx, CreateInput{UserID: 1, Title: "Wrong folder", URL: "https://folder.test", FolderPath: "Personal", Tags: []string{"go"}})
	Create(ctx, CreateInput{UserID: 2, Title: "Wrong user", URL: "https://user.test", FolderPath: "Work/Go", Tags: []string{"go"}})

	bookmarks, err := List(ctx, ListOptions{UserID: 1, FolderPathPrefix: "Work", TagsFilter: "go"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(bookmarks) != 1 || bookmarks[0].Title != "Match" {
		t.Fatalf("list returned %+v, want only Match", bookmarks)
	}
}

func TestScanAllPropagatesRowsErr(t *testing.T) {
	setupDB(t)
	Create(context.Background(), CreateInput{UserID: 1, Title: "A", URL: "https://a.test"})

	ctx, cancel := context.WithCancel(context.Background())
	rows, err := db.BookmarkDB.QueryContext(ctx, `
		WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < 1000000)
		SELECT `+Columns+` FROM bookmarks CROSS JOIN seq`)
	if err != nil {
		cancel()
		t.Fatalf("query: %v", err)
	}
	if !rows.Next() {
		cancel()
		rows.Close()
		t.Fatal("query returned no first row")
	}
	cancel()

	if _, err := ScanAll(rows); !errors.Is(err, context.Canceled) {
		t.Fatalf("ScanAll error = %v, want context.Canceled", err)
	}
}

func TestValidateTitleURLAndTags(t *testing.T) {
	title, err := ValidateTitle("  title  ")
	if err != nil || title != "title" {
		t.Fatalf("ValidateTitle = %q, %v", title, err)
	}
	if _, err := ValidateTitle(strings.Repeat("x", MaxTitleLength+1)); err == nil {
		t.Fatal("overlong title was accepted")
	}
	if err := ValidateURL(strings.Repeat("x", MaxURLLength+1)); err == nil {
		t.Fatal("overlong URL was accepted")
	}
	if err := ValidateTags([]string{strings.Repeat("x", MaxTagLength+1)}); err == nil {
		t.Fatal("overlong tag was accepted")
	}
	normalized := NormalizeTags([]string{" go ", "", "  ", "docs"})
	if len(normalized) != 2 || normalized[0] != "go" || normalized[1] != "docs" {
		t.Fatalf("NormalizeTags = %v", normalized)
	}
}

func TestMatchesAnyTag(t *testing.T) {
	tags := []string{"Work", "urgent"}

	if !MatchesAnyTag(tags, "") {
		t.Fatal("an empty filter should match everything")
	}
	if !MatchesAnyTag(tags, "work") {
		t.Fatal("matching should be case-insensitive")
	}
	if !MatchesAnyTag(tags, "missing, urgent") {
		t.Fatal("any one matching tag should be enough, and spaces trimmed")
	}
	if MatchesAnyTag(tags, "personal") {
		t.Fatal("non-matching filter should not match")
	}
	if MatchesAnyTag(nil, "work") {
		t.Fatal("a bookmark with no tags should not match a real filter")
	}
}

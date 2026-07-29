package prompt

import (
	"context"
	"strings"
	"testing"

	"browser-server/internal/db"
)

func TestValidation(t *testing.T) {
	got, err := ValidateTitle("  Spaced  ")
	if err != nil || got != "Spaced" {
		t.Fatalf("ValidateTitle = %q, %v", got, err)
	}
	if _, err := ValidateTitle("   "); err == nil {
		t.Fatal("blank title should be rejected")
	}
	if _, err := ValidateTitle(strings.Repeat("x", MaxTitleLength+1)); err == nil {
		t.Fatal("overlong title should be rejected")
	}

	if err := ValidateContent(strings.Repeat("x", MaxContentLength+1)); err == nil {
		t.Fatal("overlong content should be rejected")
	}
	if err := ValidateContent("fine"); err != nil {
		t.Fatalf("valid content rejected: %v", err)
	}

	if err := ValidateDescription(strings.Repeat("x", MaxDescriptionLength+1)); err == nil {
		t.Fatal("overlong description should be rejected")
	}

	if err := ValidateTags([]string{"ok", strings.Repeat("x", MaxTagLength+1)}); err == nil ||
		!strings.Contains(err.Error(), "tags[1]") {
		t.Fatalf("expected tags[1] error, got %v", err)
	}
}

func setupDB(t *testing.T) {
	t.Helper()
	db.InitPromptDB(t.TempDir())
	t.Cleanup(db.ClosePromptDB)
}

func TestCreateAndGetRoundTrip(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, tagsJSON, err := Create(ctx, CreateInput{
		UserID: 1, Title: "Greeting", Content: "Say hi", Description: "desc",
		Tags: []string{"chat", "intro"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tagsJSON != `["chat","intro"]` {
		t.Fatalf("tags JSON = %s", tagsJSON)
	}

	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec.Prompt.Title != "Greeting" || rec.Prompt.Content != "Say hi" {
		t.Fatalf("fields lost: %+v", rec.Prompt)
	}
	if tags := rec.Tags(); len(tags) != 2 || tags[0] != "chat" {
		t.Fatalf("tags = %v", tags)
	}
}

func TestMap(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, _, err := Create(ctx, CreateInput{UserID: 1, Title: "T", Content: "C"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	m := Map(rec)
	if _, ok := m["folder_id"]; ok {
		t.Fatal("Map should not contain folder_id")
	}
}

func TestListFiltersAndOrders(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	Create(ctx, CreateInput{UserID: 1, Title: "Alpha guide", Content: "about bananas"})
	Create(ctx, CreateInput{UserID: 1, Title: "Beta notes", Content: "about apples"})
	Create(ctx, CreateInput{UserID: 2, Title: "Other user", Content: "hidden"})

	// Listing is scoped to one user.
	all, err := List(ctx, ListQuery{UserID: 1})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 prompts for user 1, got %d", len(all))
	}

	// The query matches both title and content.
	byContent, _ := List(ctx, ListQuery{UserID: 1, Query: "bananas"})
	if len(byContent) != 1 || byContent[0].Prompt.Title != "Alpha guide" {
		t.Fatalf("content search failed: %+v", byContent)
	}

	// TitleFirst ranks title hits ahead of content-only hits.
	ranked, _ := List(ctx, ListQuery{UserID: 1, Query: "Beta", TitleFirst: true})
	if len(ranked) == 0 || ranked[0].Prompt.Title != "Beta notes" {
		t.Fatalf("title-first ordering failed: %+v", ranked)
	}

	limited, _ := List(ctx, ListQuery{UserID: 1, Limit: 1})
	if len(limited) != 1 {
		t.Fatalf("limit ignored: got %d", len(limited))
	}
}

func TestListReturnsScanErrors(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	_, err := db.PromptDB.ExecContext(ctx, `
		INSERT INTO prompts (user_id, title, content, description, tags)
		VALUES (?, ?, ?, NULL, ?)`, 1, "Invalid", "Body", "[]")
	if err != nil {
		t.Fatalf("insert malformed prompt: %v", err)
	}

	if _, err := List(ctx, ListQuery{UserID: 1}); err == nil {
		t.Fatal("expected malformed prompt row to return a scan error")
	}
}

func TestOwnershipErrors(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, _, _ := Create(ctx, CreateInput{UserID: 1, Title: "Mine", Content: "x"})

	if err := VerifyOwnership(ctx, int(id), 1); err != nil {
		t.Fatalf("owner should pass: %v", err)
	}
	if err := VerifyOwnership(ctx, int(id), 2); err != ErrPromptNotOwned {
		t.Fatalf("expected ErrPromptNotOwned, got %v", err)
	}
	if err := VerifyOwnership(ctx, 9999, 1); err != ErrPromptNotFound {
		t.Fatalf("expected ErrPromptNotFound, got %v", err)
	}
}

func TestUpdateBuilderPartialUpdate(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, _, _ := Create(ctx, CreateInput{UserID: 1, Title: "Before", Content: "keep", Description: "keep too"})

	if err := NewUpdateBuilder().Set("title", "After").Exec(ctx, int(id)); err != nil {
		t.Fatalf("update: %v", err)
	}

	rec, _ := GetByID(ctx, int(id))
	if rec.Prompt.Title != "After" {
		t.Fatalf("title not updated: %q", rec.Prompt.Title)
	}
	if rec.Prompt.Content != "keep" || rec.Prompt.Description != "keep too" {
		t.Fatalf("untouched fields changed: %+v", rec.Prompt)
	}

	if err := NewUpdateBuilder().Exec(ctx, int(id)); err == nil {
		t.Fatal("empty update should error")
	}
}

func TestDeletePrompt(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, _, _ := Create(ctx, CreateInput{UserID: 1, Title: "Doomed", Content: "x"})

	deleted, err := Delete(ctx, int(id))
	if err != nil || !deleted {
		t.Fatalf("delete: %v (deleted=%v)", err, deleted)
	}
	if _, err := GetByID(ctx, int(id)); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if deleted, _ := Delete(ctx, 9999); deleted {
		t.Fatal("deleting a missing prompt should report false")
	}
}

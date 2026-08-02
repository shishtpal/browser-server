package todo

import (
	"context"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
)

func TestSearchCandidatesUserAndTag(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	insert := func(userID int, title, tagsJSON, startDate string) {
		t.Helper()
		var sd any
		if startDate != "" {
			sd = startDate
		}
		_, err := db.TodoDB.Exec(`INSERT INTO todos (user_id, title, description, status, priority, tags, start_date, updated_at)
			VALUES (?, ?, '', 'pending', 'medium', ?, ?, CURRENT_TIMESTAMP)`,
			userID, title, tagsJSON, sd)
		if err != nil {
			t.Fatalf("insert %q: %v", title, err)
		}
	}
	insert(1, "Work task", `["work"]`, "")
	insert(1, "Urgent work", `["work","urgent"]`, "")
	insert(2, "Other work", `["work"]`, "")
	insert(1, "Meeting", `[]`, "2026-08-01")

	set, err := SearchCandidates(context.Background(), SearchFilter{UserID: 1, Tags: []string{"work"}}, searchengine.CandidateRequest{MaxCandidates: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 2 {
		t.Fatalf("expected 2 work candidates for user 1, got %d", len(set.Candidates))
	}
	if set.Truncated {
		t.Fatal("expected not truncated")
	}

	set2, err := SearchCandidates(context.Background(), SearchFilter{UserID: 1, Scheduled: true}, searchengine.CandidateRequest{MaxCandidates: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(set2.Candidates) != 1 || set2.Candidates[0].Value.Title != "Meeting" {
		t.Fatalf("expected 1 scheduled candidate, got %+v", set2.Candidates)
	}
	if set2.Candidates[0].Fields[0].Weight != 10 {
		t.Fatalf("expected title weight 10, got %f", set2.Candidates[0].Fields[0].Weight)
	}
}

func TestTodoSearchHitMapTags(t *testing.T) {
	m := TodoSearchHitMap(models.Todo{ID: 1, Title: "T", Status: "pending", Priority: "medium"}, []string{"work"}, 0.5)
	if m["score"] != 0.5 {
		t.Fatalf("expected score 0.5, got %v", m["score"])
	}
	if _, ok := m["start_date"]; ok {
		t.Fatal("expected start_date omitted")
	}
}

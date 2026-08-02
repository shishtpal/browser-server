package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/db"
)

// sameStrings compares two string slices element-wise.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func callSearchCalendar(t *testing.T, input string) ([]map[string]any, error) {
	t.Helper()
	res, err := searchCalendar(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	rows, ok := res.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", res)
	}
	return rows, nil
}

func TestSearchCalendarTagFiltering(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	insertTestTodo(t, 1, "Meeting", `["work"]`, "pending", "medium", "2026-08-01")
	insertTestTodo(t, 1, "Standup", `["work","urgent"]`, "pending", "urgent", "2026-08-02")
	insertTestTodo(t, 1, "Retro", `["work"]`, "done", "medium", "2026-08-03")
	insertTestTodo(t, 1, "No tag event", `[]`, "pending", "medium", "2026-08-04")
	insertTestTodo(t, 1, "Unscheduled work", `["work"]`, "pending", "medium", "")
	insertTestTodo(t, 1, "Social", `["homework"]`, "pending", "medium", "2026-08-06")
	insertTestTodo(t, 2, "Other user meeting", `["work"]`, "pending", "medium", "2026-08-05")

	t.Run("single exact tag", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1,"tags":["work"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		for _, want := range []string{"Meeting", "Standup", "Retro"} {
			if !rowsHaveTitle(rows, want) {
				t.Errorf("missing expected row %q", want)
			}
		}
		// work must not match homework; unscheduled and other-user rows excluded.
		for _, not := range []string{"No tag event", "Unscheduled work", "Social", "Other user meeting"} {
			if rowsHaveTitle(rows, not) {
				t.Errorf("unexpected row %q matched", not)
			}
		}
	})

	t.Run("multiple tags use AND semantics", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1,"tags":["work","urgent"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Standup" {
			t.Fatalf("expected only Standup, got %v", rows)
		}
	})

	t.Run("unscheduled tagged todos are excluded even without tag filter", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if rowsHaveTitle(rows, "Unscheduled work") {
			t.Fatal("unscheduled todo leaked into calendar results")
		}
		if len(rows) != 5 {
			t.Fatalf("expected 5 scheduled rows, got %d", len(rows))
		}
	})

	t.Run("tag filter composes with date range", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1,"tags":["work"],"from":"2026-08-02","to":"2026-08-02"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Standup" {
			t.Fatalf("expected only Standup, got %v", rows)
		}
	})

	t.Run("tag filter composes with status", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1,"tags":["work"],"status":"done"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Retro" {
			t.Fatalf("expected only Retro, got %v", rows)
		}
	})

	t.Run("tag filter composes with text query", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1,"tags":["work"],"query":"standup"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Standup" {
			t.Fatalf("expected only Standup, got %v", rows)
		}
	})

	t.Run("results are scoped to the requested user", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":2,"tags":["work"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Other user meeting" {
			t.Fatalf("expected only Other user meeting, got %v", rows)
		}
	})

	t.Run("every result includes parsed tags with empty array for untagged", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		for _, r := range rows {
			if _, ok := r["tags"].([]string); !ok {
				t.Errorf("row %v has no []string tags field", r["title"])
			}
		}
		noTag := findRow(rows, "No tag event")
		if noTag == nil || len(rowsTags(*noTag)) != 0 {
			t.Fatalf("expected No tag event to have empty tags, got %v", rowTagsPtr(noTag))
		}
		standup := findRow(rows, "Standup")
		if standup == nil || !sameStrings(rowsTags(*standup), []string{"work", "urgent"}) {
			t.Fatalf("expected Standup tags [work urgent], got %v", rowTagsPtr(standup))
		}
	})
}

func TestSearchCalendarLimitValidation(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	for i := 0; i < 25; i++ {
		insertTestTodo(t, 4, "Event", `["bulk"]`, "pending", "medium", "2026-08-01")
	}

	t.Run("limit 50 accepted and returns more than 20", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":4,"limit":50}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 25 {
			t.Fatalf("expected 25 rows, got %d", len(rows))
		}
	})

	t.Run("limit 51 rejected", func(t *testing.T) {
		_, err := callSearchCalendar(t, `{"user_id":4,"limit":51}`)
		if err == nil || !contains(err.Error(), "limit must be 1 to 50") {
			t.Fatalf("expected limit error, got %v", err)
		}
	})

	t.Run("limit 0 defaults to 10", func(t *testing.T) {
		rows, err := callSearchCalendar(t, `{"user_id":4}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 10 {
			t.Fatalf("expected default limit 10, got %d", len(rows))
		}
	})

	t.Run("negative limit rejected", func(t *testing.T) {
		_, err := callSearchCalendar(t, `{"user_id":4,"limit":-1}`)
		if err == nil || !contains(err.Error(), "limit must be 1 to 50") {
			t.Fatalf("expected limit error, got %v", err)
		}
	})
}

func TestSearchCalendarTagValidationAndStrict(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	t.Run("overlong tag rejected before query", func(t *testing.T) {
		_, err := callSearchCalendar(t, `{"user_id":1,"tags":["`+repeat("x", 101)+`"]}`)
		if err == nil || !contains(err.Error(), "tags[0] must be 100 characters or fewer") {
			t.Fatalf("expected overlong tag error, got %v", err)
		}
	})

	t.Run("unknown argument rejected", func(t *testing.T) {
		_, err := callSearchCalendar(t, `{"user_id":1,"bogus":true}`)
		if err == nil || !contains(err.Error(), "bogus") {
			t.Fatalf("expected unknown argument error, got %v", err)
		}
	})
}

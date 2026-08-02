package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/db"
)

// insertTestTodo inserts a todo row directly so tests control tags exactly;
// the add_todo_record tool always appends the browser-server-chat tag.
// startDate may be "" for an unscheduled todo.
func insertTestTodo(t *testing.T, userID int, title, tagsJSON, status, priority, startDate string) int64 {
	t.Helper()
	var sd any
	if startDate != "" {
		sd = startDate
	}
	res, err := db.TodoDB.Exec(
		`INSERT INTO todos (user_id, title, description, status, priority, tags, start_date, updated_at)
		 VALUES (?, ?, '', ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		userID, title, status, priority, tagsJSON, sd,
	)
	if err != nil {
		t.Fatalf("failed to insert todo %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read insert id for %q: %v", title, err)
	}
	return id
}

func callSearchTodos(t *testing.T, input string) ([]map[string]any, error) {
	t.Helper()
	res, err := searchTodos(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	rows, ok := res.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", res)
	}
	return rows, nil
}

func rowsHaveTitle(rows []map[string]any, title string) bool {
	for _, r := range rows {
		if r["title"] == title {
			return true
		}
	}
	return false
}

func rowsTags(row map[string]any) []string {
	tags, ok := row["tags"].([]string)
	if !ok {
		return nil
	}
	return tags
}

func TestSearchTodosTagFiltering(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	insertTestTodo(t, 1, "Work task", `["work"]`, "pending", "high", "")
	insertTestTodo(t, 1, "Urgent work", `["work","urgent"]`, "pending", "urgent", "")
	insertTestTodo(t, 1, "Homework", `["homework"]`, "pending", "medium", "")
	insertTestTodo(t, 1, "Plain", `[]`, "pending", "medium", "")
	insertTestTodo(t, 1, "Done work", `["work"]`, "done", "medium", "")
	insertTestTodo(t, 2, "Other user work", `["work"]`, "pending", "medium", "")

	t.Run("single exact tag", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"tags":["work"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		for _, want := range []string{"Work task", "Urgent work", "Done work"} {
			if !rowsHaveTitle(rows, want) {
				t.Errorf("missing expected row %q", want)
			}
		}
		// work must not match homework, untagged rows, or another user's todo.
		for _, not := range []string{"Homework", "Plain", "Other user work"} {
			if rowsHaveTitle(rows, not) {
				t.Errorf("unexpected row %q matched", not)
			}
		}
	})

	t.Run("multiple tags use AND semantics", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"tags":["work","urgent"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Urgent work" {
			t.Fatalf("expected only Urgent work, got %v", rows)
		}
	})

	t.Run("tag filter composes with status", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"tags":["work"],"status":"done"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Done work" {
			t.Fatalf("expected only Done work, got %v", rows)
		}
	})

	t.Run("tag filter composes with priority", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"tags":["work"],"priority":"high"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Work task" {
			t.Fatalf("expected only Work task, got %v", rows)
		}
	})

	t.Run("tag filter composes with text query", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"tags":["work"],"query":"urgent"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Urgent work" {
			t.Fatalf("expected only Urgent work, got %v", rows)
		}
	})

	t.Run("results are scoped to the requested user", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":2,"tags":["work"]}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 1 || rows[0]["title"] != "Other user work" {
			t.Fatalf("expected only Other user work, got %v", rows)
		}
	})

	t.Run("every result includes parsed tags with empty array for untagged", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 5 {
			t.Fatalf("expected 5 rows, got %d", len(rows))
		}
		for _, r := range rows {
			if _, ok := r["tags"].([]string); !ok {
				t.Errorf("row %v has no []string tags field", r["title"])
			}
		}
		plain := findRow(rows, "Plain")
		if plain == nil || len(rowsTags(*plain)) != 0 {
			t.Fatalf("expected Plain to have empty tags, got %v", rowTagsPtr(plain))
		}
		work := findRow(rows, "Work task")
		if work == nil || !sameStrings(rowsTags(*work), []string{"work"}) {
			t.Fatalf("expected Work task tags [work], got %v", rowTagsPtr(work))
		}
	})

	t.Run("omitting tags preserves existing search behavior", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":1,"status":"pending"}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		// user 1 has 4 pending todos and 1 done todo.
		if len(rows) != 4 {
			t.Fatalf("expected 4 pending rows, got %d", len(rows))
		}
	})
}

func findRow(rows []map[string]any, title string) *map[string]any {
	for i := range rows {
		if rows[i]["title"] == title {
			return &rows[i]
		}
	}
	return nil
}

func rowTagsPtr(row *map[string]any) any {
	if row == nil {
		return "<missing>"
	}
	return rowsTags(*row)
}

func TestSearchTodosLimitValidation(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	// Seed 25 todos for user 3 so limit 50 can return more than 20 rows.
	for i := 0; i < 25; i++ {
		insertTestTodo(t, 3, "Bulk", `["bulk"]`, "pending", "medium", "")
	}
	insertTestTodo(t, 3, "Zero default check", `[]`, "pending", "medium", "")

	t.Run("limit 50 accepted and returns more than 20", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":3,"limit":50}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 26 {
			t.Fatalf("expected 26 rows, got %d", len(rows))
		}
	})

	t.Run("limit 51 rejected", func(t *testing.T) {
		_, err := callSearchTodos(t, `{"user_id":3,"limit":51}`)
		if err == nil || !contains(err.Error(), "limit must be 1 to 50") {
			t.Fatalf("expected limit error, got %v", err)
		}
	})

	t.Run("limit 0 defaults to 10", func(t *testing.T) {
		rows, err := callSearchTodos(t, `{"user_id":3}`)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(rows) != 10 {
			t.Fatalf("expected default limit 10, got %d", len(rows))
		}
	})

	t.Run("negative limit rejected", func(t *testing.T) {
		_, err := callSearchTodos(t, `{"user_id":3,"limit":-1}`)
		if err == nil || !contains(err.Error(), "limit must be 1 to 50") {
			t.Fatalf("expected limit error, got %v", err)
		}
	})
}

func TestSearchTodosTagValidationAndStrict(t *testing.T) {
	dir := t.TempDir()
	db.InitTodoDB(dir)
	t.Cleanup(db.CloseTodoDB)

	t.Run("overlong tag rejected before query", func(t *testing.T) {
		_, err := callSearchTodos(t, `{"user_id":1,"tags":["`+repeat("x", 101)+`"]}`)
		if err == nil || !contains(err.Error(), "tags[0] must be 100 characters or fewer") {
			t.Fatalf("expected overlong tag error, got %v", err)
		}
	})

	t.Run("unknown argument rejected", func(t *testing.T) {
		_, err := callSearchTodos(t, `{"user_id":1,"bogus":true}`)
		if err == nil || !contains(err.Error(), "bogus") {
			t.Fatalf("expected unknown argument error, got %v", err)
		}
	})
}

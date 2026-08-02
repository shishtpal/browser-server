package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/todo"
)

//go:embed schemas/search_todos.json
var searchTodosSchema []byte

func registerSearchTodos(r *Registry) {
	r.add(Tool{
		Name:        "search_todos",
		Category:    "General",
		Description: "Search the local todo database. Can filter by status, priority, text query across title/description, and exact tags. Results include each todo's tags. When multiple tags are given, every tag must be present on a returned todo (AND semantics).",
		Schema:      json.RawMessage(searchTodosSchema),
		Execute:     searchTodos,
	})
}

func searchTodos(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID   int      `json:"user_id"`
		Query    string   `json:"query"`
		Status   string   `json:"status"`
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
		Limit    int      `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "status": true, "priority": true, "tags": true, "limit": true}); err != nil {
		return nil, err
	}
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required")
	}
	a.Query = strings.TrimSpace(a.Query)
	if len(a.Query) > 200 {
		return nil, fmt.Errorf("query too long")
	}
	if a.Limit == 0 {
		a.Limit = 10
	}
	if a.Limit < 1 || a.Limit > 50 {
		return nil, fmt.Errorf("limit must be 1 to 50")
	}
	if a.Priority != "" && !todo.IsValidPriority(a.Priority) {
		return nil, fmt.Errorf("priority must be one of: low, medium, high, urgent")
	}
	if a.Status != "" && !todo.IsValidStatus(a.Status) {
		return nil, fmt.Errorf("status must be one of: pending, in_progress, completed, done, cancelled, archived")
	}
	for i, tag := range a.Tags {
		if len(tag) > 100 {
			return nil, fmt.Errorf("tags[%d] must be 100 characters or fewer", i)
		}
	}

	// Build dynamic query
	where := []string{"user_id = ?"}
	args := []any{a.UserID}

	if a.Query != "" {
		where = append(where, "(title LIKE ? OR description LIKE ?)")
		args = append(args, "%"+a.Query+"%", "%"+a.Query+"%")
	}
	if a.Status != "" {
		where = append(where, "status = ?")
		args = append(args, a.Status)
	}
	if a.Priority != "" {
		where = append(where, "priority = ?")
		args = append(args, a.Priority)
	}
	for _, tag := range a.Tags {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(todos.tags) WHERE json_each.value = ?)")
		args = append(args, tag)
	}

	args = append(args, a.Limit)
	q := fmt.Sprintf(
		`SELECT id, title, description, status, priority, pinned, start_date, end_date, tags FROM todos WHERE %s ORDER BY updated_at DESC LIMIT ?`,
		strings.Join(where, " AND "),
	)

	rows, err := db.TodoDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id int
		var title, description, status, priority, tagsJSON string
		var pinned bool
		var startDate, endDate *string
		if err := rows.Scan(&id, &title, &description, &status, &priority, &pinned, &startDate, &endDate, &tagsJSON); err != nil {
			return nil, err
		}
		tags := helpers.ParseTagsFromJSON(tagsJSON)
		if tags == nil {
			tags = []string{}
		}
		entry := map[string]any{
			"id":          id,
			"title":       title,
			"description": description,
			"status":      status,
			"priority":    priority,
			"pinned":      pinned,
			"tags":        tags,
		}
		if startDate != nil {
			entry["start_date"] = *startDate
		}
		if endDate != nil {
			entry["end_date"] = *endDate
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

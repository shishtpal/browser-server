package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
)

func registerSearchTodos(r *Registry) {
	r.add(Tool{
		Name:        "search_todos",
		Category:    "General",
		Description: "Search the local todo database. Can filter by status, priority, and text query across title/description.",
		Schema: json.RawMessage(`{"type":"object","properties":{` +
			`"user_id":{"type":"integer","minimum":1},` +
			`"query":{"type":"string","maxLength":200},` +
			`"status":{"type":"string","enum":["pending","in_progress","done","cancelled"]},` +
			`"priority":{"type":"string","enum":["low","medium","high","urgent"]},` +
			`"limit":{"type":"integer","minimum":1,"maximum":20}` +
			`},"required":["user_id"],"additionalProperties":false}`),
		Execute: searchTodos,
	})
}

func searchTodos(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID   int    `json:"user_id"`
		Query    string `json:"query"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
		Limit    int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "status": true, "priority": true, "limit": true}); err != nil {
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
	if a.Limit < 1 || a.Limit > 20 {
		return nil, fmt.Errorf("limit must be 1 to 20")
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

	args = append(args, a.Limit)
	q := fmt.Sprintf(
		`SELECT id, title, description, status, priority, pinned, start_date, end_date FROM todos WHERE %s ORDER BY updated_at DESC LIMIT ?`,
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
		var title, description, status, priority string
		var pinned bool
		var startDate, endDate *string
		if err := rows.Scan(&id, &title, &description, &status, &priority, &pinned, &startDate, &endDate); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"id":          id,
			"title":       title,
			"description": description,
			"status":      status,
			"priority":    priority,
			"pinned":      pinned,
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

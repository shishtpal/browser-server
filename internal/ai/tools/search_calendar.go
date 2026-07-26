package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
)

func registerSearchCalendar(r *Registry) {
	r.add(Tool{
		Name:        "search_calendar",
		Category:    "General",
		Description: "Search calendar events (todos with scheduled dates). Can filter by date range, text query, and status. Returns events that have a start_date set.",
		Schema: json.RawMessage(`{"type":"object","properties":{` +
			`"user_id":{"type":"integer","minimum":1},` +
			`"query":{"type":"string","maxLength":200},` +
			`"from":{"type":"string","description":"Start of date range (YYYY-MM-DD)"},` +
			`"to":{"type":"string","description":"End of date range (YYYY-MM-DD)"},` +
			`"status":{"type":"string","enum":["pending","in_progress","done","cancelled"]},` +
			`"limit":{"type":"integer","minimum":1,"maximum":20}` +
			`},"required":["user_id"],"additionalProperties":false}`),
		Execute: searchCalendar,
	})
}

func searchCalendar(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		From   string `json:"from"`
		To     string `json:"to"`
		Status string `json:"status"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "from": true, "to": true, "status": true, "limit": true}); err != nil {
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

	// Only return items that have a start_date (calendar events)
	where := []string{"user_id = ?", "start_date IS NOT NULL"}
	args := []any{a.UserID}

	if a.Query != "" {
		where = append(where, "(title LIKE ? OR description LIKE ?)")
		args = append(args, "%"+a.Query+"%", "%"+a.Query+"%")
	}
	if a.From != "" {
		where = append(where, "start_date >= ?")
		args = append(args, a.From)
	}
	if a.To != "" {
		where = append(where, "(end_date <= ? OR (end_date IS NULL AND start_date <= ?))")
		args = append(args, a.To, a.To)
	}
	if a.Status != "" {
		where = append(where, "status = ?")
		args = append(args, a.Status)
	}

	args = append(args, a.Limit)
	q := fmt.Sprintf(
		`SELECT id, title, description, status, priority, start_date, end_date, rrule FROM todos WHERE %s ORDER BY start_date ASC LIMIT ?`,
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
		var startDate, endDate, rrule *string
		if err := rows.Scan(&id, &title, &description, &status, &priority, &startDate, &endDate, &rrule); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"id":          id,
			"title":       title,
			"description": description,
			"status":      status,
			"priority":    priority,
		}
		if startDate != nil {
			entry["start_date"] = *startDate
		}
		if endDate != nil {
			entry["end_date"] = *endDate
		}
		if rrule != nil && *rrule != "" {
			entry["rrule"] = *rrule
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

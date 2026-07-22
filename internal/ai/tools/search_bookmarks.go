package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
)

func registerSearchBookmarks(r *Registry) {
	r.add(Tool{
		Name:        "search_bookmarks",
		Category:    "General",
		Description: "Search the local bookmark database",
		Schema:      json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"integer","minimum":1},"query":{"type":"string","maxLength":200},"limit":{"type":"integer","minimum":1,"maximum":20}},"required":["user_id","query"],"additionalProperties":false}`),
		Execute:     searchBookmarks,
	})
}

func searchBookmarks(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "limit": true}); err != nil {
		return nil, err
	}
	a.Query = strings.TrimSpace(a.Query)
	if a.UserID < 1 || a.Query == "" || len(a.Query) > 200 {
		return nil, fmt.Errorf("invalid bookmark search arguments")
	}
	if a.Limit == 0 {
		a.Limit = 10
	}
	if a.Limit < 1 || a.Limit > 20 {
		return nil, fmt.Errorf("limit must be 1 to 20")
	}
	rows, err := db.BookmarkDB.QueryContext(ctx,
		`SELECT id,title,url FROM bookmarks WHERE user_id=? AND (title LIKE ? OR url LIKE ? OR description LIKE ?) ORDER BY updated_at DESC LIMIT ?`,
		a.UserID, "%"+a.Query+"%", "%"+a.Query+"%", "%"+a.Query+"%", a.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int
		var title, url string
		if err := rows.Scan(&id, &title, &url); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "url": url})
	}
	return out, rows.Err()
}

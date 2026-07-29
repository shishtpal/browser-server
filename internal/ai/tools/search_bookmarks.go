package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/bookmark"
)

//go:embed schemas/search_bookmarks.json
var searchBookmarksSchema []byte

func registerSearchBookmarks(r *Registry) {
	r.add(Tool{
		Name:        "search_bookmarks",
		Category:    "General",
		Description: "Search the local bookmark database. Can filter by text query across title, URL, and description. When no query is given, returns the most recently updated bookmarks (up to the limit).",
		Schema:      json.RawMessage(searchBookmarksSchema),
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
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}
	a.Query = strings.TrimSpace(a.Query)
	if len(a.Query) > 200 {
		return nil, fmt.Errorf("query must be 200 characters or fewer")
	}
	if a.Limit == 0 {
		a.Limit = 10
	}
	if a.Limit < 1 || a.Limit > 20 {
		return nil, fmt.Errorf("limit must be between 1 and 20")
	}

	bookmarks, err := bookmark.Search(ctx, a.UserID, a.Query, a.Limit)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(bookmarks))
	for _, b := range bookmarks {
		out = append(out, bookmark.SearchMap(b))
	}
	return out, nil
}

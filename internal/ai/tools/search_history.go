package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/history"
)

//go:embed schemas/search_history.json
var searchHistorySchema []byte

func registerSearchHistory(r *Registry) {
	r.add(Tool{
		Name:        "search_history",
		Category:    "General",
		Description: "Search the local browsing history database. Can filter by text query across URL/title and optionally by domain. When no query or domain is given, returns the most recently visited records (up to the page limit).",
		Schema:      json.RawMessage(searchHistorySchema),
		Execute:     searchHistory,
	})
}

func searchHistory(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Domain string `json:"domain"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "domain": true, "limit": true}); err != nil {
		return nil, err
	}
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required")
	}
	a.Query = strings.TrimSpace(a.Query)
	a.Domain = strings.TrimSpace(a.Domain)
	if len(a.Query) > 200 {
		return nil, fmt.Errorf("query too long")
	}
	if a.Limit == 0 {
		a.Limit = 50
	}
	if a.Limit < 1 || a.Limit > 50 {
		return nil, fmt.Errorf("limit must be 1 to 50")
	}

	entries, err := history.Search(ctx, a.UserID, a.Query, a.Domain, a.Limit)
	if err != nil {
		return nil, err
	}
	return history.SearchMaps(entries), nil
}

package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/prompt"
)

//go:embed schemas/search_prompts.json
var searchPromptsSchema []byte

func registerSearchPrompts(r *Registry) {
	r.add(Tool{
		Name:        "search_prompts",
		Category:    "General",
		Description: "Search the local prompt database. Can filter by user and text query across title/content.",
		Schema:      json.RawMessage(searchPromptsSchema),
		Execute:     searchPrompts,
	})
}

func searchPrompts(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "limit": true}); err != nil {
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
		a.Limit = 5
	}
	if a.Limit < 1 || a.Limit > 10 {
		return nil, fmt.Errorf("limit must be 1 to 10")
	}

	records, err := prompt.List(ctx, prompt.ListQuery{
		UserID:     a.UserID,
		Query:      a.Query,
		Limit:      a.Limit,
		TitleFirst: true,
	})
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, prompt.SearchMap(rec))
	}
	return out, nil
}

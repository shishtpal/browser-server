package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/prompt"
	"browser-server/internal/searchengine"
)

//go:embed schemas/search_prompts.json
var searchPromptsSchema []byte

func registerSearchPrompts(r *Registry) {
	r.add(Tool{
		Name:        "search_prompts",
		Category:    "General",
		Description: "Search the local prompt database. Can filter by user and text query across title/content. Results include a relevance score. Supports one-based pagination (page, page_size). The legacy `limit` argument is deprecated and maps to `page_size` when `page_size` is omitted.",
		Schema:      json.RawMessage(searchPromptsSchema),
		Execute:     searchPrompts,
	})
}

func searchPrompts(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID   int    `json:"user_id"`
		Query    string `json:"query"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
		Limit    int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "page": true, "page_size": true, "limit": true}); err != nil {
		return nil, err
	}
	if err := validateSearchPagination(raw, "limit"); err != nil {
		return nil, err
	}
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required")
	}
	a.Query = strings.TrimSpace(a.Query)
	if len(a.Query) > 200 {
		return nil, fmt.Errorf("query too long")
	}

	pageSize, err := resolvePageSize(a.PageSize, a.Limit, 10, 10)
	if err != nil {
		return nil, err
	}

	loader := func(ctx context.Context, req searchengine.CandidateRequest) (searchengine.CandidateSet[prompt.Record], error) {
		return prompt.SearchCandidates(ctx, a.UserID, req.MaxCandidates)
	}

	page, err := searchengine.Search(ctx, searchengine.Request{Query: a.Query, Page: a.Page, PageSize: pageSize}, loader)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(page.Results))
	for _, hit := range page.Results {
		out = append(out, prompt.SearchHitMap(hit.Value, hit.Score))
	}
	return fitSearchEnvelope(ctx, map[string]any{
		"query":     a.Query,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     page.Total,
		"has_more":  page.HasMore,
		"truncated": page.Truncated,
		"results":   out,
	}), nil
}

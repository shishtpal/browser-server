package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/history"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
)

//go:embed schemas/search_history.json
var searchHistorySchema []byte

func registerSearchHistory(r *Registry) {
	r.add(Tool{
		Name:        "search_history",
		Category:    "General",
		Description: "Search the local browsing history database. Can filter by text query across URL/title and optionally by domain. Results include a relevance score. Supports one-based pagination (page, page_size). When no query or domain is given, returns the most recently visited records (up to the page size). The legacy `limit` argument is deprecated and maps to `page_size` when `page_size` is omitted.",
		Schema:      json.RawMessage(searchHistorySchema),
		Execute:     searchHistory,
	})
}

func searchHistory(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID   int    `json:"user_id"`
		Query    string `json:"query"`
		Domain   string `json:"domain"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
		Limit    int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "domain": true, "page": true, "page_size": true, "limit": true}); err != nil {
		return nil, err
	}
	if err := validateSearchPagination(raw, "limit"); err != nil {
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
	if len(a.Domain) > 500 {
		return nil, fmt.Errorf("domain too long")
	}

	pageSize, err := resolvePageSize(a.PageSize, a.Limit, 50, 10)
	if err != nil {
		return nil, err
	}

	loader := func(ctx context.Context, req searchengine.CandidateRequest) (searchengine.CandidateSet[models.History], error) {
		return history.SearchCandidates(ctx, a.UserID, a.Domain, req.MaxCandidates)
	}

	page, err := searchengine.Search(ctx, searchengine.Request{Query: a.Query, Page: a.Page, PageSize: pageSize}, loader)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(page.Results))
	for _, hit := range page.Results {
		out = append(out, history.SearchHitMap(hit.Value, hit.Score))
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

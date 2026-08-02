package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
	"browser-server/internal/todo"
)

//go:embed schemas/search_todos.json
var searchTodosSchema []byte

func registerSearchTodos(r *Registry) {
	r.add(Tool{
		Name:        "search_todos",
		Category:    "General",
		Description: "Search the local todo database. Can filter by status, priority, text query across title/description, and exact tags. Results include each todo's tags and a relevance score. Supports one-based pagination (page, page_size). When multiple tags are given, every tag must be present on a returned todo (AND semantics). The legacy `limit` argument is deprecated and maps to `page_size` when `page_size` is omitted.",
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
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
		Limit    int      `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "status": true, "priority": true, "tags": true, "page": true, "page_size": true, "limit": true}); err != nil {
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

	pageSize, err := resolvePageSize(a.PageSize, a.Limit, 50, 10)
	if err != nil {
		return nil, err
	}

	loader := func(ctx context.Context, req searchengine.CandidateRequest) (searchengine.CandidateSet[models.Todo], error) {
		return todo.SearchCandidates(ctx, todo.SearchFilter{
			UserID:   a.UserID,
			Status:   a.Status,
			Priority: a.Priority,
			Tags:     a.Tags,
		}, req)
	}

	page, err := searchengine.Search(ctx, searchengine.Request{Query: a.Query, Page: a.Page, PageSize: pageSize}, loader)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(page.Results))
	for _, hit := range page.Results {
		t := hit.Value
		tags := helpers.ParseTagsFromJSON(t.Tags)
		if tags == nil {
			tags = []string{}
		}
		out = append(out, todo.TodoSearchHitMap(t, tags, hit.Score))
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

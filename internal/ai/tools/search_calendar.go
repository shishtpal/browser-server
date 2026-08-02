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

//go:embed schemas/search_calendar.json
var searchCalendarSchema []byte

func registerSearchCalendar(r *Registry) {
	r.add(Tool{
		Name:        "search_calendar",
		Category:    "General",
		Description: "Search calendar events (todos with scheduled dates). Can filter by date range, text query, status, and exact tags. Results include each event's tags and a relevance score. Supports one-based pagination (page, page_size). When multiple tags are given, every tag must be present on a returned event (AND semantics). Returns events that have a start_date set. The legacy `limit` argument is deprecated and maps to `page_size` when `page_size` is omitted.",
		Schema:      json.RawMessage(searchCalendarSchema),
		Execute:     searchCalendar,
	})
}

func searchCalendar(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID   int      `json:"user_id"`
		Query    string   `json:"query"`
		From     string   `json:"from"`
		To       string   `json:"to"`
		Status   string   `json:"status"`
		Tags     []string `json:"tags"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
		Limit    int      `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "from": true, "to": true, "status": true, "tags": true, "page": true, "page_size": true, "limit": true}); err != nil {
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
			UserID:      a.UserID,
			Status:      a.Status,
			Tags:        a.Tags,
			Scheduled:   true,
			StartDate:   a.From,
			EndDate:     a.To,
			OrderByDate: true,
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
		out = append(out, todo.CalendarSearchHitMap(t, tags, hit.Score))
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

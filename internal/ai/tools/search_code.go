package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"browser-server/internal/codesearch"
	"browser-server/internal/searchengine"
)

//go:embed schemas/search_code.json
var searchCodeSchema []byte

func registerSearchCode(r *Registry) {
	r.add(Tool{
		Name:        "search_code",
		Category:    "Code Intelligence",
		Description: "Search source files using regex, literal, or fixed-string matching. Results include a relevance score and are paginated with one-based page/page_size. The legacy `max_results` argument is deprecated and maps to `page_size` when `page_size` is omitted. When raw output is enabled for this tool, results are rendered as compact plain text instead of JSON.",
		Schema:      json.RawMessage(searchCodeSchema),
		Execute:     searchCode,
		RawContentFunc: rawSearchCodeFormatter,
	})
}

func searchCode(ctx context.Context, raw json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, codeToolTimeout)
	defer cancel()
	var a struct {
		Pattern       string   `json:"pattern"`
		Path          string   `json:"path"`
		Type          string   `json:"type"`
		Include       []string `json:"include"`
		Exclude       []string `json:"exclude"`
		CaseSensitive bool     `json:"case_sensitive"`
		WholeWord     bool     `json:"whole_word"`
		Page          int      `json:"page"`
		PageSize      int      `json:"page_size"`
		MaxResults    int      `json:"max_results"`
		ContextLines  *int     `json:"context_lines"`
	}
	if err := strict(raw, &a, map[string]bool{"pattern": true, "path": true, "include": true, "exclude": true, "case_sensitive": true, "whole_word": true, "page": true, "page_size": true, "max_results": true, "context_lines": true, "type": true}); err != nil {
		return nil, err
	}
	if err := validateSearchPagination(raw, "max_results"); err != nil {
		return nil, err
	}
	if a.Pattern == "" || len(a.Pattern) > 500 {
		return nil, fmt.Errorf("pattern is required and must not exceed 500 characters")
	}
	if a.Type == "" {
		a.Type = "regex"
	}
	if a.Type != "regex" && a.Type != "literal" && a.Type != "fixed" {
		return nil, fmt.Errorf("type must be regex, literal, or fixed")
	}
	if err := validateGlobs(a.Include, a.Exclude); err != nil {
		return nil, err
	}
	contextLines := 2
	if a.ContextLines != nil {
		contextLines = *a.ContextLines
	}
	if contextLines < 0 || contextLines > 10 {
		return nil, fmt.Errorf("context_lines must be 0 to 10")
	}

	pageSize, err := resolvePageSize(a.PageSize, a.MaxResults, 100, 10)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	loader := func(ctx context.Context, req searchengine.CandidateRequest) (searchengine.CandidateSet[codesearch.Match], error) {
		return codesearch.CandidateSet(ctx, codesearch.Options{
			Root: a.Path, Pattern: a.Pattern, Type: a.Type, Include: a.Include,
			Exclude: a.Exclude, CaseSensitive: a.CaseSensitive, WholeWord: a.WholeWord,
			ContextLines: contextLines, MaxSourceSize: maxSourceSize, MaxCandidates: req.MaxCandidates,
		})
	}

	page, err := searchengine.Search(ctx, searchengine.Request{Query: a.Pattern, Page: a.Page, PageSize: pageSize}, loader,
		searchengine.WithStrategy[codesearch.Match](searchengine.ExactStrategy[codesearch.Match]()))
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, len(page.Results))
	for i, hit := range page.Results {
		m := hit.Value
		results[i] = map[string]any{
			"file": m.File, "line": m.Line, "column": m.Column, "match": m.Match,
			"context_before": m.ContextBefore, "context_after": m.ContextAfter, "score": hit.Score,
		}
	}
	return makeSearchCodeEnvelope(
		a.Pattern,
		page.Page,
		page.PageSize,
		page.Total,
		page.Total,
		page.HasMore,
		page.Truncated,
		time.Since(start).Milliseconds(),
		results,
	), nil
}

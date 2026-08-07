package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/quiz"
	quizconfig "browser-server/internal/quiz/config"
	"browser-server/internal/searchengine"
)

//go:embed schemas/search_questions.json
var searchQuestionsSchema []byte

func registerSearchQuestions(r *Registry) {
	r.add(Tool{
		Name:        "search_questions",
		Category:    "General",
		Description: "Search the local question bank. Can filter by user, question type, difficulty, and exam/subject/topic/sub_topic tags. The `tags` filter accepts one or more tag strings; a question matches if its `tags` array contains any of them. Results include a relevance score. Supports one-based pagination (page, page_size). Set `random: true` to draw a random sample of up to `page_size` matching questions instead of ranking by relevance. The legacy `limit` argument is deprecated and maps to `page_size` when `page_size` is omitted.",
		Schema:      json.RawMessage(searchQuestionsSchema),
		Execute:     searchQuestions,
	})
}

func searchQuestions(ctx context.Context, raw json.RawMessage) (any, error) {
	if !quizconfig.Get().Enabled {
		return nil, fmt.Errorf("quiz feature disabled")
	}

	var a struct {
		UserID     int      `json:"user_id"`
		Query      string   `json:"query"`
		Type       string   `json:"type"`
		Difficulty string   `json:"difficulty"`
		Tags       []string `json:"tags"`
		Subject    string   `json:"subject"`
		Topic      string   `json:"topic"`
		SubTopic   string   `json:"sub_topic"`
		Page       int      `json:"page"`
		PageSize   int      `json:"page_size"`
		Limit      int      `json:"limit"`
		Random     bool     `json:"random"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "query": true, "type": true, "difficulty": true,
		"tags": true, "subject": true, "topic": true, "sub_topic": true,
		"page": true, "page_size": true, "limit": true, "random": true,
	}); err != nil {
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

	rules := quizconfig.Get().Rules()
	if a.Type != "" {
		if err := rules.ValidateType(a.Type); err != nil {
			return nil, err
		}
	}
	if err := rules.ValidateDifficulty(a.Difficulty); err != nil {
		return nil, err
	}

	pageSize, err := resolvePageSize(a.PageSize, a.Limit, 10, 10)
	if err != nil {
		return nil, err
	}

	filter := quiz.Filter{
		UserID: a.UserID,
		Type: a.Type, Difficulty: a.Difficulty,
		Tags: a.Tags, Subject: a.Subject, Topic: a.Topic, SubTopic: a.SubTopic,
	}

	if a.Random {
		// A random draw has no relevance order and no stable page boundaries,
		// so text ranking and pagination are both rejected rather than ignored.
		if a.Query != "" {
			return nil, fmt.Errorf("cannot specify both query and random")
		}
		if a.Page > 1 {
			return nil, fmt.Errorf("random results cannot be paginated")
		}
		records, err := quiz.SampleRandom(ctx, filter, pageSize)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			out = append(out, quiz.SearchHitMap(rec, 0))
		}
		return fitSearchEnvelope(ctx, map[string]any{
			"random":    true,
			"page":      1,
			"page_size": pageSize,
			"total":     len(out),
			"has_more":  false,
			"truncated": false,
			"results":   out,
		}), nil
	}

	loader := func(ctx context.Context, req searchengine.CandidateRequest) (searchengine.CandidateSet[quiz.Record], error) {
		return quiz.SearchCandidates(ctx, filter, req.MaxCandidates)
	}

	page, err := searchengine.Search(ctx, searchengine.Request{Query: a.Query, Page: a.Page, PageSize: pageSize}, loader)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(page.Results))
	for _, hit := range page.Results {
		out = append(out, quiz.SearchHitMap(hit.Value, hit.Score))
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

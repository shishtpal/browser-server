// Package searchengine provides a generic, typed search pipeline for local
// data sources. It normalizes queries, applies pluggable ranking strategies,
// sorts results deterministically, and paginates them.
package searchengine

import (
	"context"
	"fmt"
	"strings"
)

// DefaultCandidateCap limits how many candidates a source loads before
// ranking. This protects memory/latency; loaders may use a smaller cap.
const DefaultCandidateCap = 5000

// Request is the common pagination and query envelope for every source.
type Request struct {
	Query    string
	Page     int
	PageSize int
}

// CandidateRequest is what the engine passes to a source loader.
type CandidateRequest struct {
	// Query is the original, trimmed query string from the caller.
	Query string
	// Terms are normalized, whitespace-separated query terms.
	Terms []string
	// MaxCandidates is the maximum number of candidates the source should load.
	MaxCandidates int
}

// Field is a single searchable text field and its weight.
type Field struct {
	// Name is informational only; it is not used for scoring.
	Name string
	// Text is the searchable content.
	Text string
	// Weight multiplies matches in this field.
	Weight float64
}

// Candidate is one source-specific record ready for ranking.
type Candidate[T any] struct {
	// Key is a stable, unique identifier used for deterministic tie-breaking.
	Key string
	// Fields are the searchable text fields exposed to the ranking strategy.
	Fields []Field
	// Value is the source-specific record.
	Value T
	// BaseScore is an optional score supplied by the source (e.g. FTS5 BM25).
	// Strategies may use it as a starting point or ignore it.
	BaseScore float64
	// SourceRank preserves the source's domain ordering for empty queries and
	// stable ties. Lower values rank first.
	SourceRank int
}

// CandidateSet is the result of a source loader. It includes the candidates
// and whether the source stopped early because of MaxCandidates.
type CandidateSet[T any] struct {
	Candidates []Candidate[T]
	Truncated  bool
}

// Loader loads candidates for a source. It is responsible for ownership checks,
// structured filters, scanning, and field weighting.
type Loader[T any] func(context.Context, CandidateRequest) (CandidateSet[T], error)

// Hit is one scored result.
type Hit[T any] struct {
	Value      T
	Score      float64
	baseScore  float64
	sourceRank int
	key        string
}

// Page is the search output envelope.
type Page[T any] struct {
	Query     string   `json:"query"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
	Total     int      `json:"total"`
	HasMore   bool     `json:"has_more"`
	Truncated bool     `json:"truncated"`
	Results   []Hit[T] `json:"results"`
}

// Option customizes a search invocation.
type Option[T any] func(*config[T])

// WithStrategy sets a non-default ranking strategy.
func WithStrategy[T any](s Strategy[T]) Option[T] {
	return func(cfg *config[T]) { cfg.strategy = s }
}

// WithCandidateCap sets the maximum number of candidates to load from a source.
func WithCandidateCap[T any](cap int) Option[T] {
	return func(cfg *config[T]) {
		if cap < 1 {
			cap = DefaultCandidateCap
		}
		cfg.candidateCap = cap
	}
}

type config[T any] struct {
	strategy     Strategy[T]
	candidateCap int
}

// Search runs the generic search pipeline for the typed loader.
func Search[T any](ctx context.Context, req Request, loader Loader[T], opts ...Option[T]) (Page[T], error) {
	page := req.Page
	if page < 0 {
		return Page[T]{}, fmt.Errorf("page must be at least 1")
	}
	if page == 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 0 {
		return Page[T]{}, fmt.Errorf("page_size must be between 1 and 100")
	}
	if pageSize == 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		return Page[T]{}, fmt.Errorf("page_size must be between 1 and 100")
	}

	cfg := &config[T]{
		strategy:     FuzzyStrategy[T](),
		candidateCap: DefaultCandidateCap,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	query := strings.TrimSpace(req.Query)
	terms := normalizeTerms(query)

	set, err := loader(ctx, CandidateRequest{Query: query, Terms: terms, MaxCandidates: cfg.candidateCap})
	if err != nil {
		return Page[T]{}, err
	}

	candidates := set.Candidates
	truncated := set.Truncated
	if len(candidates) > cfg.candidateCap {
		candidates = candidates[:cfg.candidateCap]
		truncated = true
	}

	scored := make([]Hit[T], 0, len(candidates))
	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return Page[T]{}, err
		}
		score := cfg.strategy.Score(ctx, terms, c)
		if score > 0 || query == "" {
			scored = append(scored, Hit[T]{
				Value: c.Value, Score: roundScore(score), baseScore: c.BaseScore,
				sourceRank: c.SourceRank, key: c.Key,
			})
		}
	}

	cfg.strategy.Sort(scored, candidates)

	total := len(scored)
	start := total
	if page-1 <= total/pageSize {
		start = (page - 1) * pageSize
	}
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pageResults := scored[start:end]

	return Page[T]{
		Query:     query,
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		HasMore:   end < total,
		Truncated: truncated,
		Results:   pageResults,
	}, nil
}

// normalizeTerms returns lowercased, non-empty tokens separated by whitespace.
func normalizeTerms(query string) []string {
	var terms []string
	for _, t := range strings.Fields(query) {
		t = strings.ToLower(t)
		if t != "" {
			terms = append(terms, t)
		}
	}
	return terms
}

// roundScore rounds a score to four decimals to keep JSON stable.
func roundScore(s float64) float64 {
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	return float64(int(s*10000+0.5)) / 10000
}

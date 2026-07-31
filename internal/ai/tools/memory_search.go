package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

func (s *memoryStore) recall(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		ID                string `json:"id"`
		Search            string `json:"search"`
		IncludeReferences bool   `json:"include_references"`
		LoadLazy          bool   `json:"load_lazy"`
		MaxDepth          int    `json:"max_depth"`
		Limit             int    `json:"limit"`
	}
	if e := strict(raw, &a, map[string]bool{"id": true, "search": true, "include_references": true, "max_depth": true, "load_lazy": true, "limit": true}); e != nil {
		return nil, e
	}
	if a.Limit == 0 {
		a.Limit = 20
	}
	if a.Limit < 1 || a.Limit > 100 {
		return nil, fmt.Errorf("limit must be 1 to 100")
	}
	if a.ID != "" {
		m, e := s.read(a.ID, a.LoadLazy || !mLazy(s, a.ID))
		if e != nil {
			return nil, e
		}
		out := []memoryData{m}
		if a.IncludeReferences {
			rs, e := s.resolveFrom(ctx, a.ID, a.MaxDepth, true)
			if e != nil {
				return nil, e
			}
			out = rs
		}
		return map[string]any{"memories": out, "total_count": len(out)}, nil
	}
	if a.Search == "" {
		return nil, fmt.Errorf("either id or search is required")
	}
	return s.searchValues(ctx, a.Search, nil, "", "", a.Limit, !a.LoadLazy)
}

func mLazy(s *memoryStore, id string) bool {
	m, e := s.read(id, false)
	return e == nil && m.Metadata.Lazy
}

func (s *memoryStore) search(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Query        string   `json:"query"`
		Category     string   `json:"category"`
		Importance   string   `json:"importance"`
		Tags         []string `json:"tags"`
		Limit        int      `json:"limit"`
		MetadataOnly bool     `json:"metadata_only"`
	}
	if e := strict(raw, &a, map[string]bool{"query": true, "tags": true, "category": true, "importance": true, "limit": true, "metadata_only": true}); e != nil {
		return nil, e
	}
	if a.Limit == 0 {
		a.Limit = 20
	}
	return s.searchValues(ctx, a.Query, a.Tags, a.Category, a.Importance, a.Limit, a.MetadataOnly)
}

// fuzzyScore computes a relevance score for a query against a haystack.
// It splits the query into words and scores based on:
// - exact full-query substring match (bonus)
// - individual word matches (partial credit per word)
// - word position proximity in title vs content (title matches weighted higher)
// Returns 0 if fewer than half the query words match (threshold).
func fuzzyScore(query string, words []string, title, content, tagStr string) float64 {
	if len(words) == 0 {
		return 1.0 // no query means everything matches equally
	}

	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)
	tagsLower := strings.ToLower(tagStr)
	hay := titleLower + " " + contentLower + " " + tagsLower

	matched := 0
	var score float64

	for _, w := range words {
		if strings.Contains(hay, w) {
			matched++
			// Title match is worth more than content match
			if strings.Contains(titleLower, w) {
				score += 3.0
			} else if strings.Contains(tagsLower, w) {
				score += 2.0
			} else {
				score += 1.0
			}
		}
	}

	// Require at least half the words to match (fuzzy threshold)
	threshold := (len(words) + 1) / 2 // ceiling division
	if matched < threshold {
		return 0
	}

	// Bonus for exact full-query substring match
	if strings.Contains(hay, query) {
		score += float64(len(words)) * 2.0
	}

	// Normalize by word count so longer queries don't inflate scores
	score = score / float64(len(words))

	// Bonus for matching all words
	if matched == len(words) {
		score *= 1.5
	}

	return score
}

func (s *memoryStore) searchValues(ctx context.Context, q string, tags []string, cat, imp string, limit int, meta bool) (any, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be 1 to 100")
	}
	q = strings.ToLower(strings.TrimSpace(q))
	words := strings.Fields(q)

	scored := []scoredMemory{}
	e := s.walk(ctx, func(_ string, m memoryData) error {
		if cat != "" && m.Metadata.Category != cat {
			return nil
		}
		if imp != "" && m.Metadata.Importance != imp {
			return nil
		}
		for _, t := range tags {
			found := false
			for _, mt := range m.Metadata.Tags {
				if mt == t {
					found = true
				}
			}
			if !found {
				return nil
			}
		}

		// Compute fuzzy relevance score
		sc := fuzzyScore(q, words, m.Metadata.Title, m.Content, strings.Join(m.Metadata.Tags, " "))
		if q != "" && sc == 0 {
			return nil
		}

		if meta || m.Metadata.Lazy {
			m.Content = ""
		}
		scored = append(scored, scoredMemory{data: m, score: sc})
		return nil
	})
	if errors.Is(e, fs.SkipAll) {
		e = nil
	}
	if e != nil {
		return nil, e
	}

	// Sort by score descending (best matches first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply limit
	if len(scored) > limit {
		scored = scored[:limit]
	}

	out := make([]memoryData, len(scored))
	for i, sm := range scored {
		out[i] = sm.data
	}

	return map[string]any{"memories": out, "total_count": len(out)}, nil
}

func (s *memoryStore) list(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Type, Tag, Category, Importance string
		Limit                           int
	}
	if e := strict(raw, &a, map[string]bool{"type": true, "tag": true, "category": true, "importance": true, "limit": true}); e != nil {
		return nil, e
	}
	tags := []string{}
	if a.Tag != "" {
		tags = []string{a.Tag}
	}
	return s.searchValues(ctx, "", tags, a.Category, a.Importance, defaultLimit(a.Limit), true)
}

func defaultLimit(v int) int {
	if v == 0 {
		return 100
	}
	return v
}

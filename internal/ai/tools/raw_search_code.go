package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// rawSearchCodeFormatter turns a search_code result envelope into a compact,
// token-efficient text representation. It accepts the same typed result value
// produced by searchCode so JSON and raw rendering share one source of truth.
//
// Output format:
//
//	QUERY <query> | matches=N page=P/S score=CONST has_more=true truncated=true
//	FILE <path>
//	L:C [score=S]
//	  -2 <context line two above>
//	  -1 <context line one above>
//	  <match line itself>
//	  +1 <context line one below>
//	  +2 <context line two below>
//
// The match line is included explicitly so the hit is unambiguous even when
// context lines are blank or omitted. Blank context lines are dropped to save
// tokens. Embedded newlines are rendered as the visible glyph ⏎ so the
// one-source-line-per-rendered-line invariant is preserved.
//
// Scores are rendered only when they vary across results. If every hit has
// the same score, that score is hoisted into the header line.
//
// Returns (nil, false) when v is not a search-code envelope, so the registry
// falls back to JSON safely.
func rawSearchCodeFormatter(v any) ([]byte, bool) {
	env, ok := v.(searchCodeEnvelope)
	if !ok {
		return nil, false
	}

	var b strings.Builder
	if len(env.Results) == 0 {
		fmt.Fprintf(&b, "QUERY %s | no matches\n", env.Query)
		return []byte(b.String()), true
	}

	scoreConst := true
	constScore := env.Results[0].Score
	for _, r := range env.Results[1:] {
		if r.Score != constScore {
			scoreConst = false
			break
		}
	}

	header := fmt.Sprintf("QUERY %s | matches=%d page=%d", env.Query, len(env.Results), env.Page)
	if env.PageSize > 0 {
		header += fmt.Sprintf("/%d", env.PageSize)
	}
	if scoreConst {
		header += fmt.Sprintf(" score=%g", constScore)
	}
	if env.HasMore {
		header += " has_more=true"
	}
	if env.Truncated {
		header += " truncated=true"
	}
	fmt.Fprintln(&b, header)

	for _, file := range env.Files {
		fmt.Fprintf(&b, "\nFILE %s\n", file.Path)
		for _, r := range file.Hits {
			line := fmt.Sprintf("%d:%d", r.Line, r.Column)
			if !scoreConst {
				line += fmt.Sprintf(" score=%g", r.Score)
			}
			fmt.Fprintln(&b, line)

			// Render context before: closest line is -1, furthest is -N.
			n := len(r.ContextBefore)
			for i, c := range r.ContextBefore {
				if isBlank(c) {
					continue
				}
				offset := -(n - i)
				fmt.Fprintf(&b, "  %d %s\n", offset, sanitizeLine(c))
			}

			fmt.Fprintf(&b, "  > %s\n", sanitizeLine(r.Match))

			// Render context after: +1, +2, ...
			for i, c := range r.ContextAfter {
				if isBlank(c) {
					continue
				}
				fmt.Fprintf(&b, "  +%d %s\n", i+1, sanitizeLine(c))
			}
		}
	}

	return []byte(b.String()), true
}

// searchCodeEnvelope is the typed result produced by searchCode. Keeping it
// typed lets the raw formatter avoid re-decoding JSON. It still marshals to
// the same JSON shape the original search_code used.
type searchCodeEnvelope struct {
	Query         string
	Page          int
	PageSize      int
	Total         int
	TotalMatches  int
	HasMore       bool
	Truncated     bool
	SearchTimeMs  int64
	Results       []searchCodeHit
	Files         []searchCodeFile
}

type searchCodeHit struct {
	File          string
	Line          int
	Column        int
	Match         string
	Score         float64
	ContextBefore []string
	ContextAfter  []string
}

type searchCodeFile struct {
	Path string
	Hits []searchCodeHit
}

// makeSearchCodeEnvelope converts the public JSON-friendly map produced by
// searchCode into a typed envelope suitable for both JSON marshaling and raw
// formatting. This is the single shared source value.
func makeSearchCodeEnvelope(query string, page, pageSize, total, totalMatches int, hasMore, truncated bool, searchTimeMs int64, rawResults []map[string]any) searchCodeEnvelope {
	hits := make([]searchCodeHit, len(rawResults))
	files := make([]searchCodeFile, 0, len(rawResults))
	fileIndex := map[string]int{}

	for i, raw := range rawResults {
		hit := searchCodeHit{
			File:          stringFrom(raw, "file"),
			Line:          intFrom(raw, "line"),
			Column:        intFrom(raw, "column"),
			Match:         stringFrom(raw, "match"),
			Score:         floatFrom(raw, "score"),
			ContextBefore: stringSliceFrom(raw, "context_before"),
			ContextAfter:  stringSliceFrom(raw, "context_after"),
		}
		hits[i] = hit

		idx, ok := fileIndex[hit.File]
		if !ok {
			idx = len(files)
			fileIndex[hit.File] = idx
			files = append(files, searchCodeFile{Path: hit.File})
		}
		files[idx].Hits = append(files[idx].Hits, hit)
	}

	return searchCodeEnvelope{
		Query:         query,
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		TotalMatches:  totalMatches,
		HasMore:       hasMore,
		Truncated:     truncated,
		SearchTimeMs:  searchTimeMs,
		Results:       hits,
		Files:         files,
	}
}

// MarshalJSON implements json.Marshaler to preserve the original search_code
// JSON output shape: flat envelope with a results array of maps.
func (e searchCodeEnvelope) MarshalJSON() ([]byte, error) {
	results := make([]map[string]any, len(e.Results))
	for i, r := range e.Results {
		results[i] = map[string]any{
			"file":           r.File,
			"line":           r.Line,
			"column":         r.Column,
			"match":          r.Match,
			"score":          r.Score,
			"context_before": r.ContextBefore,
			"context_after":  r.ContextAfter,
		}
	}
	return json.Marshal(map[string]any{
		"query":          e.Query,
		"page":           e.Page,
		"page_size":      e.PageSize,
		"total":          e.Total,
		"has_more":       e.HasMore,
		"truncated":      e.Truncated,
		"results":        results,
		"total_matches":  e.TotalMatches,
		"search_time_ms": e.SearchTimeMs,
	})
}

// AsMap serializes the envelope to JSON and decodes it back into a map[string]any
// for tests that assert against the original JSON shape.
func (e searchCodeEnvelope) AsMap() (map[string]any, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// stringFrom extracts a string value from a map[string]any.
func stringFrom(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// intFrom extracts an int value from a map[string]any.
func intFrom(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return 0
}

// floatFrom extracts a float64 value from a map[string]any.
func floatFrom(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	if v, ok := m[key].(float32); ok {
		return float64(v)
	}
	return 0
}

// stringSliceFrom extracts a []string value from a map[string]any.
func stringSliceFrom(m map[string]any, key string) []string {
	if v, ok := m[key].([]string); ok {
		return v
	}
	if v, ok := m[key].([]any); ok {
		out := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				out[i] = s
			}
		}
		return out
	}
	return nil
}

// isBlank reports whether a context line carries no visible signal.
func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// sanitizeLine keeps a source line as a single rendered line. Embedded newline
// and carriage-return characters are replaced with a visible marker, and any
// trailing newline is removed. Other Unicode is preserved.
func sanitizeLine(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "⏎")
	s = strings.ReplaceAll(s, "\n", "⏎")
	s = strings.ReplaceAll(s, "\r", "⏎")
	// Trim a trailing visible marker if the original line ended with a newline.
	s = strings.TrimRight(s, "⏎")
	if !utf8.ValidString(s) {
		// Should not happen for code search results, but be defensive.
		s = strings.ToValidUTF8(s, "")
	}
	return s
}

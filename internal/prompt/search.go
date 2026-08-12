package prompt

import (
	"context"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/searchengine"
)

// SearchCandidates loads prompt records for fuzzy ranking. It does not apply
// the final text LIKE, title-first ordering, or page limit; those are the
// engine's job. It is intended for the search_prompts AI tool.
func SearchCandidates(ctx context.Context, userID int, maxCandidates int) (searchengine.CandidateSet[Record], error) {
	where := []string{"user_id = ?"}
	args := []any{userID}

	query := "SELECT " + Columns + " FROM prompts WHERE " + strings.Join(where, " AND ") + " ORDER BY created_at DESC, id DESC"
	if maxCandidates > 0 {
		query += " LIMIT ?"
		args = append(args, maxCandidates+1)
	}
	rows, err := db.PromptDB.QueryContext(ctx, query, args...)
	if err != nil {
		return searchengine.CandidateSet[Record]{}, err
	}
	records, err := ScanAll(rows)
	if err != nil {
		return searchengine.CandidateSet[Record]{}, err
	}
	truncated := maxCandidates > 0 && len(records) > maxCandidates
	if truncated {
		records = records[:maxCandidates]
	}
	candidates := make([]searchengine.Candidate[Record], len(records))
	for i, rec := range records {
		tags := rec.Tags()
		candidates[i] = searchengine.Candidate[Record]{
			Key: fmt.Sprintf("prompt:%d", rec.Prompt.ID),
			Fields: []searchengine.Field{
				{Name: "title", Text: rec.Prompt.Title, Weight: 10},
				{Name: "content", Text: rec.Prompt.Content, Weight: 2},
				{Name: "description", Text: rec.Prompt.Description, Weight: 3},
				{Name: "tags", Text: strings.Join(tags, " "), Weight: 5},
			},
			Value:      rec,
			SourceRank: i,
		}
	}
	return searchengine.CandidateSet[Record]{Candidates: candidates, Truncated: truncated}, nil
}

// SearchHitMap renders a scored prompt record as the map the search_prompts tool
// returns, preserving the existing fields of prompt.SearchMap plus a score.
func SearchHitMap(rec Record, score float64) map[string]any {
	return map[string]any{
		"id":          rec.Prompt.ID,
		"title":       rec.Prompt.Title,
		"content":     rec.Prompt.Content,
		"description": rec.Prompt.Description,
		"pinned":      rec.Prompt.Pinned,
		"score":       score,
	}
}

package history

import (
	"context"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
)

// HistoryCandidate is the search-engine candidate type for history.
type HistoryCandidate = searchengine.Candidate[models.History]

// HistoryCandidateSet is a set of history candidates.
type HistoryCandidateSet = searchengine.CandidateSet[models.History]

// SearchCandidates loads history entries for fuzzy ranking. It does not apply
// the final term AND SQL, but it does apply ownership and optional exact
// domain filtering. It is intended for the search_history AI tool.
func SearchCandidates(ctx context.Context, userID int, domain string, maxCandidates int) (HistoryCandidateSet, error) {
	where := []string{"user_id = ?"}
	args := []any{userID}
	if domain != "" {
		where = append(where, "domain = ?")
		args = append(args, domain)
	}
	query := "SELECT " + Columns + " FROM history WHERE " + strings.Join(where, " AND ") + " ORDER BY visited_at DESC, id DESC"
	if maxCandidates > 0 {
		query += " LIMIT ?"
		args = append(args, maxCandidates+1)
	}
	rows, err := db.HistoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return HistoryCandidateSet{}, err
	}
	entries, err := ScanAll(rows)
	if err != nil {
		return HistoryCandidateSet{}, err
	}
	truncated := maxCandidates > 0 && len(entries) > maxCandidates
	if truncated {
		entries = entries[:maxCandidates]
	}
	candidates := make([]HistoryCandidate, len(entries))
	for i, h := range entries {
		candidates[i] = HistoryCandidate{
			Key: fmt.Sprintf("history:%d", h.ID),
			Fields: []searchengine.Field{
				{Name: "title", Text: h.Title, Weight: 10},
				{Name: "url", Text: h.URL, Weight: 7},
				{Name: "domain", Text: h.Domain, Weight: 5},
			},
			Value:      h,
			SourceRank: i,
		}
	}
	return HistoryCandidateSet{Candidates: candidates, Truncated: truncated}, nil
}

// SearchHitMap renders a scored history entry as the map the search_history
// tool returns, preserving the existing fields of history.SearchMap plus a
// score.
func SearchHitMap(h models.History, score float64) map[string]any {
	return map[string]any{
		"id":         h.ID,
		"url":        h.URL,
		"title":      h.Title,
		"domain":     h.Domain,
		"visited_at": h.VisitedAt,
		"duration":   h.Duration,
		"score":      score,
	}
}

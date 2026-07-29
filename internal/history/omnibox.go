package history

import (
	"context"
	"database/sql"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

// OmniboxSearch returns URL-grouped history suggestions for the omnibox, most
// recently visited first. Each result is tagged with Source = "history".
func OmniboxSearch(ctx context.Context, userID int, search string, limit int) ([]models.OmniboxSearchResult, error) {
	where := "WHERE 1=1"
	args := []any{}

	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	termClause, termArgs := SearchTerms(search, "title", "url")
	where += termClause
	args = append(args, termArgs...)

	query := "SELECT url, title, COUNT(*), MAX(visited_at) FROM history " +
		where + " GROUP BY url ORDER BY MAX(visited_at) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.HistoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.OmniboxSearchResult, 0)
	for rows.Next() {
		var r models.OmniboxSearchResult
		var lastVisited sql.NullString
		if err := rows.Scan(&r.URL, &r.Title, &r.VisitCount, &lastVisited); err != nil {
			return nil, err
		}
		r.Source = "history"
		if lastVisited.Valid {
			parsed := parseSQLiteTime(lastVisited.String)
			r.LastVisited = &parsed
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

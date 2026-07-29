package history

import (
	"context"
	"database/sql"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

// GroupedOptions filters the URL-aggregated history view.
type GroupedOptions struct {
	UserID int
	Search string // whitespace-separated terms, AND-ed
	Column string // "all" (default), "title", or "url"
	Domain string
	Limit  int
	Offset int
}

// ListGrouped returns history aggregated by URL, searched and paged entirely
// on the server so clients never load every row at once. SQLite's bare-column
// rule means title is taken from the row holding MAX(visited_at), i.e. the
// most recent visit for that URL.
func ListGrouped(ctx context.Context, opts GroupedOptions) (models.GroupedHistoryResponse, error) {
	where := "WHERE 1=1"
	args := []any{}

	if opts.UserID > 0 {
		where += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if domain := strings.ToLower(strings.TrimSpace(opts.Domain)); domain != "" {
		where += " AND domain = ?"
		args = append(args, domain)
	}

	// Each whitespace-separated term must match (AND), mirroring the previous
	// client-side search behaviour.
	searchColumns := []string{"title", "url"}
	switch opts.Column {
	case "title":
		searchColumns = []string{"title"}
	case "url":
		searchColumns = []string{"url"}
	}
	termClause, termArgs := SearchTerms(opts.Search, searchColumns...)
	where += termClause
	args = append(args, termArgs...)

	var total int
	if err := db.HistoryDB.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT url) FROM history "+where, args...).Scan(&total); err != nil {
		return models.GroupedHistoryResponse{}, err
	}

	query := "SELECT url, title, COUNT(*), COALESCE(SUM(duration), 0), MAX(visited_at) FROM history " +
		where + " GROUP BY url ORDER BY MAX(visited_at) DESC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := db.HistoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return models.GroupedHistoryResponse{}, err
	}
	defer rows.Close()

	entries := make([]models.GroupedHistoryEntry, 0)
	for rows.Next() {
		var entry models.GroupedHistoryEntry
		var lastVisited sql.NullString
		if err := rows.Scan(&entry.URL, &entry.Title, &entry.Count, &entry.TotalDuration, &lastVisited); err != nil {
			return models.GroupedHistoryResponse{}, err
		}
		entry.LastVisited = nullTime(lastVisited)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return models.GroupedHistoryResponse{}, err
	}

	return models.GroupedHistoryResponse{
		Entries: entries,
		Total:   total,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
	}, nil
}

// ListDomains returns every hostname represented in history, ordered by visit
// count so the most-used domain appears first.
func ListDomains(ctx context.Context, userID int, search string) ([]models.HistoryDomainSummary, error) {
	where := "WHERE domain <> ''"
	args := []any{}
	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}
	if search != "" {
		where += " AND domain LIKE ?"
		args = append(args, "%"+strings.ToLower(search)+"%")
	}

	query := "SELECT domain, COUNT(*), COUNT(DISTINCT url), COALESCE(SUM(duration), 0), MAX(visited_at) " +
		"FROM history " + where + " GROUP BY domain" +
		" ORDER BY COUNT(*) DESC, MAX(visited_at) DESC"
	rows, err := db.HistoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make([]models.HistoryDomainSummary, 0)
	for rows.Next() {
		var d models.HistoryDomainSummary
		var lastVisited sql.NullString
		if err := rows.Scan(&d.Domain, &d.VisitCount, &d.URLCount, &d.TotalDuration, &lastVisited); err != nil {
			return nil, err
		}
		d.LastVisited = nullTime(lastVisited)
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return domains, nil
}

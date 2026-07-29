package history

import (
	"context"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

// ListOptions filters the flat history listing.
type ListOptions struct {
	UserID int
	URL    string
	Limit  int
	Offset int
}

// List returns history entries ordered by most recently visited, optionally
// filtered by user and a URL substring. When Limit > 0 the result is paged.
func List(ctx context.Context, opts ListOptions) ([]models.History, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 3)
	if opts.UserID > 0 {
		where = append(where, "user_id = ?")
		args = append(args, opts.UserID)
	}
	if opts.URL != "" {
		where = append(where, "url LIKE ?")
		args = append(args, "%"+opts.URL+"%")
	}
	query := "SELECT " + Columns + " FROM history WHERE " + strings.Join(where, " AND ") +
		" ORDER BY visited_at DESC"
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}
	rows, err := db.HistoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

// Search finds history entries matching a query and optional domain, most
// recently visited first, scoped to one user.
//
// When query contains multiple whitespace-separated terms, every term must
// match (AND) against url OR title, giving multi-word searches meaningful
// results rather than a single substring match.
func Search(ctx context.Context, userID int, query, domain string, limit int) ([]models.History, error) {
	where := []string{"user_id = ?"}
	args := []any{userID}
	if query != "" {
		clause, termArgs := SearchTerms(query, "url", "title")
		// SearchTerms returns a clause prefixed with " AND "; strip it so it
		// can be used as a bare WHERE condition.
		where = append(where, clause[5:])
		args = append(args, termArgs...)
	}
	if domain != "" {
		where = append(where, "domain = ?")
		args = append(args, domain)
	}
	args = append(args, limit)
	rows, err := db.HistoryDB.QueryContext(ctx,
		"SELECT "+Columns+" FROM history WHERE "+strings.Join(where, " AND ")+
			" ORDER BY visited_at DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

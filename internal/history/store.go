package history

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

type rowScanner interface{ Scan(...any) error }

// Scan reads a row selected with Columns into a History entry.
func Scan(scanner rowScanner) (models.History, error) {
	var h models.History
	err := scanner.Scan(&h.ID, &h.UserID, &h.URL, &h.Title, &h.Domain, &h.VisitedAt, &h.Duration)
	return h, err
}

// ScanAll reads every row from a Columns query, returning the first scan
// error encountered (matching the sibling bookmark package).
func ScanAll(rows *sql.Rows) ([]models.History, error) {
	defer rows.Close()
	entries := make([]models.History, 0)
	for rows.Next() {
		h, err := Scan(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetByID loads one history entry by primary key.
func GetByID(ctx context.Context, id int) (models.History, error) {
	h, err := Scan(db.HistoryDB.QueryRowContext(ctx,
		"SELECT "+Columns+" FROM history WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return h, ErrNotFound
	}
	return h, err
}

// Create inserts a history entry, defaulting visited_at to now and deriving
// the domain from the URL (entry.Domain is accepted on the input model for
// API serialisation but is ignored — the server always derives it from the
// URL for consistency). It returns the new row id.
func Create(ctx context.Context, entry models.History) (int64, error) {
	if entry.VisitedAt.IsZero() {
		entry.VisitedAt = time.Now()
	}
	result, err := db.HistoryDB.ExecContext(ctx,
		"INSERT INTO history (user_id, url, domain, title, visited_at, duration) VALUES (?, ?, ?, ?, ?, ?)",
		entry.UserID, entry.URL, helpers.URLHostname(entry.URL), entry.Title, entry.VisitedAt, entry.Duration)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Delete removes a history entry and reports whether a row matched.
func Delete(ctx context.Context, id int) (bool, error) {
	result, err := db.HistoryDB.ExecContext(ctx, "DELETE FROM history WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

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

// ExistingURLs returns the set of URLs already recorded for a user, used by
// the importer to skip duplicates.
func ExistingURLs(ctx context.Context, userID int) (map[string]struct{}, error) {
	urls := make(map[string]struct{})
	rows, err := db.HistoryDB.QueryContext(ctx, "SELECT url FROM history WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls[url] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

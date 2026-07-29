package bookmark

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

type rowScanner interface{ Scan(...any) error }

func Scan(scanner rowScanner) (models.Bookmark, error) {
	var b models.Bookmark
	err := scanner.Scan(
		&b.ID, &b.UserID, &b.Title, &b.URL, &b.Description,
		&b.Tags, &b.FolderPath, &b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

func ScanAll(rows *sql.Rows) ([]models.Bookmark, error) {
	defer rows.Close()
	bookmarks := make([]models.Bookmark, 0)
	for rows.Next() {
		b, err := Scan(rows)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bookmarks, nil
}

func GetByID(ctx context.Context, id int) (models.Bookmark, error) {
	b, err := Scan(db.BookmarkDB.QueryRowContext(ctx,
		"SELECT "+Columns+" FROM bookmarks WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

func GetByCaptureID(ctx context.Context, userID int, captureID string) (models.Bookmark, error) {
	b, err := Scan(db.BookmarkDB.QueryRowContext(ctx,
		"SELECT "+Columns+" FROM bookmarks WHERE user_id = ? AND capture_id = ?", userID, captureID))
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

type ListOptions struct {
	UserID           int
	TagsFilter       string
	FolderPathPrefix string
	Limit            int
}

func List(ctx context.Context, options ListOptions) ([]models.Bookmark, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 3)
	if options.UserID > 0 {
		where = append(where, "user_id = ?")
		args = append(args, options.UserID)
	}
	if options.FolderPathPrefix != "" {
		where = append(where, "folder_path LIKE ?")
		args = append(args, options.FolderPathPrefix+"%")
	}

	query := "SELECT " + Columns + " FROM bookmarks WHERE " + strings.Join(where, " AND ") + " ORDER BY updated_at DESC"
	if options.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, options.Limit)
	}
	rows, err := db.BookmarkDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	bookmarks, err := ScanAll(rows)
	if err != nil {
		return nil, err
	}
	if options.TagsFilter == "" {
		return bookmarks, nil
	}
	filtered := make([]models.Bookmark, 0, len(bookmarks))
	for _, b := range bookmarks {
		if MatchesAnyTag(helpers.ParseTagsFromJSON(b.Tags), options.TagsFilter) {
			filtered = append(filtered, b)
		}
	}
	return filtered, nil
}

type CreateInput struct {
	UserID      int
	Title       string
	URL         string
	Description string
	FolderPath  string
	CaptureID   string
	Tags        []string
}

func (in CreateInput) tagsJSON() string { return helpers.TagsToJSON(in.Tags) }

func Create(ctx context.Context, in CreateInput) (id int64, inserted bool, err error) {
	result, err := db.BookmarkDB.ExecContext(ctx, `
		INSERT INTO bookmarks (user_id, title, url, description, tags, folder_path, capture_id)
		VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''))
		ON CONFLICT(user_id, capture_id) DO NOTHING`,
		in.UserID, in.Title, in.URL, in.Description, in.tagsJSON(), in.FolderPath, in.CaptureID)
	if err != nil {
		return 0, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	if rows == 0 {
		return 0, false, nil
	}
	id, err = result.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func Update(ctx context.Context, id int, in CreateInput) error {
	result, err := db.BookmarkDB.ExecContext(ctx, `
		UPDATE bookmarks SET user_id = ?, title = ?, url = ?, description = ?, tags = ?, folder_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		in.UserID, in.Title, in.URL, in.Description, in.tagsJSON(), in.FolderPath, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBookmarkNotFound
	}
	return nil
}

func Delete(ctx context.Context, id int) (bool, error) {
	result, err := db.BookmarkDB.ExecContext(ctx, "DELETE FROM bookmarks WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func ExistingURLs(ctx context.Context, userID int) (map[string]struct{}, error) {
	urls := make(map[string]struct{})
	rows, err := db.BookmarkDB.QueryContext(ctx, "SELECT url FROM bookmarks WHERE user_id = ?", userID)
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

func Search(ctx context.Context, userID int, query string, limit int) ([]models.Bookmark, error) {
	where := []string{"user_id = ?"}
	args := []any{userID}
	if query != "" {
		like := "%" + query + "%"
		where = append(where, "(title LIKE ? OR url LIKE ? OR description LIKE ?)")
		args = append(args, like, like, like)
	}
	args = append(args, limit)
	rows, err := db.BookmarkDB.QueryContext(ctx,
		"SELECT "+Columns+" FROM bookmarks WHERE "+strings.Join(where, " AND ")+" ORDER BY updated_at DESC LIMIT ?",
		args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

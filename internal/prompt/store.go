package prompt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

// Columns is the canonical column list for prompt SELECT queries.
const Columns = "id, user_id, title, content, description, tags, pinned, created_at, updated_at"

var (
	// ErrNotFound is returned when a prompt does not exist.
	ErrNotFound = errors.New("not found")

	// ErrPromptNotFound and ErrPromptNotOwned describe prompt ownership failures,
	// so callers can map them to the right HTTP status.
	ErrPromptNotFound = errors.New("Prompt not found")
	ErrPromptNotOwned = errors.New("Prompt does not belong to user")
)

// Record couples a prompt row with its decoded tags.
type Record struct {
	Prompt models.Prompt
}

// Tags returns the decoded tag list for the record.
func (r Record) Tags() []string { return helpers.ParseTagsFromJSON(r.Prompt.Tags) }

type rowScanner interface{ Scan(...any) error }

// Scan reads a row selected with Columns into a Record.
func Scan(scanner rowScanner) (Record, error) {
	var rec Record
	err := scanner.Scan(
		&rec.Prompt.ID, &rec.Prompt.UserID, &rec.Prompt.Title,
		&rec.Prompt.Content, &rec.Prompt.Description, &rec.Prompt.Tags,
		&rec.Prompt.Pinned, &rec.Prompt.CreatedAt, &rec.Prompt.UpdatedAt,
	)
	return rec, err
}

// ScanAll reads every row into Records.
func ScanAll(rows *sql.Rows) ([]Record, error) {
	defer rows.Close()
	records := make([]Record, 0)
	for rows.Next() {
		rec, err := Scan(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// GetByID loads one prompt by id.
func GetByID(ctx context.Context, id int) (Record, error) {
	rec, err := Scan(db.PromptDB.QueryRowContext(ctx,
		"SELECT "+Columns+" FROM prompts WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return rec, ErrNotFound
	}
	return rec, err
}

// ListQuery describes a prompt listing or search.
type ListQuery struct {
	UserID int
	// Query filters on title and content when set.
	Query string
	// Limit caps the result count when greater than zero.
	Limit int
	// TitleFirst orders title matches ahead of content-only matches.
	TitleFirst bool
}

// List runs a prompt listing or search and returns the matching records.
func List(ctx context.Context, q ListQuery) ([]Record, error) {
	where := []string{"user_id = ?"}
	args := []any{q.UserID}

	if q.Query != "" {
		where = append(where, "(title LIKE ? OR content LIKE ?)")
		like := "%" + q.Query + "%"
		args = append(args, like, like)
	}

	sqlQuery := "SELECT " + Columns + " FROM prompts WHERE " + strings.Join(where, " AND ")

	if q.TitleFirst {
		sqlQuery += " ORDER BY CASE WHEN title LIKE ? THEN 0 ELSE 1 END, created_at DESC"
		args = append(args, "%"+q.Query+"%")
	} else {
		sqlQuery += " ORDER BY created_at DESC"
	}

	if q.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, q.Limit)
	}

	rows, err := db.PromptDB.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

// VerifyOwnership checks that a prompt exists and belongs to the user.
func VerifyOwnership(ctx context.Context, id, userID int) error {
	var ownerID int
	err := db.PromptDB.QueryRowContext(ctx,
		"SELECT user_id FROM prompts WHERE id = ?", id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPromptNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrPromptNotOwned
	}
	return nil
}

// CreateInput describes a prompt to insert.
type CreateInput struct {
	UserID      int
	Title       string
	Content     string
	Description string
	Tags        []string
	Pinned      bool

	// CreatedAt sets the creation timestamp. It defaults to time.Now().
	// Listings order by created_at, so this is deliberately stored with
	// sub-second precision: SQL's CURRENT_TIMESTAMP only resolves to whole
	// seconds, which makes the order of prompts created in the same second
	// non-deterministic.
	CreatedAt time.Time
}

// Create inserts a prompt and returns its new id and stored tags JSON.
func Create(ctx context.Context, in CreateInput) (int64, string, error) {
	tagsJSON := helpers.TagsToJSON(in.Tags)
	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	result, err := db.PromptDB.ExecContext(ctx, `
		INSERT INTO prompts (user_id, title, content, description, tags, pinned, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.UserID, in.Title, in.Content, in.Description, tagsJSON, in.Pinned, createdAt, createdAt)
	if err != nil {
		return 0, tagsJSON, err
	}
	id, _ := result.LastInsertId()
	return id, tagsJSON, nil
}

// UpdateBuilder accumulates the SET clauses of a partial prompt update.
type UpdateBuilder struct {
	clauses []string
	args    []any
}

// NewUpdateBuilder returns an empty builder.
func NewUpdateBuilder() *UpdateBuilder { return &UpdateBuilder{} }

// Set records "column = ?" with its bound value.
func (b *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	b.clauses = append(b.clauses, column+" = ?")
	b.args = append(b.args, value)
	return b
}

// Empty reports whether no fields were set.
func (b *UpdateBuilder) Empty() bool { return len(b.clauses) == 0 }

// Exec applies the update to one prompt, always refreshing updated_at.
func (b *UpdateBuilder) Exec(ctx context.Context, id int) error {
	if b.Empty() {
		return fmt.Errorf("no updatable fields provided")
	}
	args := append(append([]any{}, b.args...), time.Now(), id)
	_, err := db.PromptDB.ExecContext(ctx,
		"UPDATE prompts SET "+strings.Join(b.clauses, ", ")+", updated_at = ? WHERE id = ?",
		args...)
	return err
}

// Delete removes a prompt and reports whether a row matched.
func Delete(ctx context.Context, id int) (bool, error) {
	result, err := db.PromptDB.ExecContext(ctx, "DELETE FROM prompts WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// GetDistinctTags returns the distinct tag values for a user's prompts.
func GetDistinctTags(ctx context.Context, userID int) ([]string, error) {
	const query = `SELECT DISTINCT json_each.value
		FROM prompts, json_each(prompts.tags)
		WHERE prompts.user_id = ?
		  AND json_each.value IS NOT NULL
		  AND json_each.value != ''
		ORDER BY json_each.value
		LIMIT 500`

	rows, err := db.PromptDB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

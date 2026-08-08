package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

// Columns is the canonical column list for question SELECT queries.
const Columns = "id, user_id, type, difficulty, question, explanation, options_json, answer_json, image_filename, tags, subject, topic, sub_topic, source, created_at, updated_at"

var (
	// ErrNotFound is returned when a question does not exist.
	ErrNotFound = errors.New("not found")

	// ErrQuestionNotFound and ErrQuestionNotOwned describe ownership failures,
	// so callers can map them to the right HTTP status.
	ErrQuestionNotFound = errors.New("Question not found")
	ErrQuestionNotOwned = errors.New("Question does not belong to user")

	// ErrPaperNotFound is returned when a paper does not exist.
	ErrPaperNotFound = errors.New("Paper not found")
)

// Record couples a question row with its decoded options/answers helpers.
type Record struct {
	Question models.Question
}

// Options returns the decoded option list for choice question types.
func (r Record) Options() []models.QuestionOption {
	if r.Question.Options == "" {
		return []models.QuestionOption{}
	}
	options := []models.QuestionOption{}
	if err := json.Unmarshal([]byte(r.Question.Options), &options); err != nil {
		return []models.QuestionOption{}
	}
	return options
}

// CorrectAnswers returns the indexes of options marked correct.
func (r Record) CorrectAnswers() []int {
	out := []int{}
	for _, opt := range r.Options() {
		if opt.Correct {
			out = append(out, opt.Index)
		}
	}
	return out
}

// ExpectedText returns the stored expected answer text for input questions.
func (r Record) ExpectedText() string {
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(r.Question.Answer), &payload); err != nil {
		return ""
	}
	return payload.Text
}

// ChronologyItems returns the decoded chronology items, ordered by CorrectOrder.
func (r Record) ChronologyItems() []models.ChronologyItem {
	if r.Question.Options == "" {
		return []models.ChronologyItem{}
	}
	items := []models.ChronologyItem{}
	if err := json.Unmarshal([]byte(r.Question.Options), &items); err != nil {
		return []models.ChronologyItem{}
	}
	return items
}

// Tags returns the decoded tag list (e.g. exam names) for the question.
// An empty or invalid JSON column yields an empty slice (never nil) so the
// value remains JSON-serializable as `[]`.
func (r Record) Tags() []string {
	if r.Question.Tags == "" {
		return []string{}
	}
	tags := []string{}
	if err := json.Unmarshal([]byte(r.Question.Tags), &tags); err != nil {
		return []string{}
	}
	return tags
}

// TagText joins the tag columns for fuzzy-search weighting.
func (r Record) TagText() string {
	tags := r.Tags()
	return strings.Join(append(tags, r.Question.Subject, r.Question.Topic, r.Question.SubTopic), " ")
}

type rowScanner interface{ Scan(...any) error }

// Scan reads a row selected with Columns into a Record.
func Scan(scanner rowScanner) (Record, error) {
	var rec Record
	err := scanner.Scan(
		&rec.Question.ID, &rec.Question.UserID, &rec.Question.Type,
		&rec.Question.Difficulty, &rec.Question.Question, &rec.Question.Explanation,
		&rec.Question.Options, &rec.Question.Answer, &rec.Question.ImageFilename,
		&rec.Question.Tags, &rec.Question.Subject, &rec.Question.Topic,
		&rec.Question.SubTopic, &rec.Question.Source,
		&rec.Question.CreatedAt, &rec.Question.UpdatedAt,
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

// GetByID loads one question by id.
func GetByID(ctx context.Context, id int) (Record, error) {
	rec, err := Scan(db.QuizDB.QueryRowContext(ctx,
		"SELECT "+Columns+" FROM questions WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return rec, ErrNotFound
	}
	return rec, err
}

// Filter is the set of column filters shared by question listing, fuzzy
// search candidate loading, and paper section sampling.
type Filter struct {
	UserID     int
	Type       string
	Difficulty string
	Tags       []string // match questions whose tags array contains ANY of these
	Subject    string
	Topic      string
	SubTopic   string
}

// Where renders the filter as a SQL predicate plus its bound arguments. The
// user scope is always applied, so the predicate is never empty.
func (f Filter) Where() (string, []any) {
	where := []string{"user_id = ?"}
	args := []any{f.UserID}

	for _, eq := range []struct{ column, value string }{
		{"type", f.Type},
		{"difficulty", f.Difficulty},
		{"subject", f.Subject},
		{"topic", f.Topic},
		{"sub_topic", f.SubTopic},
	} {
		if eq.value != "" {
			where = append(where, eq.column+" = ?")
			args = append(args, eq.value)
		}
	}

	if len(f.Tags) > 0 {
		// Match any question whose JSON-array `tags` column contains at
		// least one of the requested values. The IN list is bound to
		// positional params so the values are matched as JSON strings.
		placeholders := make([]string, len(f.Tags))
		for i, t := range f.Tags {
			placeholders[i] = "?"
			args = append(args, t)
		}
		where = append(where,
			"EXISTS (SELECT 1 FROM json_each(tags) WHERE json_each.value IN ("+
				strings.Join(placeholders, ",")+"))")
	}

	return strings.Join(where, " AND "), args
}

// ListQuery describes a question listing.
type ListQuery struct {
	Filter
	Query  string
	Limit  int
	Offset int
}

// List returns the matching questions ordered newest-first.
func List(ctx context.Context, q ListQuery) ([]Record, error) {
	predicate, args := q.Filter.Where()

	if q.Query != "" {
		predicate += " AND (question LIKE ? OR explanation LIKE ?)"
		like := "%" + q.Query + "%"
		args = append(args, like, like)
	}

	sqlQuery := "SELECT " + Columns + " FROM questions WHERE " + predicate +
		" ORDER BY created_at DESC, id DESC"
	if q.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, q.Limit)
		if q.Offset > 0 {
			sqlQuery += " OFFSET ?"
			args = append(args, q.Offset)
		}
	}

	rows, err := db.QuizDB.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

// VerifyOwnership checks that a question exists and belongs to the user.
func VerifyOwnership(ctx context.Context, id, userID int) error {
	var ownerID int
	err := db.QuizDB.QueryRowContext(ctx,
		"SELECT user_id FROM questions WHERE id = ?", id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrQuestionNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrQuestionNotOwned
	}
	return nil
}

// CreateInput describes a question to insert. Options and Answer are stored
// as raw JSON strings (matching the bookmark-tag pattern); callers marshal
// decoded shapes into them. Tags is marshaled to a JSON array internally.
type CreateInput struct {
	UserID        int
	Type          string
	Difficulty    string
	Question      string
	Explanation   string
	Options       string // JSON array of QuestionOption or ChronologyItem
	Answer        string // JSON shape; [] for choice types, {"text":...} for input
	ImageFilename string
	Tags          []string
	Subject       string
	Topic         string
	SubTopic      string
	Source        string

	// CreatedAt sets the creation timestamp. It defaults to time.Now().
	CreatedAt time.Time
}

// EncodeTags marshals a tag list into the JSON-array form stored in the
// `tags` column. A nil slice yields "[]" so the column never holds NULL.
func EncodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	out, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(out)
}

// Create inserts a question and returns its new id.
func Create(ctx context.Context, in CreateInput) (int64, error) {
	options := in.Options
	if options == "" {
		options = "[]"
	}
	answer := in.Answer
	if answer == "" {
		answer = "[]"
	}
	difficulty := in.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}
	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	result, err := db.QuizDB.ExecContext(ctx, `
		INSERT INTO questions (user_id, type, difficulty, question, explanation, options_json, answer_json, image_filename, tags, subject, topic, sub_topic, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.UserID, in.Type, difficulty, in.Question, in.Explanation, options, answer,
		in.ImageFilename, EncodeTags(in.Tags), in.Subject, in.Topic, in.SubTopic, in.Source,
		createdAt, createdAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateBuilder accumulates the SET clauses of a partial question update.
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

// Exec applies the update to one question, always refreshing updated_at.
func (b *UpdateBuilder) Exec(ctx context.Context, id int) error {
	if b.Empty() {
		return fmt.Errorf("no updatable fields provided")
	}
	args := append(append([]any{}, b.args...), time.Now(), id)
	_, err := db.QuizDB.ExecContext(ctx,
		"UPDATE questions SET "+strings.Join(b.clauses, ", ")+", updated_at = ? WHERE id = ?",
		args...)
	return err
}

// Delete removes a question and reports whether a row matched.
func Delete(ctx context.Context, id int) (bool, error) {
	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM question_review_state WHERE question_id = ?", id); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM questions WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// ─── Images ─────────────────────────────────────────────

// DeleteImageRecord removes the question_images row for one filename.
// Best-effort cleanup after a replacement upload.
func DeleteImageRecord(ctx context.Context, questionID int, filename string) error {
	_, err := db.QuizDB.ExecContext(ctx,
		"DELETE FROM question_images WHERE question_id = ? AND filename = ?", questionID, filename)
	return err
}

// DeleteImageRecords removes every question_images row for a question.
func DeleteImageRecords(ctx context.Context, questionID int) error {
	_, err := db.QuizDB.ExecContext(ctx,
		"DELETE FROM question_images WHERE question_id = ?", questionID)
	return err
}

// SetImageFilename records the uploaded image filename on the question and
// inserts a question_images row.
func SetImageFilename(ctx context.Context, id int, filename string) error {
	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		"UPDATE questions SET image_filename = ?, updated_at = ? WHERE id = ?",
		filename, time.Now(), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO question_images (question_id, filename) VALUES (?, ?)",
		id, filename); err != nil {
		return err
	}
	return tx.Commit()
}

// ─── Tag vocabulary & stats ─────────────────────────────

// queryStrings runs a single-column query and collects the values.
func queryStrings(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := db.QuizDB.QueryContext(ctx, query, args...)
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

// queryCounts runs a (key, count) query and collects it into a map, skipping
// blank keys.
func queryCounts(ctx context.Context, query string, args ...any) (map[string]int, error) {
	rows, err := db.QuizDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		if k != "" {
			out[k] = n
		}
	}
	return out, rows.Err()
}

// distinctTagsQuery explodes the JSON-array `tags` column with json_each so
// the individual tag strings can be aggregated.
const distinctTagsQuery = `SELECT DISTINCT json_each.value
	   FROM questions, json_each(questions.tags)
	  WHERE questions.user_id = ?
	    AND json_each.value IS NOT NULL
	    AND json_each.value != ''
	  ORDER BY json_each.value`

// TagVocabulary returns the distinct values of the four tag-bearing columns
// for a user. Tags are flattened out of the JSON-array column so callers
// see a flat list of unique tag strings.
func TagVocabulary(ctx context.Context, userID int) (models.TagVocabulary, error) {
	var out models.TagVocabulary

	tags, err := queryStrings(ctx, distinctTagsQuery, userID)
	if err != nil {
		return out, err
	}
	out.Tags = tags

	for _, c := range []struct {
		target *[]string
		column string
	}{
		{&out.Subjects, "subject"},
		{&out.Topics, "topic"},
		{&out.SubTopics, "sub_topic"},
	} {
		values, err := queryStrings(ctx,
			"SELECT DISTINCT "+c.column+" FROM questions WHERE user_id = ? AND "+c.column+" != '' ORDER BY "+c.column,
			userID)
		if err != nil {
			return out, err
		}
		*c.target = values
	}
	return out, nil
}

// Stats aggregates question counts for the dashboard.
func Stats(ctx context.Context, userID int) (models.QuizStats, error) {
	var out models.QuizStats

	if err := db.QuizDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM questions WHERE user_id = ?", userID).Scan(&out.Total); err != nil {
		return out, err
	}

	grouped := func(column string) (map[string]int, error) {
		return queryCounts(ctx,
			"SELECT "+column+", COUNT(*) FROM questions WHERE user_id = ? GROUP BY "+column+" ORDER BY COUNT(*) DESC",
			userID)
	}

	byType, err := grouped("type")
	if err != nil {
		return out, err
	}
	byDifficulty, err := grouped("difficulty")
	if err != nil {
		return out, err
	}

	// Tags live in a JSON array; explode with json_each and count distinct
	// questions per tag value so multi-tagged questions don't inflate counts.
	byTags, err := queryCounts(ctx, `SELECT json_each.value, COUNT(DISTINCT questions.id)
		   FROM questions, json_each(questions.tags)
		  WHERE questions.user_id = ?
		    AND json_each.value IS NOT NULL
		    AND json_each.value != ''
		  GROUP BY json_each.value
		  ORDER BY 2 DESC`, userID)
	if err != nil {
		return out, err
	}

	if err := db.QuizDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM question_papers WHERE user_id = ?", userID).Scan(&out.PaperCount); err != nil {
		return out, err
	}

	out.ByType = byType
	out.ByDifficulty = byDifficulty
	out.ByTags = byTags
	return out, nil
}

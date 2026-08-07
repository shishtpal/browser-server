package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

// GenerateSectionedPaper samples count questions per section (ORDER BY
// RANDOM() LIMIT n) and enforces the no-duplicates policy across sections.
// The maxQuestions cap comes from the operator's configured limits.
//
// Validation failures are returned as *FieldError so callers can distinguish
// a bad request from a database failure.
func GenerateSectionedPaper(ctx context.Context, userID int, sections []models.QuestionPaperSection, allowDuplicates bool, maxQuestions int) ([]Record, error) {
	seen := map[int]bool{}
	out := make([]Record, 0)
	for i, section := range sections {
		if section.Count < 1 {
			return nil, fieldErrorf("sections", "sections[%d].count must be at least 1", i)
		}
		records, err := sampleSection(ctx, userID, section)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			if !allowDuplicates && seen[rec.Question.ID] {
				continue
			}
			seen[rec.Question.ID] = true
			out = append(out, rec)
		}
	}
	if maxQuestions > 0 && len(out) > maxQuestions {
		return nil, fieldErrorf("sections", "paper would exceed %d questions", maxQuestions)
	}
	return out, nil
}

func sampleSection(ctx context.Context, userID int, s models.QuestionPaperSection) ([]Record, error) {
	predicate, args := Filter{
		UserID:     userID,
		Type:       s.Type,
		Difficulty: s.Difficulty,
		Tags:       s.Tags,
		Subject:    s.Subject,
		Topic:      s.Topic,
		SubTopic:   s.SubTopic,
	}.Where()

	query := "SELECT " + Columns + " FROM questions WHERE " + predicate +
		" ORDER BY RANDOM() LIMIT ?"
	args = append(args, s.Count)

	rows, err := db.QuizDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

// PersistPaper inserts the paper row and its question_paper_items rows in one
// transaction, returning the new paper id.
func PersistPaper(ctx context.Context, userID int, title string, sections []models.QuestionPaperSection, records []Record) (int64, error) {
	sectionsJSON, err := json.Marshal(sections)
	if err != nil {
		return 0, err
	}

	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO question_papers (user_id, title, sections_json, question_count, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		userID, title, string(sectionsJSON), len(records), time.Now())
	if err != nil {
		return 0, err
	}
	paperID, _ := result.LastInsertId()

	for pos, rec := range records {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO question_paper_items (paper_id, question_id, position) VALUES (?, ?, ?)",
			paperID, rec.Question.ID, pos); err != nil {
			return 0, err
		}
	}
	return paperID, tx.Commit()
}

// ListPapers returns a user's papers, newest first, without question payloads.
func ListPapers(ctx context.Context, userID, limit, offset int) ([]models.QuestionPaper, error) {
	query := "SELECT id, user_id, title, sections_json, question_count, created_at FROM question_papers WHERE user_id = ? ORDER BY created_at DESC, id DESC"
	args := []any{userID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}
	rows, err := db.QuizDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	papers := make([]models.QuestionPaper, 0)
	for rows.Next() {
		var p models.QuestionPaper
		var sectionsJSON string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &sectionsJSON, &p.QuestionCount, &p.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(sectionsJSON), &p.Sections); err != nil {
			p.Sections = []models.QuestionPaperSection{}
		}
		papers = append(papers, p)
	}
	return papers, rows.Err()
}

// GetPaperByID loads one paper with its full question list in order.
func GetPaperByID(ctx context.Context, id int) (models.QuestionPaper, error) {
	var p models.QuestionPaper
	var sectionsJSON string
	err := db.QuizDB.QueryRowContext(ctx,
		"SELECT id, user_id, title, sections_json, question_count, created_at FROM question_papers WHERE id = ?", id).
		Scan(&p.ID, &p.UserID, &p.Title, &sectionsJSON, &p.QuestionCount, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrPaperNotFound
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal([]byte(sectionsJSON), &p.Sections); err != nil {
		p.Sections = []models.QuestionPaperSection{}
	}

	rows, err := db.QuizDB.QueryContext(ctx, `
		SELECT `+Columns+` FROM questions
		JOIN question_paper_items ON question_paper_items.question_id = questions.id
		WHERE question_paper_items.paper_id = ?
		ORDER BY question_paper_items.position`, id)
	if err != nil {
		return p, err
	}
	records, err := ScanAll(rows)
	if err != nil {
		return p, err
	}
	p.Questions = Responses(records)
	return p, nil
}

// DeletePaper removes a paper; question_paper_items rows cascade via FK.
func DeletePaper(ctx context.Context, id int) (bool, error) {
	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Delete items explicitly: the connection is opened without
	// foreign_keys=ON, so the ON DELETE CASCADE is not enforced. (A
	// "PRAGMA foreign_keys" issued inside a transaction is a no-op in
	// SQLite, so it cannot be turned on here either.)
	if _, err := tx.ExecContext(ctx, "DELETE FROM question_paper_items WHERE paper_id = ?", id); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM question_papers WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// CountPapers reports how many papers a user owns.
func CountPapers(ctx context.Context, userID int) (int, error) {
	var count int
	err := db.QuizDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM question_papers WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

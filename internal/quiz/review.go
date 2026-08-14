package quiz

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

const (
	RatingAgain = "again"
	RatingHard  = "hard"
	RatingGood  = "good"
	RatingEasy  = "easy"

	// CardStatusNew marks a question that has never been reviewed.
	CardStatusNew = "new"
	// CardStatusDue marks a question whose review interval has elapsed.
	CardStatusDue       = "due"
	CardStatusScheduled = "scheduled"

	minEaseFactor = 1.3
	maxEaseFactor = 3.0
	maxInterval   = 365 * 24 * time.Hour
)

// ReviewState is the persisted scheduling state for one question. The three
// tail fields hold FSRS state and stay zero for rows the SM-2 engine owns.
//
// LearningStep uses FSRS semantics: LearnStepLearning (0) means the card is
// sitting on the 10-minute Again requeue; LearnStepReview (1) is graduated.
type ReviewState struct {
	QuestionID      int
	UserID          int
	Repetitions     int
	IntervalSeconds int64
	EaseFactor      float64
	DueAt           time.Time
	LastRating      string
	LastReviewedAt  time.Time
	SkipCount       int
	LastSkippedAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Difficulty      float64
	Stability       float64
	LearningStep    int
}

func clampEase(value float64) float64 { return math.Max(minEaseFactor, math.Min(maxEaseFactor, value)) }

func clampInterval(interval time.Duration) time.Duration {
	if interval > maxInterval {
		return maxInterval
	}
	if interval < time.Second {
		return time.Second
	}
	return interval.Round(time.Second)
}

// ScheduleReview applies the deterministic first-version spaced repetition
// rules. All timestamps are normalized to UTC: due_at is compared as TEXT by
// SQLite, so a non-UTC write would sort incorrectly against existing rows.
func ScheduleReview(previous *ReviewState, rating string, now time.Time) ReviewState {
	state := ReviewState{EaseFactor: 2.5}
	if previous != nil {
		state = *previous
		if state.EaseFactor == 0 {
			state.EaseFactor = 2.5
		}
	}
	prior := time.Duration(state.IntervalSeconds) * time.Second
	newOrReset := previous == nil || state.Repetitions == 0
	var interval time.Duration
	switch rating {
	case RatingAgain:
		state.Repetitions = 0
		state.EaseFactor = clampEase(state.EaseFactor - .20)
		interval = 10 * time.Minute
	case RatingHard:
		state.Repetitions++
		state.EaseFactor = clampEase(state.EaseFactor - .15)
		if newOrReset {
			interval = 12 * time.Hour
		} else {
			interval = time.Duration(float64(prior) * 1.2)
			if interval < 12*time.Hour {
				interval = 12 * time.Hour
			}
		}
	case RatingGood:
		state.Repetitions++
		if newOrReset {
			interval = 24 * time.Hour
		} else if state.Repetitions == 2 {
			interval = 3 * 24 * time.Hour
		} else {
			interval = time.Duration(float64(prior) * state.EaseFactor)
		}
	case RatingEasy:
		state.Repetitions++
		state.EaseFactor = clampEase(state.EaseFactor + .15)
		if newOrReset {
			interval = 4 * 24 * time.Hour
		} else {
			interval = time.Duration(float64(prior) * state.EaseFactor * 1.3)
		}
	default:
		// Leave scheduling untouched rather than letting a zero interval
		// fall through to clampInterval and reschedule one second out.
		return state
	}
	interval = clampInterval(interval)
	state.IntervalSeconds = int64(interval / time.Second)
	state.DueAt = now.UTC().Add(interval)
	state.LastReviewedAt = now.UTC()
	state.LastRating = rating
	return state
}

func scanReview(scanner interface{ Scan(...any) error }) (ReviewState, error) {
	var s ReviewState
	var difficulty, stability sql.NullFloat64
	var learningStep sql.NullInt64
	err := scanner.Scan(&s.QuestionID, &s.UserID, &s.Repetitions, &s.IntervalSeconds, &s.EaseFactor, &s.DueAt, &s.LastRating, &s.LastReviewedAt, &s.SkipCount, &s.LastSkippedAt, &s.CreatedAt, &s.UpdatedAt, &difficulty, &stability, &learningStep)
	s.Difficulty = difficulty.Float64
	s.Stability = stability.Float64
	s.LearningStep = int(learningStep.Int64)
	return s, err
}

func reviewResponse(s ReviewState) models.QuestionReviewState {
	return models.QuestionReviewState{QuestionID: s.QuestionID, Repetitions: s.Repetitions, IntervalSeconds: s.IntervalSeconds, EaseFactor: s.EaseFactor, DueAt: s.DueAt, LastRating: s.LastRating, LastReviewedAt: s.LastReviewedAt, SkipCount: s.SkipCount, LastSkippedAt: s.LastSkippedAt}
}

const reviewColumns = "question_id, user_id, repetitions, interval_seconds, ease_factor, due_at, last_rating, last_reviewed_at, skip_count, last_skipped_at, created_at, updated_at, difficulty, stability, learning_step"

// qualifiedQuestionColumns prefixes each Columns entry with a table alias.
// Columns must stay a plain comma-separated identifier list; a column holding
// an expression with a comma (e.g. COALESCE(a, b)) would split incorrectly.
func qualifiedQuestionColumns(alias string) string {
	parts := strings.Split(Columns, ", ")
	for i := range parts {
		if strings.ContainsAny(parts[i], "(),") {
			panic("quiz.Columns must be a plain identifier list, got: " + parts[i])
		}
		parts[i] = alias + "." + parts[i]
	}
	return strings.Join(parts, ", ")
}

// Practice queue modes. An empty mode selects the default "due first, then new" behavior.
const (
	CardModeNew     = "new"
	CardModeSkipped = "skipped"
	CardModeHard    = "hard"
)

func ValidateCardMode(mode string) error {
	switch mode {
	case "", CardModeNew, CardModeSkipped, CardModeHard:
		return nil
	default:
		return errors.New("mode must be one of: new, skipped, hard")
	}
}

// modeFilter returns the extra WHERE fragment selecting only the requested
// practice bucket. Hard cards come straight from the questions table; the
// skipped bucket is any persisted review row with skip history.
func modeFilter(mode string) string {
	switch mode {
	case CardModeNew:
		return " AND r.question_id IS NULL"
	case CardModeSkipped:
		return " AND r.skip_count > 0"
	case CardModeHard:
		return " AND q.difficulty = 'hard'"
	default:
		return ""
	}
}

// ListCards returns due reviews followed by never-reviewed questions. Tags reuse Filter.Where via a matching-ID subquery.
func ListCards(ctx context.Context, userID int, tags []string, limit int, now time.Time, practice bool, mode string, scheduler string) (models.QuestionCardQueue, error) {
	var out models.QuestionCardQueue
	predicate, args := (Filter{UserID: userID, Tags: tags}).Where()
	base := "q.id IN (SELECT id FROM questions WHERE " + predicate + ")"
	// Join is scoped to the requesting user so one user cannot see another's review state.
	joinClause := "LEFT JOIN question_review_state r ON r.question_id = q.id AND r.user_id = ?"
	countSQL := "SELECT " +
		"COALESCE(SUM(CASE WHEN r.question_id IS NOT NULL AND r.due_at <= ? THEN 1 ELSE 0 END), 0), " +
		"COALESCE(SUM(CASE WHEN r.question_id IS NULL THEN 1 ELSE 0 END), 0) " +
		"FROM questions q " + joinClause + " WHERE " + base + modeFilter(mode)
	// Bound in SQL text order: the `due_at <= ?` in the SELECT list precedes
	// the join's user scope, which precedes the filter predicate. Mode queries
	// append their own ? below, after the default because SQLite binds in text order.
	countArgs := append([]any{now.UTC(), userID}, args...)
	if mode == CardModeHard {
		countArgs = append(countArgs, now.UTC())
	}
	if err := db.QuizDB.QueryRowContext(ctx, countSQL, countArgs...).Scan(&out.DueCount, &out.NewCount); err != nil {
		return out, err
	}
	out.AvailableCount = out.DueCount + out.NewCount
	// A practice or mode-filtered queue may consist entirely of cards whose due
	// date is still in the future, so the default due/new count cannot short-circuit it.
	if limit <= 0 || (!practice && mode == "" && out.AvailableCount == 0) {
		out.Items = []models.QuestionCardItem{}
		return out, nil
	}
	cardFilter := "(r.question_id IS NULL OR r.due_at <= ?)"
	if practice || mode != "" {
		cardFilter = "1=1"
	}
	// SM-2 keeps its historical due-group/new-group ordering. FSRS surfaces
	// cards still on the 10-minute Again requeue first so the user can flush
	// a just-failed card before fresh material.
	orderClause := "ORDER BY CASE WHEN r.question_id IS NULL THEN 1 ELSE 0 END, r.due_at ASC, q.id DESC"
	if scheduler == SchedulerFSRS {
		orderClause = "ORDER BY CASE WHEN r.question_id IS NOT NULL AND r.learning_step = ? AND r.due_at <= ? THEN 0 " +
			"WHEN r.question_id IS NOT NULL AND r.due_at <= ? THEN 1 ELSE 2 END, r.due_at ASC, q.id DESC"
	}
	query := "SELECT " + qualifiedQuestionColumns("q") + ", " +
		"r.question_id, r.user_id, r.repetitions, r.interval_seconds, r.ease_factor, r.due_at, r.last_rating, r.last_reviewed_at, r.skip_count, r.last_skipped_at, r.created_at, r.updated_at, r.difficulty, r.stability, r.learning_step " +
		"FROM questions q " + joinClause + " WHERE " + base + modeFilter(mode) +
		" AND " + cardFilter + " " + orderClause + " LIMIT ?"
	// userID for the join, then filter args, then (FSRS order-rank params) now + limit.
	queryArgs := append([]any{userID}, args...)
	if !practice && mode == "" {
		// cardFilter appears before the FSRS ordering expression in the SQL.
		queryArgs = append(queryArgs, now.UTC())
	}
	if scheduler == SchedulerFSRS {
		queryArgs = append(queryArgs, LearnStepLearning, now.UTC(), now.UTC())
	}
	queryArgs = append(queryArgs, limit)
	rows, err := db.QuizDB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Items = []models.QuestionCardItem{}
	for rows.Next() {
		var rec Record
		var reviewID, reviewUserID, repetitions, intervalSeconds, skipCount, learningStep sql.NullInt64
		var ease, difficulty, stability sql.NullFloat64
		var dueAt, lastReviewedAt, lastSkippedAt, createdAt, updatedAt sql.NullTime
		var lastRating sql.NullString
		err := rows.Scan(
			&rec.Question.ID, &rec.Question.UserID, &rec.Question.Type, &rec.Question.Difficulty,
			&rec.Question.Question, &rec.Question.Explanation, &rec.Question.Options, &rec.Question.Answer,
			&rec.Question.ImageFilename, &rec.Question.Tags, &rec.Question.Subject, &rec.Question.Topic,
			&rec.Question.SubTopic, &rec.Question.Source, &rec.Question.CreatedAt, &rec.Question.UpdatedAt,
			&reviewID, &reviewUserID, &repetitions, &intervalSeconds, &ease, &dueAt, &lastRating,
			&lastReviewedAt, &skipCount, &lastSkippedAt, &createdAt, &updatedAt, &difficulty, &stability, &learningStep,
		)
		if err != nil {
			return out, err
		}
		item := models.QuestionCardItem{Question: Response(rec), Status: CardStatusNew}
		if reviewID.Valid {
			s := ReviewState{QuestionID: int(reviewID.Int64), UserID: int(reviewUserID.Int64), Repetitions: int(repetitions.Int64), IntervalSeconds: intervalSeconds.Int64, EaseFactor: ease.Float64, DueAt: dueAt.Time, LastRating: lastRating.String, LastReviewedAt: lastReviewedAt.Time, SkipCount: int(skipCount.Int64), CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time}
			if lastSkippedAt.Valid {
				t := lastSkippedAt.Time
				s.LastSkippedAt = &t
			}
			state := reviewResponse(s)
			item.Review = &state
			item.Status = CardStatusDue
			if (practice || mode != "") && dueAt.Time.After(now.UTC()) {
				item.Status = CardStatusScheduled
			}
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

// settingsGetter reads one user_settings value. It is a var so the quiz
// package stays decoupled from the users DB; handlers wire it up at import
// time for the HTTP path.
var settingsGetter = func(_ int, _ string) (string, error) {
	return "", sql.ErrNoRows
}

var defaultScheduler atomic.Value

func init() {
	defaultScheduler.Store(SchedulerSM2)
}

// SetUserSettingsGetter installs the KV lookup used to resolve a user's
// scheduler. Called once from the handlers package init.
func SetUserSettingsGetter(fn func(userID int, key string) (string, error)) {
	settingsGetter = fn
}

// SetDefaultScheduler sets the scheduler used when a user has not stored a
// preference. Invalid values retain the safe SM-2 default.
func SetDefaultScheduler(scheduler string) {
	if scheduler == SchedulerFSRS {
		defaultScheduler.Store(SchedulerFSRS)
		return
	}
	defaultScheduler.Store(SchedulerSM2)
}

// UserSettingKeyScheduler is the user_settings key for the scheduler choice.
const UserSettingKeyScheduler = "quiz.scheduler"

// resolveSchedulerForUser reads the user's stored scheduler and falls back
// to the supplied default when the row is missing, unreadable, or unknown.
func resolveSchedulerForUser(userID int, fallback string) string {
	if v, err := settingsGetter(userID, UserSettingKeyScheduler); err == nil {
		switch v {
		case SchedulerSM2, SchedulerFSRS:
			return v
		}
	}
	return fallback
}

// SchedulerForUser resolves the effective scheduler for a user: their stored
// preference wins; the configured scheduler is the default.
func SchedulerForUser(userID int) string {
	return resolveSchedulerForUser(userID, defaultScheduler.Load().(string))
}

// ReviewQuestion verifies ownership and inserts or updates scheduling state atomically.
// scheduler selects the engine (SchedulerSM2 default); on an FSRS caller's
// first rating of an SM-2-era row, ConvertSM2ToFSRS seeds the FSRS columns.
func ReviewQuestion(ctx context.Context, questionID, userID int, rating string, now time.Time, scheduler string) (models.QuestionReviewState, error) {
	var out models.QuestionReviewState
	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var owner int
	if err := tx.QueryRowContext(ctx, "SELECT user_id FROM questions WHERE id = ?", questionID).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, ErrQuestionNotFound
		}
		return out, err
	}
	if owner != userID {
		return out, ErrQuestionNotOwned
	}
	var prior *ReviewState
	state, err := scanReview(tx.QueryRowContext(ctx, "SELECT "+reviewColumns+" FROM question_review_state WHERE question_id = ?", questionID))
	if err == nil {
		prior = &state
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	next := ScheduleReview(prior, rating, now)
	if scheduler == SchedulerFSRS {
		snapshot := prior
		if snapshot != nil && (snapshot.Stability <= 0 || snapshot.Difficulty <= 0) {
			converted := ConvertSM2ToFSRS(*snapshot)
			snapshot = &converted
		}
		next = ScheduleFSRS(snapshot, rating, now)
	}
	next.QuestionID, next.UserID = questionID, userID
	if prior == nil {
		next.CreatedAt = now.UTC()
	} else {
		next.CreatedAt = prior.CreatedAt
	}
	next.UpdatedAt = now.UTC()
	// FSRS columns stay NULL for SM-2 rows (only the FSRS engine writes
	// them); use nullable values so the upsert doesn't have to branch.
	var diffArg, stabArg sql.NullFloat64
	var stepArg sql.NullInt64
	if scheduler == SchedulerFSRS {
		diffArg = sql.NullFloat64{Float64: next.Difficulty, Valid: true}
		stabArg = sql.NullFloat64{Float64: next.Stability, Valid: true}
		stepArg = sql.NullInt64{Int64: int64(next.LearningStep), Valid: true}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_review_state (question_id, user_id, repetitions, interval_seconds, ease_factor, due_at, last_rating, last_reviewed_at, created_at, updated_at, difficulty, stability, learning_step)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(question_id) DO UPDATE SET user_id=excluded.user_id, repetitions=excluded.repetitions, interval_seconds=excluded.interval_seconds, ease_factor=excluded.ease_factor, due_at=excluded.due_at, last_rating=excluded.last_rating, last_reviewed_at=excluded.last_reviewed_at, updated_at=excluded.updated_at, difficulty=excluded.difficulty, stability=excluded.stability, learning_step=excluded.learning_step`,
		next.QuestionID, next.UserID, next.Repetitions, next.IntervalSeconds, next.EaseFactor, next.DueAt, next.LastRating, next.LastReviewedAt, next.CreatedAt, next.UpdatedAt, diffArg, stabArg, stepArg)
	if err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return reviewResponse(next), nil
}

// RecordSkip logs that the user punted a card without treating it as a
// scheduling failure. Pure-zero review-state values mean the card still
// presents as "new" to the SM-2 engine, while the skip metrics feed the
// "Only skipped" practice mode.
func RecordSkip(ctx context.Context, questionID, userID int, now time.Time) (models.QuestionReviewState, error) {
	var out models.QuestionReviewState
	tx, err := db.QuizDB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var owner int
	if err := tx.QueryRowContext(ctx, "SELECT user_id FROM questions WHERE id = ?", questionID).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, ErrQuestionNotFound
		}
		return out, err
	}
	if owner != userID {
		return out, ErrQuestionNotOwned
	}
	prior, err := scanReview(tx.QueryRowContext(ctx,
		"SELECT "+reviewColumns+" FROM question_review_state WHERE question_id = ? AND user_id = ?", questionID, userID))
	next := prior
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return out, err
		}
		next = ReviewState{
			QuestionID: questionID,
			UserID:     userID,
			EaseFactor: 2.5,
			DueAt:      now.UTC(),
			CreatedAt:  now.UTC(),
		}
	}
	next.SkipCount++
	skipped := now.UTC()
	next.LastSkippedAt = &skipped
	next.UpdatedAt = now.UTC()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO question_review_state (question_id, user_id, repetitions, interval_seconds, ease_factor, due_at, last_rating, last_reviewed_at, skip_count, last_skipped_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(question_id) DO UPDATE SET skip_count = skip_count + 1, last_skipped_at = excluded.last_skipped_at, updated_at = excluded.updated_at`,
		next.QuestionID, next.UserID, next.Repetitions, next.IntervalSeconds, next.EaseFactor, next.DueAt, next.LastRating, next.LastReviewedAt, next.SkipCount, next.LastSkippedAt, next.CreatedAt, next.UpdatedAt)
	if err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return reviewResponse(next), nil
}

func ValidateRating(rating string) error {
	switch rating {
	case RatingAgain, RatingHard, RatingGood, RatingEasy:
		return nil
	default:
		return errors.New("rating must be one of: again, hard, good, easy")
	}
}

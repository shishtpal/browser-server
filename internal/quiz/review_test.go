package quiz

import (
	"context"
	"errors"
	"testing"
	"time"

	"browser-server/internal/db"
)

var reviewNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

func TestScheduleReviewFirstReview(t *testing.T) {
	for _, tc := range []struct {
		rating string
		want   time.Duration
	}{
		{RatingAgain, 10 * time.Minute},
		{RatingHard, 12 * time.Hour},
		{RatingGood, 24 * time.Hour},
		{RatingEasy, 4 * 24 * time.Hour},
	} {
		state := ScheduleReview(nil, tc.rating, reviewNow)
		if got := time.Duration(state.IntervalSeconds) * time.Second; got != tc.want {
			t.Errorf("%s: interval = %v, want %v", tc.rating, got, tc.want)
		}
		if !state.DueAt.Equal(reviewNow.Add(tc.want)) {
			t.Errorf("%s: due_at = %v, want %v", tc.rating, state.DueAt, reviewNow.Add(tc.want))
		}
		if state.LastRating != tc.rating {
			t.Errorf("%s: last_rating = %q", tc.rating, state.LastRating)
		}
	}
}

func TestScheduleReviewAgainResetsRepetitions(t *testing.T) {
	prior := &ReviewState{
		Repetitions:     5,
		IntervalSeconds: int64(30 * 24 * time.Hour / time.Second),
		EaseFactor:      2.5,
	}
	state := ScheduleReview(prior, RatingAgain, reviewNow)
	if state.Repetitions != 0 {
		t.Fatalf("expected repetitions reset to 0, got %d", state.Repetitions)
	}
	if got := time.Duration(state.IntervalSeconds) * time.Second; got != 10*time.Minute {
		t.Fatalf("expected 10m relearn interval, got %v", got)
	}
	if state.EaseFactor != 2.3 {
		t.Fatalf("expected ease 2.3, got %v", state.EaseFactor)
	}
}

func TestScheduleReviewClampsEaseAndInterval(t *testing.T) {
	state := &ReviewState{Repetitions: 1, IntervalSeconds: 3600, EaseFactor: 1.35}
	for i := 0; i < 5; i++ {
		next := ScheduleReview(state, RatingAgain, reviewNow)
		state = &next
	}
	if state.EaseFactor != minEaseFactor {
		t.Fatalf("expected ease clamped to %v, got %v", minEaseFactor, state.EaseFactor)
	}

	huge := &ReviewState{
		Repetitions:     9,
		IntervalSeconds: int64(300 * 24 * time.Hour / time.Second),
		EaseFactor:      3.0,
	}
	capped := ScheduleReview(huge, RatingEasy, reviewNow)
	if got := time.Duration(capped.IntervalSeconds) * time.Second; got != maxInterval {
		t.Fatalf("expected interval clamped to %v, got %v", maxInterval, got)
	}
}

func TestScheduleReviewUnknownRatingIsInert(t *testing.T) {
	prior := ReviewState{Repetitions: 3, IntervalSeconds: 86400, EaseFactor: 2.5, LastRating: RatingGood}
	if state := ScheduleReview(&prior, "bogus", reviewNow); state != prior {
		t.Fatalf("unknown rating must leave state untouched, got %+v", state)
	}
}

func TestValidateRating(t *testing.T) {
	for _, ok := range []string{RatingAgain, RatingHard, RatingGood, RatingEasy} {
		if err := ValidateRating(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "AGAIN", "skip"} {
		if err := ValidateRating(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestReviewQuestionOwnership(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 2, RatingGood, reviewNow); !errors.Is(err, ErrQuestionNotOwned) {
		t.Fatalf("expected ErrQuestionNotOwned, got %v", err)
	}
	if _, err := ReviewQuestion(ctx, 9999, 1, RatingGood, reviewNow); !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}

func TestReviewQuestionUpsertPreservesCreatedAt(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow); err != nil {
		t.Fatalf("first review: %v", err)
	}
	second := reviewNow.Add(48 * time.Hour)
	state, err := ReviewQuestion(ctx, int(id), 1, RatingGood, second)
	if err != nil {
		t.Fatalf("second review: %v", err)
	}
	if state.Repetitions != 2 {
		t.Fatalf("expected repetitions 2, got %d", state.Repetitions)
	}
	if got := time.Duration(state.IntervalSeconds) * time.Second; got != 3*24*time.Hour {
		t.Fatalf("expected 3d interval at repetition 2, got %v", got)
	}

	var createdAt, updatedAt time.Time
	if err := db.QuizDB.QueryRow(
		"SELECT created_at, updated_at FROM question_review_state WHERE question_id = ?", id,
	).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if !createdAt.Equal(reviewNow) {
		t.Fatalf("created_at should stay at first review %v, got %v", reviewNow, createdAt)
	}
	if !updatedAt.Equal(second) {
		t.Fatalf("updated_at should advance to %v, got %v", second, updatedAt)
	}
}

func TestListCardsCountsDueAndNew(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	ids := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q", Tags: []string{"SSC"}})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids = append(ids, int(id))
	}
	if _, err := Create(ctx, CreateInput{UserID: 2, Type: "input", Question: "Other"}); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	// Reviewed a day ago with "again" (10m interval), so it is overdue now.
	if _, err := ReviewQuestion(ctx, ids[0], 1, RatingAgain, reviewNow.Add(-24*time.Hour)); err != nil {
		t.Fatalf("review overdue: %v", err)
	}
	// Reviewed just now with "easy" (4d interval), so it is not yet due.
	if _, err := ReviewQuestion(ctx, ids[1], 1, RatingEasy, reviewNow); err != nil {
		t.Fatalf("review future: %v", err)
	}

	queue, err := ListCards(ctx, 1, nil, 20, reviewNow, false, "")
	if err != nil {
		t.Fatalf("list cards: %v", err)
	}
	if queue.DueCount != 1 {
		t.Errorf("due_count = %d, want 1", queue.DueCount)
	}
	if queue.NewCount != 1 {
		t.Errorf("new_count = %d, want 1", queue.NewCount)
	}
	if queue.AvailableCount != 2 {
		t.Errorf("available_count = %d, want 2", queue.AvailableCount)
	}
	if len(queue.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(queue.Items))
	}
	if queue.Items[0].Status != CardStatusDue || queue.Items[0].Question.ID != ids[0] {
		t.Errorf("expected due card %d first, got id=%d status=%s",
			ids[0], queue.Items[0].Question.ID, queue.Items[0].Status)
	}
	if queue.Items[0].Review == nil {
		t.Error("due card should carry review state")
	}
	if queue.Items[1].Status != CardStatusNew || queue.Items[1].Review != nil {
		t.Errorf("expected new card second with no review state, got status=%s", queue.Items[1].Status)
	}
	for _, item := range queue.Items {
		if item.Question.UserID != 1 {
			t.Errorf("queue leaked another user's question: %+v", item.Question)
		}
	}
}

func TestListCardsTagFilterAndLimit(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	for _, tag := range []string{"SSC", "SSC", "UPSC"} {
		if _, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q", Tags: []string{tag}}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	tagged, err := ListCards(ctx, 1, []string{"SSC"}, 20, reviewNow, false, "")
	if err != nil {
		t.Fatalf("list cards by tag: %v", err)
	}
	if tagged.NewCount != 2 || len(tagged.Items) != 2 {
		t.Fatalf("tag filter: new=%d items=%d, want 2/2", tagged.NewCount, len(tagged.Items))
	}

	// Limit truncates the items but the counts still describe the whole queue.
	limited, err := ListCards(ctx, 1, nil, 1, reviewNow, false, "")
	if err != nil {
		t.Fatalf("list cards with limit: %v", err)
	}
	if len(limited.Items) != 1 {
		t.Fatalf("expected 1 item under limit, got %d", len(limited.Items))
	}
	if limited.AvailableCount != 3 {
		t.Fatalf("available_count = %d, want 3", limited.AvailableCount)
	}
}

func TestListCardsPracticeIncludesScheduledOnlySelection(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q", Tags: []string{"SSC"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingEasy, reviewNow); err != nil {
		t.Fatalf("review: %v", err)
	}

	normal, err := ListCards(ctx, 1, []string{"SSC"}, 20, reviewNow, false, "")
	if err != nil {
		t.Fatalf("list normal cards: %v", err)
	}
	if len(normal.Items) != 0 || normal.AvailableCount != 0 {
		t.Fatalf("normal queue should be empty, got items=%d available=%d", len(normal.Items), normal.AvailableCount)
	}

	practice, err := ListCards(ctx, 1, []string{"SSC"}, 20, reviewNow, true, "")
	if err != nil {
		t.Fatalf("list practice cards: %v", err)
	}
	if len(practice.Items) != 1 {
		t.Fatalf("practice queue items = %d, want 1", len(practice.Items))
	}
	if practice.Items[0].Question.ID != int(id) || practice.Items[0].Status != CardStatusScheduled {
		t.Fatalf("practice item = id %d status %q, want id %d status %q", practice.Items[0].Question.ID, practice.Items[0].Status, id, CardStatusScheduled)
	}
}

func TestRecordSkipCreatesAndIncrements(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	loc, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	state, err := RecordSkip(ctx, int(loc), 1, reviewNow)
	if err != nil {
		t.Fatalf("record skip: %v", err)
	}
	if state.SkipCount != 1 {
		t.Fatalf("skip_count = %d, want 1", state.SkipCount)
	}
	if state.LastSkippedAt == nil || !state.LastSkippedAt.Equal(reviewNow) {
		t.Fatalf("last_skipped_at = %v, want %v", state.LastSkippedAt, reviewNow)
	}
	if _, err := RecordSkip(ctx, int(loc), 1, reviewNow.Add(time.Hour)); err != nil {
		t.Fatalf("second skip: %v", err)
	}
	var skipCount int
	var lastSkippedAt time.Time
	if err := db.QuizDB.QueryRow(
		"SELECT skip_count, last_skipped_at FROM question_review_state WHERE question_id = ?", loc,
	).Scan(&skipCount, &lastSkippedAt); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if skipCount != 2 {
		t.Fatalf("skip_count = %d, want 2", skipCount)
	}
	if !lastSkippedAt.Equal(reviewNow.Add(time.Hour)) {
		t.Fatalf("last_skipped_at = %v, want %v", lastSkippedAt, reviewNow.Add(time.Hour))
	}
}

func TestRecordSkipOwnership(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	loc, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := RecordSkip(ctx, int(loc), 2, reviewNow); !errors.Is(err, ErrQuestionNotOwned) {
		t.Fatalf("expected ErrQuestionNotOwned, got %v", err)
	}
	if _, err := RecordSkip(ctx, 9999, 1, reviewNow); !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}

func TestRecordSkipKeepsSM2Scheduling(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	loc, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(loc), 1, RatingGood, reviewNow); err != nil {
		t.Fatalf("review: %v", err)
	}
	state, err := RecordSkip(ctx, int(loc), 1, reviewNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("skip: %v", err)
	}
	if state.Repetitions != 1 {
		t.Fatalf("repetitions = %d, want 1 after prior review", state.Repetitions)
	}
	if state.SkipCount != 1 {
		t.Fatalf("skip_count = %d, want 1", state.SkipCount)
	}
}

func TestListCardsSkipsMode(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		_, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q", Tags: []string{"SSC"}})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	ids := []int{}
	rows, err := db.QuizDB.Query("SELECT id FROM questions WHERE user_id = 1 ORDER BY id")
	if err != nil {
		t.Fatalf("list ids: %v", err)
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) != 2 {
		t.Fatalf("want 2 questions, got %d", len(ids))
	}
	if _, err := RecordSkip(ctx, ids[0], 1, reviewNow); err != nil {
		t.Fatalf("skip: %v", err)
	}
	queue, err := ListCards(ctx, 1, nil, 20, reviewNow, false, CardModeSkipped)
	if err != nil {
		t.Fatalf("list skipped: %v", err)
	}
	if len(queue.Items) != 1 || queue.Items[0].Question.ID != ids[0] {
		t.Fatalf("expected only skipped card, got %v", queue.Items)
	}
	if queue.Items[0].Review.SkipCount != 1 {
		t.Fatalf("review.skip_count = %d, want 1", queue.Items[0].Review.SkipCount)
	}
}

func TestListCardsModesNewAndHard(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	nuid, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "new", Difficulty: "easy"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	hardID, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "hard", Difficulty: "hard"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(hardID), 1, RatingEasy, reviewNow); err != nil {
		t.Fatalf("review hard: %v", err)
	}
	queueNew, err := ListCards(ctx, 1, nil, 20, reviewNow, false, CardModeNew)
	if err != nil {
		t.Fatalf("list new: %v", err)
	}
	if len(queueNew.Items) != 1 || queueNew.Items[0].Question.ID != int(nuid) {
		t.Fatalf("mode=new should return only the unrated question, got %v", queueNew.Items)
	}
	queueHard, err := ListCards(ctx, 1, nil, 20, reviewNow, false, CardModeHard)
	if err != nil {
		t.Fatalf("list hard: %v", err)
	}
	if len(queueHard.Items) != 1 || queueHard.Items[0].Question.ID != int(hardID) {
		t.Fatalf("mode=hard should respect difficulty, got %v", queueHard.Items)
	}
}

func TestValidateCardMode(t *testing.T) {
	for _, ok := range []string{"", CardModeNew, CardModeSkipped, CardModeHard} {
		if err := ValidateCardMode(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"all", "skip", "due"} {
		if err := ValidateCardMode(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestDeleteRemovesReviewState(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow); err != nil {
		t.Fatalf("review: %v", err)
	}
	if _, err := Delete(ctx, int(id)); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var count int
	if err := db.QuizDB.QueryRow(
		"SELECT COUNT(*) FROM question_review_state WHERE question_id = ?", id).Scan(&count); err != nil {
		t.Fatalf("count review rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected review state removed with question, got %d rows", count)
	}
}

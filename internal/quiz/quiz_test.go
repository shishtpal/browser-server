package quiz

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

func setupDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	db.InitQuizDB(dir)
	t.Cleanup(func() { db.CloseQuizDB() })
}

// ─── Validation ─────────────────────────────────────────

func TestValidateQuestionText(t *testing.T) {
	rules := DefaultRules()
	if err := rules.ValidateQuestionText(""); err == nil {
		t.Fatal("expected error for empty question")
	}
	if err := rules.ValidateQuestionText(strings.Repeat("x", DefaultMaxQuestionLength+1)); err == nil {
		t.Fatal("expected error for overlong question")
	}
	if err := rules.ValidateQuestionText("What is 2+2?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExplanation(t *testing.T) {
	rules := DefaultRules()
	if err := rules.ValidateExplanation(strings.Repeat("x", DefaultMaxExplanationLength+1)); err == nil {
		t.Fatal("expected error for overlong explanation")
	}
	if err := rules.ValidateExplanation(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEnums(t *testing.T) {
	rules := DefaultRules()
	rules.AllowedTypes = []string{"single_choice", "input"}
	rules.AllowedDifficulties = []string{"easy", "hard"}
	if err := rules.ValidateType("multiple_choice"); err == nil {
		t.Fatal("expected error for disallowed type")
	}
	if err := rules.ValidateType("single_choice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := rules.ValidateDifficulty("extreme"); err == nil {
		t.Fatal("expected error for disallowed difficulty")
	}
	if err := rules.ValidateDifficulty(""); err != nil {
		t.Fatalf("empty difficulty should be allowed: %v", err)
	}
}

func TestValidateOptions(t *testing.T) {
	rules := DefaultRules()
	valid := []models.QuestionOption{
		{Index: 0, Text: "A", Correct: true},
		{Index: 1, Text: "B"},
	}
	if err := rules.ValidateOptions(valid, "single_choice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := rules.ValidateOptions(valid, "multiple_choice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// input/chronology must not carry options
	if err := rules.ValidateOptions(valid, "input"); err == nil {
		t.Fatal("expected error for options on input type")
	}
	if err := rules.ValidateOptions(nil, "input"); err != nil {
		t.Fatalf("nil options for input should be fine: %v", err)
	}
	// too few options
	if err := rules.ValidateOptions(valid[:1], "single_choice"); err == nil {
		t.Fatal("expected error for a single option")
	}
	// no correct answer
	none := []models.QuestionOption{{Index: 0, Text: "A"}, {Index: 1, Text: "B"}}
	if err := rules.ValidateOptions(none, "single_choice"); err == nil {
		t.Fatal("expected error when no option is correct")
	}
	// single_choice with two correct
	two := []models.QuestionOption{{Index: 0, Text: "A", Correct: true}, {Index: 1, Text: "B", Correct: true}}
	if err := rules.ValidateOptions(two, "single_choice"); err == nil {
		t.Fatal("expected error for two correct options on single_choice")
	}
	if err := rules.ValidateOptions(two, "multiple_choice"); err != nil {
		t.Fatalf("two correct should be fine for multiple_choice: %v", err)
	}
	// over the per-question cap
	tooMany := make([]models.QuestionOption, DefaultMaxOptionsPerQuestion+1)
	for i := range tooMany {
		tooMany[i] = models.QuestionOption{Index: i, Text: "x"}
	}
	tooMany[0].Correct = true
	if err := rules.ValidateOptions(tooMany, "multiple_choice"); err == nil {
		t.Fatal("expected error for too many options")
	}
}

func TestValidateChronologyItems(t *testing.T) {
	valid := []models.ChronologyItem{
		{Index: 0, Text: "First", CorrectOrder: 2},
		{Index: 1, Text: "Second", CorrectOrder: 1},
	}
	rules := DefaultRules()
	if err := rules.ValidateChronologyItems(valid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := rules.ValidateChronologyItems(valid[:1]); err == nil {
		t.Fatal("expected error for a single item")
	}
	dup := []models.ChronologyItem{
		{Index: 0, Text: "A", CorrectOrder: 1},
		{Index: 1, Text: "B", CorrectOrder: 1},
	}
	if err := rules.ValidateChronologyItems(dup); err == nil {
		t.Fatal("expected error for duplicate correct_order")
	}
	outOfRange := []models.ChronologyItem{
		{Index: 0, Text: "A", CorrectOrder: 1},
		{Index: 1, Text: "B", CorrectOrder: 5},
	}
	if err := rules.ValidateChronologyItems(outOfRange); err == nil {
		t.Fatal("expected error for out-of-range correct_order")
	}
}

// ─── Store round-trips ──────────────────────────────────

func TestCreateGetRoundTrip(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, err := Create(ctx, CreateInput{
		UserID:     1,
		Type:       "single_choice",
		Difficulty: "easy",
		Question:   "What is 2+2?",
		Options:    `[{"index":0,"text":"3"},{"index":1,"text":"4","correct":true}]`,
		Answer:     "[]",
		Tags:       []string{"SSC"},
		Subject:    "Math",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec.Question.Difficulty != "easy" {
		t.Fatalf("unexpected difficulty: %v", rec.Question.Difficulty)
	}
	if got := rec.Tags(); len(got) != 1 || got[0] != "SSC" {
		t.Fatalf("unexpected tags: %v", got)
	}
	if got := rec.CorrectAnswers(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("unexpected correct answers: %v", got)
	}

	if _, err := GetByID(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateDefaultsDifficulty(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q", Answer: `{"text":"42"}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec.Question.Difficulty != "medium" {
		t.Fatalf("expected default difficulty medium, got %q", rec.Question.Difficulty)
	}
	if rec.ExpectedText() != "42" {
		t.Fatalf("expected answer text 42, got %q", rec.ExpectedText())
	}
}

func TestListFilters(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	must := func(in CreateInput) {
		if _, err := Create(ctx, in); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	must(CreateInput{UserID: 1, Type: "single_choice", Question: "alpha", Tags: []string{"SSC"}, Subject: "Math"})
	must(CreateInput{UserID: 1, Type: "input", Question: "beta", Tags: []string{"UPSC"}, Subject: "History"})
	must(CreateInput{UserID: 2, Type: "input", Question: "other user"})

	all, err := List(ctx, ListQuery{Filter: Filter{UserID: 1}})
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 records, got %d err=%v", len(all), err)
	}
	filtered, err := List(ctx, ListQuery{Filter: Filter{UserID: 1, Tags: []string{"SSC"}}})
	if err != nil || len(filtered) != 1 {
		t.Fatalf("expected 1 SSC record, got %d err=%v", len(filtered), err)
	}
	byType, err := List(ctx, ListQuery{Filter: Filter{UserID: 1, Type: "input"}})
	if err != nil || len(byType) != 1 || byType[0].Question.Question != "beta" {
		t.Fatalf("unexpected type filter result: %+v err=%v", byType, err)
	}
	byQuery, err := List(ctx, ListQuery{Filter: Filter{UserID: 1}, Query: "alph"})
	if err != nil || len(byQuery) != 1 {
		t.Fatalf("expected 1 query match, got %d err=%v", len(byQuery), err)
	}
}

func TestVerifyOwnership(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := VerifyOwnership(ctx, int(id), 1); err != nil {
		t.Fatalf("expected ownership ok: %v", err)
	}
	if err := VerifyOwnership(ctx, int(id), 2); !errors.Is(err, ErrQuestionNotOwned) {
		t.Fatalf("expected ErrQuestionNotOwned, got %v", err)
	}
	if err := VerifyOwnership(ctx, 9999, 1); !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}

func TestUpdateBuilderPartial(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Before", Tags: []string{"SSC"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	b := NewUpdateBuilder()
	if err := b.Exec(ctx, int(id)); err == nil {
		t.Fatal("expected error for empty update")
	}

	b = NewUpdateBuilder().Set("question", "After").Set("difficulty", "hard")
	if err := b.Exec(ctx, int(id)); err != nil {
		t.Fatalf("update: %v", err)
	}

	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec.Question.Question != "After" || rec.Question.Difficulty != "hard" {
		t.Fatalf("unexpected record after update: %+v", rec.Question)
	}
	if got := rec.Tags(); len(got) != 1 || got[0] != "SSC" {
		t.Fatalf("tags dropped during update: %v", got)
	}
	if !rec.Question.UpdatedAt.After(rec.Question.CreatedAt) && !rec.Question.UpdatedAt.Equal(rec.Question.CreatedAt) {
		t.Fatal("updated_at should be at or after created_at")
	}
}

func TestDelete(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	deleted, err := Delete(ctx, int(id))
	if err != nil || !deleted {
		t.Fatalf("expected delete, got %v err=%v", deleted, err)
	}
	deleted, err = Delete(ctx, int(id))
	if err != nil || deleted {
		t.Fatalf("expected second delete to miss, got %v err=%v", deleted, err)
	}
}

// ─── Scheduler resolution + FSRS path through ReviewQuestion ─────────────

func TestUserSettingResolver(t *testing.T) {
	if got := resolveSchedulerForUser(9999, SchedulerFSRS); got != SchedulerFSRS {
		t.Fatalf("empty setting must fall back to default, got %q", got)
	}
	if got := resolveSchedulerForUser(9999, SchedulerSM2); got != SchedulerSM2 {
		t.Fatalf("fall back to sm2, got %q", got)
	}
}

func TestUserSettingResolverWithKV(t *testing.T) {
	dir := t.TempDir()
	db.InitUserDB(dir)
	// In production cmd/server imports handlers, which registers the getter;
	// here we register it directly since setupDB doesn't touch UserDB.
	SetUserSettingsGetter(func(userID int, key string) (string, error) {
		var v string
		err := db.UserDB.QueryRow("SELECT value FROM user_settings WHERE user_id = ? AND key = ?", userID, key).Scan(&v)
		return v, err
	})
	t.Cleanup(func() { db.UserDB.Close(); db.UserDB = nil })
	if _, err := db.UserDB.Exec(
		"INSERT INTO user_settings (user_id, key, value) VALUES (1, 'quiz.scheduler', 'fsrs')",
	); err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if got := resolveSchedulerForUser(1, SchedulerSM2); got != SchedulerFSRS {
		t.Fatalf("fsrs user should win over sm2 default, got %q", got)
	}
	if got := resolveSchedulerForUser(2, SchedulerSM2); got != SchedulerSM2 {
		t.Fatalf("unset user falls back to default, got %q", got)
	}
	// An unknown key never hijacks the scheduler.
	if _, err := db.UserDB.Exec(
		"INSERT INTO user_settings (user_id, key, value) VALUES (2, 'quiz.scheduler', 'bogus')",
	); err != nil {
		t.Fatalf("seed bogus: %v", err)
	}
	if got := resolveSchedulerForUser(2, SchedulerSM2); got != SchedulerSM2 {
		t.Fatalf("bogus value must fall back to default, got %q", got)
	}
}

// ─── FSRS path through ReviewQuestion ────────────────────

func TestReviewQuestionFSRSPersistsColumns(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow, SchedulerFSRS); err != nil {
		t.Fatalf("fsrs review: %v", err)
	}
	var difficulty, stability float64
	var learningStep int
	var interval int64
	if err := db.QuizDB.QueryRow(
		"SELECT difficulty, stability, learning_step, interval_seconds FROM question_review_state WHERE question_id = ?", id,
	).Scan(&difficulty, &stability, &learningStep, &interval); err != nil {
		t.Fatalf("read fsrs columns: %v", err)
	}
	if stability <= 0 || difficulty <= 0 {
		t.Fatalf("fsrs columns empty: difficulty=%v stability=%v", difficulty, stability)
	}
	if learningStep != LearnStepReview {
		t.Fatalf("learning_step = %d, want %d", learningStep, LearnStepReview)
	}
	if got := time.Duration(interval) * time.Second; got != 24*time.Hour {
		t.Fatalf("first Good interval = %v, want capped 24h", got)
	}

	// Second day Good should grow the interval via FSRS stability.
	state, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow.Add(24*time.Hour), SchedulerFSRS)
	if err != nil {
		t.Fatalf("second fsrs review: %v", err)
	}
	if got := time.Duration(state.IntervalSeconds) * time.Second; got <= 24*time.Hour {
		t.Fatalf("second Good interval = %v, want > 24h", got)
	}
	if state.Repetitions != 2 {
		t.Fatalf("repetitions = %d, want 2", state.Repetitions)
	}
}

func TestReviewQuestionSM2LeavesFSRSColumnsNull(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow, SchedulerSM2); err != nil {
		t.Fatalf("sm2 review: %v", err)
	}
	var difficulty, stability sql.NullFloat64
	var learningStep sql.NullInt64
	if err := db.QuizDB.QueryRow(
		"SELECT difficulty, stability, learning_step FROM question_review_state WHERE question_id = ?", id,
	).Scan(&difficulty, &stability, &learningStep); err != nil {
		t.Fatalf("read fsrs columns: %v", err)
	}
	if difficulty.Valid || stability.Valid || learningStep.Valid {
		t.Fatalf("SM-2 should leave FSRS columns NULL, got difficulty=%v stability=%v step=%v",
			difficulty.Valid, stability.Valid, learningStep.Valid)
	}
}

func TestMigrateSM2ToFSRS(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	id, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Build a small SM-2 history: two Good reviews a day apart.
	if _, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow.Add(-48*time.Hour), SchedulerSM2); err != nil {
		t.Fatalf("sm2 first: %v", err)
	}
	sm2, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow.Add(-24*time.Hour), SchedulerSM2)
	if err != nil {
		t.Fatalf("sm2 second: %v", err)
	}
	if got := time.Duration(sm2.IntervalSeconds) * time.Second; got != 3*24*time.Hour {
		t.Fatalf("sm2 seed interval = %v, want 3d", got)
	}
	// The original 3-day due date must remain; conversion only seeds new columns.
	wantDue := sm2.DueAt

	flipped, err := ReviewQuestion(ctx, int(id), 1, RatingGood, reviewNow, SchedulerFSRS)
	if err != nil {
		t.Fatalf("fsrs review after flip: %v", err)
	}
	var difficulty, stability float64
	var learningStep int
	if err := db.QuizDB.QueryRow(
		"SELECT difficulty, stability, learning_step FROM question_review_state WHERE question_id = ?", id,
	).Scan(&difficulty, &stability, &learningStep); err != nil {
		t.Fatalf("read converted columns: %v", err)
	}
	if difficulty <= 0 || stability <= 0 {
		t.Fatalf("conversion left columns empty: difficulty=%v stability=%v", difficulty, stability)
	}
	// The converted 3d/EF stability seed grew by exactly one FSRS Good at
	// 1-day retrievability; assert it stayed in a sane band rather than
	// exact-equals (FSRS growth depends on difficulty and decay curves).
	if stability < 1.2 || stability > 6 {
		t.Fatalf("stability %v out of the expected post-flip band", stability)
	}
	if learningStep != LearnStepReview {
		t.Fatalf("learning_step = %d, want %d", learningStep, LearnStepReview)
	}
	if flipped.DueAt.Before(reviewNow.Add(24 * time.Hour)) {
		t.Fatalf("fsrs interval after conversion should reach past tomorrow, due_at = %v (was %v)", flipped.DueAt, wantDue)
	}
	if flipped.Repetitions != 3 {
		t.Fatalf("repetitions = %d, want 3", flipped.Repetitions)
	}
}

// ─── Papers ─────────────────────────────────────────────

func TestGenerateAndPersistPaper(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	for _, q := range []struct{ tag, subject string }{
		{"SSC", "Math"}, {"SSC", "Math"}, {"SSC", "English"},
	} {
		if _, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: q.tag + " " + q.subject, Tags: []string{q.tag}, Subject: q.subject}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	sections := []models.QuestionPaperSection{
		{Tags: []string{"SSC"}, Subject: "Math", Count: 2},
		{Tags: []string{"SSC"}, Subject: "English", Count: 1},
	}
	records, err := GenerateSectionedPaper(ctx, 1, sections, false, DefaultMaxPaperSize)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(records))
	}

	paperID, err := PersistPaper(ctx, 1, "Sample Paper", sections, records)
	if err != nil {
		t.Fatalf("persist: %v", err)
	}

	paper, err := GetPaperByID(ctx, int(paperID))
	if err != nil {
		t.Fatalf("get paper: %v", err)
	}
	if paper.QuestionCount != 3 || len(paper.Questions) != 3 {
		t.Fatalf("unexpected paper: %+v", paper)
	}

	// Overlapping sections must not duplicate a question when disallowed.
	overlap := []models.QuestionPaperSection{
		{Subject: "Math", Count: 2},
		{Subject: "Math", Count: 2},
	}
	records, err = GenerateSectionedPaper(ctx, 1, overlap, false, DefaultMaxPaperSize)
	if err != nil {
		t.Fatalf("generate overlap: %v", err)
	}
	seen := map[int]bool{}
	for _, rec := range records {
		if seen[rec.Question.ID] {
			t.Fatalf("duplicate question %d in paper", rec.Question.ID)
		}
		seen[rec.Question.ID] = true
	}

	// Bad section count is rejected.
	if _, err := GenerateSectionedPaper(ctx, 1, []models.QuestionPaperSection{{Count: 0}}, false, DefaultMaxPaperSize); err == nil {
		t.Fatal("expected error for zero-count section")
	}

	papers, err := ListPapers(ctx, 1, 10, 0)
	if err != nil || len(papers) != 1 {
		t.Fatalf("expected 1 paper, got %d err=%v", len(papers), err)
	}

	deleted, err := DeletePaper(ctx, int(paperID))
	if err != nil || !deleted {
		t.Fatalf("delete paper: %v err=%v", deleted, err)
	}
	if _, err := GetPaperByID(ctx, int(paperID)); !errors.Is(err, ErrPaperNotFound) {
		t.Fatalf("expected ErrPaperNotFound, got %v", err)
	}
}

func TestStatsAndVocabulary(t *testing.T) {
	setupDB(t)
	ctx := context.Background()
	if _, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q1", Tags: []string{"SSC"}, Subject: "Math"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "Q2", Tags: []string{"SSC"}, Subject: "English", Difficulty: "easy"}); err != nil {
		t.Fatal(err)
	}

	stats, err := Stats(ctx, 1)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Total != 2 {
		t.Fatalf("expected total 2, got %v", stats.Total)
	}
	if stats.ByTags["SSC"] != 2 {
		t.Fatalf("unexpected by_tags: %v", stats.ByTags)
	}

	vocab, err := TagVocabulary(ctx, 1)
	if err != nil {
		t.Fatalf("vocab: %v", err)
	}
	if len(vocab.Subjects) != 2 || len(vocab.Tags) != 1 {
		t.Fatalf("unexpected vocab: %v", vocab)
	}
}

// ─── Regression: empty helpers return [] not nil ─────────────

func TestEmptyHelpersReturnEmptySlice(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	// A chronology question created without resupplied items would store
	// options_json = "[]". Both Options() and ChronologyItems() must return
	// an empty slice (not nil) so the JSON shape stays stable.
	id, err := Create(ctx, CreateInput{
		UserID: 1, Type: "chronology", Difficulty: "easy",
		Question: "Order these",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := rec.Options(); got == nil {
		t.Fatal("Options() returned nil for empty JSON")
	}
	if got := rec.ChronologyItems(); got == nil {
		t.Fatal("ChronologyItems() returned nil for empty JSON")
	}
}

// ─── Regression: paper generation dedupes across sections ────

func TestGenerateSectionedPaperDedupes(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	// Create 5 easy Math questions and 5 easy GK questions.
	for i := 0; i < 5; i++ {
		if _, err := Create(ctx, CreateInput{
			UserID: 1, Type: "input", Difficulty: "easy",
			Question: "Math?", Subject: "Math",
		}); err != nil {
			t.Fatalf("create math: %v", err)
		}
		if _, err := Create(ctx, CreateInput{
			UserID: 1, Type: "input", Difficulty: "easy",
			Question: "GK?", Subject: "GK",
		}); err != nil {
			t.Fatalf("create gk: %v", err)
		}
	}

	sections := []models.QuestionPaperSection{
		{Subject: "Math", Difficulty: "easy", Count: 4},
		{Subject: "GK", Difficulty: "easy", Count: 4},
	}
	records, err := GenerateSectionedPaper(ctx, 1, sections, false, DefaultMaxPaperSize)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(records) != 8 {
		t.Fatalf("expected 8 unique questions, got %d", len(records))
	}
	seen := map[int]bool{}
	for _, r := range records {
		if seen[r.Question.ID] {
			t.Fatalf("duplicate id %d in paper", r.Question.ID)
		}
		seen[r.Question.ID] = true
	}
}

// ─── Regression: tags array round-trip ─────────────────────

func TestCreateQuestionWithMultipleTags(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	id, err := Create(ctx, CreateInput{
		UserID: 1, Type: "input", Question: "Polity",
		Tags: []string{"SSC", "RRB", "Banking"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec, err := GetByID(ctx, int(id))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got := rec.Tags()
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %v", got)
	}
	want := map[string]bool{"SSC": true, "RRB": true, "Banking": true}
	for _, tag := range got {
		if !want[tag] {
			t.Fatalf("unexpected tag %q", tag)
		}
	}

	// JSON column is the canonical storage; ensure it round-trips.
	if rec.Question.Tags != `["SSC","RRB","Banking"]` {
		t.Fatalf("raw tags column = %q", rec.Question.Tags)
	}
}

// ─── Regression: tag filter matches any value ─────────────────

func TestListQuestionsByTagMatchesAny(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	must := func(in CreateInput) {
		if _, err := Create(ctx, in); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	must(CreateInput{UserID: 1, Type: "input", Question: "polity", Tags: []string{"UPSC"}})
	must(CreateInput{UserID: 1, Type: "input", Question: "arith", Tags: []string{"SSC", "RRB"}})
	must(CreateInput{UserID: 1, Type: "input", Question: "geog", Tags: []string{"Banking"}})
	must(CreateInput{UserID: 2, Type: "input", Question: "other user", Tags: []string{"SSC"}})

	// Single-tag filter.
	got, err := List(ctx, ListQuery{Filter: Filter{UserID: 1, Tags: []string{"SSC"}}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Question.Question != "arith" {
		t.Fatalf("expected only arith, got %+v", got)
	}

	// Multi-tag filter matches any — returns arith + geog.
	got, err = List(ctx, ListQuery{Filter: Filter{UserID: 1, Tags: []string{"SSC", "Banking"}}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}

	// User scoping still wins — UPCS-only user 1 row is returned, user 2's
	// SSC row is not.
	got, err = List(ctx, ListQuery{Filter: Filter{UserID: 1, Tags: []string{"UPSC"}}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Question.Question != "polity" {
		t.Fatalf("expected polity, got %+v", got)
	}
}

// ─── Regression: paper section with tags filter ─────────────────

func TestGeneratePaperWithTagFilter(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	for _, q := range []struct {
		tags    []string
		subject string
	}{
		{[]string{"SSC"}, "Math"},
		{[]string{"SSC", "RRB"}, "Math"},
		{[]string{"UPSC"}, "History"},
	} {
		if _, err := Create(ctx, CreateInput{
			UserID: 1, Type: "input",
			Question: q.subject + "-" + q.tags[0],
			Tags:     q.tags, Subject: q.subject,
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	sections := []models.QuestionPaperSection{
		{Tags: []string{"SSC"}, Count: 10},
	}
	records, err := GenerateSectionedPaper(ctx, 1, sections, true, DefaultMaxPaperSize)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 SSC-tagged rows, got %d", len(records))
	}
	for _, r := range records {
		tags := r.Tags()
		found := false
		for _, t := range tags {
			if t == "SSC" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("paper included non-SSC row: %v", tags)
		}
	}
}

// ─── Regression: tags dashboard stats ─────────────────────

func TestStatsByTagsCountDistinctQuestions(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	// Two questions both carry "SSC" — by_tags should count each question
	// once, not double-count because of the JSON-array aggregation.
	must := func(in CreateInput) {
		if _, err := Create(ctx, in); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	must(CreateInput{UserID: 1, Type: "input", Question: "a", Tags: []string{"SSC"}})
	must(CreateInput{UserID: 1, Type: "input", Question: "b", Tags: []string{"SSC", "RRB"}})
	must(CreateInput{UserID: 1, Type: "input", Question: "c", Tags: []string{"UPSC"}})

	stats, err := Stats(ctx, 1)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	byTags := stats.ByTags
	if byTags["SSC"] != 2 {
		t.Fatalf("expected SSC count 2, got %d", byTags["SSC"])
	}
	if byTags["RRB"] != 1 {
		t.Fatalf("expected RRB count 1, got %d", byTags["RRB"])
	}
	if byTags["UPSC"] != 1 {
		t.Fatalf("expected UPSC count 1, got %d", byTags["UPSC"])
	}
}

func TestSampleRandomRespectsFilterAndCount(t *testing.T) {
	setupDB(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if _, err := Create(ctx, CreateInput{
			UserID: 1, Type: "input", Difficulty: "easy",
			Question: "math", Subject: "Math",
		}); err != nil {
			t.Fatalf("create math: %v", err)
		}
	}
	if _, err := Create(ctx, CreateInput{UserID: 1, Type: "input", Question: "gk", Subject: "GK"}); err != nil {
		t.Fatalf("create gk: %v", err)
	}
	if _, err := Create(ctx, CreateInput{UserID: 2, Type: "input", Question: "other", Subject: "Math"}); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	got, err := SampleRandom(ctx, Filter{UserID: 1, Subject: "Math"}, 4)
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 records, got %d", len(got))
	}
	seen := map[int]bool{}
	for _, rec := range got {
		if rec.Question.Subject != "Math" || rec.Question.UserID != 1 {
			t.Fatalf("filter leaked: %+v", rec.Question)
		}
		if seen[rec.Question.ID] {
			t.Fatalf("duplicate question id %d in sample", rec.Question.ID)
		}
		seen[rec.Question.ID] = true
	}

	// Asking for more than exist returns everything that matches, not an error.
	all, err := SampleRandom(ctx, Filter{UserID: 1, Subject: "GK"}, 50)
	if err != nil {
		t.Fatalf("sample gk: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 GK record, got %d", len(all))
	}
}

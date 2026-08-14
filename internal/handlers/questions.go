package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/quiz"
	quizconfig "browser-server/internal/quiz/config"
)

func quizEnabled(w http.ResponseWriter) bool {
	if !quizconfig.Get().Enabled {
		helpers.WriteError(w, http.StatusNotFound, "quiz feature disabled")
		return false
	}
	return true
}

func quizImageDir() string {
	return quizconfig.Get().ResolveImageDir(db.GetDataPath())
}

// writeQuizValidationError renders a domain validation failure. Errors that
// name a request field become a per-field validation response; anything else
// falls back to a plain 400.
func writeQuizValidationError(w http.ResponseWriter, err error) {
	var fieldErr *quiz.FieldError
	if errors.As(err, &fieldErr) {
		helpers.WriteValidationError(w, map[string]string{fieldErr.Field: fieldErr.Error()})
		return
	}
	helpers.WriteError(w, http.StatusBadRequest, err.Error())
}

// loadQuestionForRequest resolves the {id} path param to a question,
// writing the appropriate error response and returning ok=false on failure.
func loadQuestionForRequest(w http.ResponseWriter, r *http.Request) (quiz.Record, int, bool) {
	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid question id")
		return quiz.Record{}, 0, false
	}
	rec, err := quiz.GetByID(r.Context(), id)
	if errors.Is(err, quiz.ErrNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "Question not found")
		return quiz.Record{}, 0, false
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return quiz.Record{}, 0, false
	}
	return rec, id, true
}

// ─── Questions ──────────────────────────────────────────

func GetQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	q := r.URL.Query()
	records, err := quiz.List(r.Context(), quiz.ListQuery{
		Filter: quiz.Filter{
			UserID:     userID,
			Type:       q.Get("type"),
			Difficulty: q.Get("difficulty"),
			Tags:       q["tag"],
			Subject:    q.Get("subject"),
			Topic:      q.Get("topic"),
			SubTopic:   q.Get("sub_topic"),
		},
		Query:  strings.TrimSpace(q.Get("q")),
		Limit:  helpers.GetLimitFromQuery(r, 200),
		Offset: helpers.GetOffsetFromQuery(r),
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(quiz.Responses(records))
}

func CreateQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	var input struct {
		UserID          int                     `json:"user_id"`
		Type            string                  `json:"type"`
		Difficulty      string                  `json:"difficulty"`
		Question        string                  `json:"question"`
		Explanation     string                  `json:"explanation"`
		Options         []models.QuestionOption `json:"options"`
		ChronologyItems []models.ChronologyItem `json:"chronology_items"`
		ExpectedText    string                  `json:"expected_text"`
		Tags            []string                `json:"tags"`
		Subject         string                  `json:"subject"`
		Topic           string                  `json:"topic"`
		SubTopic        string                  `json:"sub_topic"`
		Source          string                  `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	createInput, err := quizconfig.Get().Rules().BuildCreate(quiz.CreateFields{
		UserID:      input.UserID,
		Type:        input.Type,
		Difficulty:  input.Difficulty,
		Question:    input.Question,
		Explanation: input.Explanation,
		Payload: quiz.AnswerPayload{
			Options:         input.Options,
			ChronologyItems: input.ChronologyItems,
			ExpectedText:    input.ExpectedText,
		},
		Tags:     input.Tags,
		Subject:  input.Subject,
		Topic:    input.Topic,
		SubTopic: input.SubTopic,
		Source:   input.Source,
	})
	if err != nil {
		writeQuizValidationError(w, err)
		return
	}

	id, err := quiz.Create(r.Context(), createInput)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rec, err := quiz.GetByID(r.Context(), int(id))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(quiz.Response(rec))
}

func GetQuestionByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	rec, _, ok := loadQuestionForRequest(w, r)
	if !ok {
		return
	}

	json.NewEncoder(w).Encode(quiz.Response(rec))
}

func UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	rec, id, ok := loadQuestionForRequest(w, r)
	if !ok {
		return
	}

	var input quiz.EditFields
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Answer payloads are validated against the effective type (a type change
	// must re-supply matching answers).
	builder, err := quizconfig.Get().Rules().BuildUpdate(input, rec.Question.Type)
	if err != nil {
		writeQuizValidationError(w, err)
		return
	}
	if builder.Empty() {
		helpers.WriteError(w, http.StatusBadRequest, "No updatable fields provided")
		return
	}
	if err := builder.Exec(r.Context(), id); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rec, err = quiz.GetByID(r.Context(), id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(quiz.Response(rec))
}

func DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	rec, id, ok := loadQuestionForRequest(w, r)
	if !ok {
		return
	}

	deleted, err := quiz.Delete(r.Context(), id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if !deleted {
		helpers.WriteError(w, http.StatusNotFound, "Question not found")
		return
	}
	// Best-effort cleanup of the attached image file and its bookkeeping rows.
	if rec.Question.ImageFilename != "" {
		os.Remove(filepath.Join(quizImageDir(), filepath.Base(rec.Question.ImageFilename)))
	}
	if err := quiz.DeleteImageRecords(r.Context(), id); err != nil {
		log.Printf("quiz: failed to clean up image rows for question %d: %v", id, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Spaced-repetition cards ────────────────────────────

func GetQuestionCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}
	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			helpers.WriteError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	practice := r.URL.Query().Get("practice") == "true"
	mode := r.URL.Query().Get("mode")
	if err := quiz.ValidateCardMode(mode); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	queue, err := quiz.ListCards(r.Context(), userID, r.URL.Query()["tag"], limit, time.Now(), practice, mode, quiz.SchedulerForUser(userID))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(queue)
}

func ReviewQuestionCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}
	questionID := helpers.GetIDFromPath(r)
	if questionID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid question id")
		return
	}
	var input struct {
		UserID int    `json:"user_id"`
		Rating string `json:"rating"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if input.UserID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if err := quiz.ValidateRating(input.Rating); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := quiz.ReviewQuestion(r.Context(), questionID, input.UserID, input.Rating, time.Now(), quiz.SchedulerForUser(input.UserID))
	if errors.Is(err, quiz.ErrQuestionNotFound) || errors.Is(err, quiz.ErrQuestionNotOwned) {
		helpers.WriteError(w, http.StatusNotFound, "Question not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(state)
}

func SkipQuestionCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}
	questionID := helpers.GetIDFromPath(r)
	if questionID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid question id")
		return
	}
	var input struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if input.UserID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	state, err := quiz.RecordSkip(r.Context(), questionID, input.UserID, time.Now())
	if errors.Is(err, quiz.ErrQuestionNotFound) || errors.Is(err, quiz.ErrQuestionNotOwned) {
		helpers.WriteError(w, http.StatusNotFound, "Question not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(state)
}

// ─── Images ─────────────────────────────────────────────

func UploadQuestionImage(w http.ResponseWriter, r *http.Request) {
	if !quizEnabled(w) {
		return
	}

	rec, id, ok := loadQuestionForRequest(w, r)
	if !ok {
		return
	}

	// Remember any prior image so a successful replacement can clean up the
	// orphaned file. Re-uploading on the same question replaces the column
	// without otherwise noticing the leftover bytes on disk.
	priorFilename := rec.Question.ImageFilename

	cfg := quizconfig.Get()
	if err := r.ParseMultipartForm(cfg.Limits.MaxImageBytes + (1 << 20)); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Missing file")
		return
	}
	defer file.Close()

	if header.Size > cfg.Limits.MaxImageBytes {
		helpers.WriteError(w, http.StatusRequestEntityTooLarge, "Image exceeds the maximum size")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
	default:
		ext = ".png"
	}

	var buf [16]byte
	rand.Read(buf[:])
	filename := fmt.Sprintf("q%d-%x%s", id, buf, ext)

	dir := quizImageDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to prepare image directory")
		return
	}
	outPath := filepath.Join(dir, filename)
	out, err := os.Create(outPath)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	_, copyErr := io.Copy(out, file)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(outPath)
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	if err := quiz.SetImageFilename(r.Context(), id, filename); err != nil {
		os.Remove(outPath)
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Best-effort cleanup of the prior image now that the new one is recorded.
	// We compare the basenames so a path-traversal-style prior value cannot
	// steer the removal somewhere unintended.
	if priorFilename != "" && priorFilename != filename {
		prior := filepath.Base(priorFilename)
		if prior != "" && prior != filename {
			os.Remove(filepath.Join(dir, prior))
			if err := quiz.DeleteImageRecord(r.Context(), id, prior); err != nil {
				log.Printf("quiz: failed to remove prior image row for question %d: %v", id, err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":        id,
		"filename":  filename,
		"image_url": fmt.Sprintf("/api/quiz/questions/%d/image", id),
	})
}

func GetQuestionImage(w http.ResponseWriter, r *http.Request) {
	if !quizEnabled(w) {
		return
	}

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid question id")
		return
	}

	rec, err := quiz.GetByID(r.Context(), id)
	if err != nil || rec.Question.ImageFilename == "" {
		helpers.WriteError(w, http.StatusNotFound, "Image not found")
		return
	}

	// Guard against path traversal: only serve files that live directly in the
	// image directory.
	filename := filepath.Base(rec.Question.ImageFilename)
	http.ServeFile(w, r, filepath.Join(quizImageDir(), filename))
}

// ─── Papers ─────────────────────────────────────────────

func GeneratePaper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	var input struct {
		UserID   int                           `json:"user_id"`
		Title    string                        `json:"title"`
		Sections []models.QuestionPaperSection `json:"sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("title", input.Title)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}
	if len(input.Sections) == 0 {
		helpers.WriteValidationError(w, map[string]string{"sections": "must contain at least one section"})
		return
	}

	cfg := quizconfig.Get()
	rules := cfg.Rules()
	total := 0
	for i, s := range input.Sections {
		if s.Count < 1 {
			helpers.WriteValidationError(w, map[string]string{"sections": fmt.Sprintf("sections[%d].count must be at least 1", i)})
			return
		}
		if s.Type != "" {
			if err := rules.ValidateType(s.Type); err != nil {
				helpers.WriteValidationError(w, map[string]string{"sections": fmt.Sprintf("sections[%d]: %v", i, err)})
				return
			}
		}
		if err := rules.ValidateDifficulty(s.Difficulty); err != nil {
			helpers.WriteValidationError(w, map[string]string{"sections": fmt.Sprintf("sections[%d]: %v", i, err)})
			return
		}
		total += s.Count
	}
	if total > rules.Limits.MaxPaperSize {
		helpers.WriteValidationError(w, map[string]string{"sections": fmt.Sprintf("paper would exceed %d questions", rules.Limits.MaxPaperSize)})
		return
	}

	count, err := quiz.CountPapers(r.Context(), input.UserID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if count >= cfg.Limits.MaxPapersPerUser {
		helpers.WriteError(w, http.StatusConflict, "Paper limit reached for this user")
		return
	}

	records, err := quiz.GenerateSectionedPaper(r.Context(), input.UserID, input.Sections,
		cfg.PaperGeneration.AllowDuplicateQuestionsWithinPaper, rules.Limits.MaxPaperSize)
	if err != nil {
		// Only a section-validation failure is the caller's fault; a query
		// failure must not be reported as a bad request.
		var fieldErr *quiz.FieldError
		if errors.As(err, &fieldErr) {
			helpers.WriteValidationError(w, map[string]string{fieldErr.Field: fieldErr.Error()})
		} else {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		}
		return
	}
	if len(records) == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "No questions matched the requested sections")
		return
	}

	paperID, err := quiz.PersistPaper(r.Context(), input.UserID, strings.TrimSpace(input.Title), input.Sections, records)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	paper, err := quiz.GetPaperByID(r.Context(), int(paperID))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(paper)
}

func GetPapers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	papers, err := quiz.ListPapers(r.Context(), userID, helpers.GetLimitFromQuery(r, 100), helpers.GetOffsetFromQuery(r))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(papers)
}

func GetPaperByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid paper id")
		return
	}

	paper, err := quiz.GetPaperByID(r.Context(), id)
	if errors.Is(err, quiz.ErrPaperNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "Paper not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(paper)
}

func DeletePaper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid paper id")
		return
	}

	deleted, err := quiz.DeletePaper(r.Context(), id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if !deleted {
		helpers.WriteError(w, http.StatusNotFound, "Paper not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Vocabulary & stats ─────────────────────────────────

func GetTagVocabulary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	vocab, err := quiz.TagVocabulary(r.Context(), userID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(vocab)
}

func GetQuizStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	stats, err := quiz.Stats(r.Context(), userID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(stats)
}

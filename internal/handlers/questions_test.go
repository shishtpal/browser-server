package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"browser-server/internal/db"
	quizconfig "browser-server/internal/quiz/config"

	"github.com/gorilla/mux"
)

// enableQuizForTest installs a minimal enabled quiz config and inits the
// quiz DB on a temp dir. The caller is responsible for closing the DB.
func enableQuizForTest(t *testing.T) {
	t.Helper()
	db.InitQuizDB(t.TempDir())
	quizconfig.SetForTest(&quizconfig.Config{
		Enabled:              true,
		DBPath:               ".data/quiz.db",
		ImageDir:             ".data/quiz-images",
		Limits:               quizconfig.DefaultLimitsForTest(),
		AllowedQuestionTypes: []string{"single_choice", "multiple_choice", "input", "chronology"},
		AllowedDifficulties:  []string{"easy", "medium", "hard"},
		TagCategories:        []string{"subject", "topic", "sub_topic"},
		PaperGeneration:      quizconfig.DefaultPaperGenerationForTest(),
	})
	t.Cleanup(func() {
		db.CloseQuizDB()
		quizconfig.ResetForTest()
	})
}

// ─── Regression: changing type without resupplying payloads resets the
// stored JSON to "[]" so the row stays consistent with the new type.

func TestUpdateQuestionResetsStalePayloadsOnTypeChange(t *testing.T) {
	enableQuizForTest(t)

	// Seed a single_choice question via the create handler so we exercise
	// the same path the operator uses.
	body := map[string]any{
		"user_id":     1,
		"type":        "single_choice",
		"difficulty":  "easy",
		"question":    "Pick A",
		"explanation": "because",
		"options": []map[string]any{
			{"index": 0, "text": "A", "correct": true},
			{"index": 1, "text": "B"},
		},
	}
	encoded, _ := json.Marshal(body)
	createReq := httptest.NewRequest(http.MethodPost, "/api/quiz/questions", bytes.NewReader(encoded))
	createResp := httptest.NewRecorder()
	CreateQuestion(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createResp.Code, createResp.Body.String())
	}

	// Now flip the type to "input" without supplying expected_text. The
	// handler must reset options_json/answer_json to "[]" so the row no
	// longer carries choice payloads that don't match the new type.
	updateBody := map[string]any{"type": "input"}
	encodedUpdate, _ := json.Marshal(updateBody)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/quiz/questions/1", bytes.NewReader(encodedUpdate))
	updateReq = mux.SetURLVars(updateReq, map[string]string{"id": "1"})
	updateResp := httptest.NewRecorder()
	UpdateQuestion(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateResp.Code, updateResp.Body.String())
	}

	// Re-fetch via the GET handler and confirm the raw columns were reset.
	getReq := httptest.NewRequest(http.MethodGet, "/api/quiz/questions/1", nil)
	getReq = mux.SetURLVars(getReq, map[string]string{"id": "1"})
	getResp := httptest.NewRecorder()
	GetQuestionByID(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getResp.Code, getResp.Body.String())
	}

	var got struct {
		Type    string `json:"type"`
		Options any    `json:"options"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Type != "input" {
		t.Fatalf("type = %q, want input", got.Type)
	}
	// `options` is omitted on a non-choice response (omitempty), so we verify
	// the underlying columns instead via the SQLite store.
	var optionsJSON, answerJSON string
	if err := db.QuizDB.QueryRow(
		"SELECT options_json, answer_json FROM questions WHERE id = 1").Scan(&optionsJSON, &answerJSON); err != nil {
		t.Fatalf("query: %v", err)
	}
	if optionsJSON != "[]" {
		t.Fatalf("options_json = %q, want \"[]\"", optionsJSON)
	}
	if answerJSON != "[]" {
		t.Fatalf("answer_json = %q, want \"[]\"", answerJSON)
	}
}

// ─── Regression: tags array round-trips through the REST API ───────

func TestCreateQuestionWireTagsRoundTrip(t *testing.T) {
	enableQuizForTest(t)

	body := map[string]any{
		"user_id":  1,
		"type":     "single_choice",
		"question": "Polity",
		"tags":     []string{"SSC", "RRB"},
		"subject":  "GK",
		"options": []map[string]any{
			{"index": 0, "text": "A", "correct": true},
			{"index": 1, "text": "B"},
		},
	}
	encoded, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/quiz/questions", bytes.NewReader(encoded))
	rec := httptest.NewRecorder()
	CreateQuestion(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID     int      `json:"id"`
		Tags   []string `json:"tags"`
		Source string   `json:"source"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tags) != 2 || resp.Tags[0] != "SSC" || resp.Tags[1] != "RRB" {
		t.Fatalf("tags round-trip mismatch: %v", resp.Tags)
	}

	// Re-fetch via the GET handler.
	getReq := httptest.NewRequest(http.MethodGet, "/api/quiz/questions/1", nil)
	getReq = mux.SetURLVars(getReq, map[string]string{"id": "1"})
	getResp := httptest.NewRecorder()
	GetQuestionByID(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getResp.Code, getResp.Body.String())
	}

	var got struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("get tags = %v", got.Tags)
	}

	// GET /api/quiz/tags returns the flattened vocabulary.
	tagsReq := httptest.NewRequest(http.MethodGet, "/api/quiz/tags?user_id=1", nil)
	tagsResp := httptest.NewRecorder()
	GetTagVocabulary(tagsResp, tagsReq)
	if tagsResp.Code != http.StatusOK {
		t.Fatalf("tags status = %d, body = %s", tagsResp.Code, tagsResp.Body.String())
	}
	var vocab struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(tagsResp.Body).Decode(&vocab); err != nil {
		t.Fatalf("decode vocab: %v", err)
	}
	if len(vocab.Tags) != 2 {
		t.Fatalf("vocab tags = %v", vocab.Tags)
	}

	// GET /api/quiz/stats returns by_tags (not by_exam).
	statsReq := httptest.NewRequest(http.MethodGet, "/api/quiz/stats?user_id=1", nil)
	statsResp := httptest.NewRecorder()
	GetQuizStats(statsResp, statsReq)
	if statsResp.Code != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", statsResp.Code, statsResp.Body.String())
	}
	var stats struct {
		ByTags map[string]int `json:"by_tags"`
	}
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.ByTags["SSC"] != 1 {
		t.Fatalf("stats by_tags = %v", stats.ByTags)
	}
}

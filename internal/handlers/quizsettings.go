package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/quiz"
)

func init() {
	// Wire the quiz package's KV lookup so SchedulerForUser sees the same
	// user_settings rows this handler writes.
	quiz.SetUserSettingsGetter(func(userID int, key string) (string, error) {
		if db.UserDB == nil {
			return "", sql.ErrNoRows
		}
		var v string
		err := db.UserDB.QueryRow("SELECT value FROM user_settings WHERE user_id = ? AND key = ?", userID, key).Scan(&v)
		return v, err
	})
}

// GetQuizSettings returns a user's spaced-repetition scheduler choice. The
// frontend reads it once per session start; the value never changes how a
// question is answered, only when it comes back.
func GetQuizSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}
	id := helpers.GetIDFromPath(r)
	if id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid user id")
		return
	}
	if !quizSettingsUserExists(w, id) {
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"user_id": id, "scheduler": quiz.SchedulerForUser(id)})
}

// UpdateQuizSettings persists a user's scheduler choice. Unknown values are
// rejected so clients can't quietly store typos like "fsr".
func UpdateQuizSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !quizEnabled(w) {
		return
	}
	id := helpers.GetIDFromPath(r)
	if id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid user id")
		return
	}
	if !quizSettingsUserExists(w, id) {
		return
	}
	var input struct {
		Scheduler string `json:"scheduler"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if input.Scheduler != quiz.SchedulerSM2 && input.Scheduler != quiz.SchedulerFSRS {
		helpers.WriteError(w, http.StatusBadRequest, "scheduler must be \"sm2\" or \"fsrs\"")
		return
	}
	if db.UserDB == nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	_, err := db.UserDB.Exec(`INSERT INTO user_settings (user_id, key, value) VALUES (?, ?, ?)
		ON CONFLICT(user_id, key) DO UPDATE SET value=excluded.value`,
		id, quiz.UserSettingKeyScheduler, input.Scheduler)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"user_id": id, "scheduler": quiz.SchedulerForUser(id)})
}

func quizSettingsUserExists(w http.ResponseWriter, userID int) bool {
	if db.UserDB == nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return false
	}
	var found int
	err := db.UserDB.QueryRow("SELECT 1 FROM users WHERE id = ?", userID).Scan(&found)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "User not found")
		return false
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return false
	}
	return true
}

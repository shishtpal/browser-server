package handlers

import (
	"encoding/json"
	"net/http"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
)

func ReorderTodos(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Items []struct {
			ID       int `json:"id"`
			Position int `json:"position"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(input.Items) == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "items must not be empty")
		return
	}

	tx, err := db.TodoDB.Begin()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	for _, item := range input.Items {
		_, err := tx.Exec("UPDATE todos SET position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", item.Position, item.ID)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

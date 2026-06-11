package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	urlFilter := r.URL.Query().Get("url")

	query := "SELECT id, user_id, url, title, visited_at, duration FROM history WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if urlFilter != "" {
		query += " AND url LIKE ?"
		args = append(args, "%"+urlFilter+"%")
	}

	query += " ORDER BY visited_at DESC"

	rows, err := db.HistoryDB.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []models.History
	for rows.Next() {
		var entry models.History
		err := rows.Scan(&entry.ID, &entry.UserID, &entry.URL, &entry.Title, &entry.VisitedAt, &entry.Duration)
		if err != nil {
			continue
		}
		history = append(history, entry)
	}

	json.NewEncoder(w).Encode(history)
}

func CreateHistory(w http.ResponseWriter, r *http.Request) {
	var entry models.History
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if entry.VisitedAt.IsZero() {
		entry.VisitedAt = time.Now()
	}

	result, err := db.HistoryDB.Exec("INSERT INTO history (user_id, url, title, visited_at, duration) VALUES (?, ?, ?, ?, ?)",
		entry.UserID, entry.URL, entry.Title, entry.VisitedAt, entry.Duration)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	entry.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func GetHistoryByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var entry models.History
	err := db.HistoryDB.QueryRow("SELECT id, user_id, url, title, visited_at, duration FROM history WHERE id = ?", id).
		Scan(&entry.ID, &entry.UserID, &entry.URL, &entry.Title, &entry.VisitedAt, &entry.Duration)

	if err == sql.ErrNoRows {
		http.Error(w, "History entry not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func DeleteHistory(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	result, err := db.HistoryDB.Exec("DELETE FROM history WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "History entry not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

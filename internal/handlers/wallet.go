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

func GetWallet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	websiteFilter := r.URL.Query().Get("website")

	query := "SELECT id, user_id, username, password, website, description, created_at, updated_at FROM wallet WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if websiteFilter != "" {
		query += " AND website LIKE ?"
		args = append(args, "%"+websiteFilter+"%")
	}

	rows, err := db.WalletDB.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	wallet := []models.WalletEntry{}
	for rows.Next() {
		var entry models.WalletEntry
		err := rows.Scan(&entry.ID, &entry.UserID, &entry.Username, &entry.Password, &entry.Website, &entry.Description, &entry.CreatedAt, &entry.UpdatedAt)
		if err != nil {
			continue
		}
		// Passwords stay hidden by default; they must be requested via the
		// reveal endpoint for a specific website + username.
		entry.Password = ""
		wallet = append(wallet, entry)
	}

	json.NewEncoder(w).Encode(wallet)
}

// RevealWalletPassword returns the password for a single credential identified
// by the requested user, website, and username. All three are required so a
// password can only be requested for a specific domain + username pair.
func RevealWalletPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	website := r.URL.Query().Get("website")
	username := r.URL.Query().Get("username")

	if userID == 0 || website == "" || username == "" {
		http.Error(w, "user_id, website, and username are required", http.StatusBadRequest)
		return
	}

	var password string
	err := db.WalletDB.QueryRow(
		"SELECT password FROM wallet WHERE user_id = ? AND website = ? AND username = ? LIMIT 1",
		userID, website, username,
	).Scan(&password)

	if err == sql.ErrNoRows {
		http.Error(w, "Wallet entry not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"password": password})
}

func CreateWalletEntry(w http.ResponseWriter, r *http.Request) {
	var entry models.WalletEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.WalletDB.Exec("INSERT INTO wallet (user_id, username, password, website, description) VALUES (?, ?, ?, ?, ?)",
		entry.UserID, entry.Username, entry.Password, entry.Website, entry.Description)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	entry.ID = int(id)
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func GetWalletByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var entry models.WalletEntry
	err := db.WalletDB.QueryRow("SELECT id, user_id, username, password, website, description, created_at, updated_at FROM wallet WHERE id = ?", id).
		Scan(&entry.ID, &entry.UserID, &entry.Username, &entry.Password, &entry.Website, &entry.Description, &entry.CreatedAt, &entry.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Wallet entry not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Password stays hidden by default; use the reveal endpoint to request it.
	entry.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func UpdateWalletEntry(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var entry models.WalletEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.WalletDB.Exec("UPDATE wallet SET user_id = ?, username = ?, password = ?, website = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		entry.UserID, entry.Username, entry.Password, entry.Website, entry.Description, id)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	entry.ID = id
	entry.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func DeleteWalletEntry(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	result, err := db.WalletDB.Exec("DELETE FROM wallet WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Wallet entry not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

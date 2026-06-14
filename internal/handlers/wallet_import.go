package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

// ImportWallet imports password entries from a Chrome-based browser CSV export.
// The expected header is: name,url,username,password,note
// Entries are de-duplicated on (website, username) per user.
func ImportWallet(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserIDFromQuery(r)
	if userID == 0 {
		userID = 1
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // allow variable column counts

	header, err := reader.Read()
	if err != nil {
		http.Error(w, "Failed to read CSV header. Is this a valid passwords export?", http.StatusBadRequest)
		return
	}

	idx := csvColumnIndex(header)
	if idx["username"] < 0 || idx["password"] < 0 {
		http.Error(w, "CSV is missing required 'username' or 'password' columns", http.StatusBadRequest)
		return
	}

	// Collect existing (website|username) pairs for de-duplication.
	existing := make(map[string]bool)
	rows, err := db.WalletDB.Query("SELECT website, username FROM wallet WHERE user_id = ?", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var website, username string
			rows.Scan(&website, &username)
			existing[dedupKey(website, username)] = true
		}
	}

	imported := 0
	skipped := 0
	errors := []string{}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if len(errors) < 10 {
				errors = append(errors, fmt.Sprintf("Failed to read row: %v", err))
			}
			continue
		}

		username := field(record, idx["username"])
		password := field(record, idx["password"])
		website := field(record, idx["name"])
		link := field(record, idx["url"])
		note := field(record, idx["note"])

		if website == "" {
			website = hostFromURL(link)
		}
		if website == "" {
			website = link
		}

		// Skip blank rows.
		if website == "" && username == "" && password == "" {
			continue
		}

		key := dedupKey(website, username)
		if existing[key] {
			skipped++
			continue
		}

		_, err = db.WalletDB.Exec(
			"INSERT INTO wallet (user_id, username, password, website, description) VALUES (?, ?, ?, ?, ?)",
			userID, username, password, website, note,
		)
		if err != nil {
			if len(errors) < 10 {
				errors = append(errors, fmt.Sprintf("Failed to import %s: %v", website, err))
			}
			continue
		}

		existing[key] = true
		imported++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.WalletImportResult{
		Imported: imported,
		Skipped:  skipped,
		Errors:   errors,
	})
}

// csvColumnIndex maps known column names to their position, defaulting to -1.
func csvColumnIndex(header []string) map[string]int {
	idx := map[string]int{"name": -1, "url": -1, "username": -1, "password": -1, "note": -1}
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "name", "title":
			idx["name"] = i
		case "url", "origin":
			idx["url"] = i
		case "username", "login":
			idx["username"] = i
		case "password":
			idx["password"] = i
		case "note", "notes", "comment":
			idx["note"] = i
		}
	}
	return idx
}

func field(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}

func dedupKey(website, username string) string {
	return strings.ToLower(website) + "|" + strings.ToLower(username)
}

func hostFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

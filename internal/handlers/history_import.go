package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

// chromeTimestampToTime converts Chrome/WebKit timestamp (microseconds since 1601-01-01) to time.Time.
func chromeTimestampToTime(chromeTS int64) time.Time {
	// Chrome timestamps are microseconds since January 1, 1601 00:00:00 UTC
	// Unix epoch offset: 11644473600 seconds
	unixSec := (chromeTS / 1000000) - 11644473600
	return time.Unix(unixSec, 0)
}

func ImportHistory(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserIDFromQuery(r)
	if userID == 0 {
		userID = 1
	}

	err := r.ParseMultipartForm(64 << 20) // 64MB limit for SQLite files
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to a temp file so SQLite can open it
	tmpDir, err := os.MkdirTemp("", "chrome-history-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, header.Filename)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		http.Error(w, "Failed to write temp file", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	// Open the uploaded SQLite database as read-only
	chromeDB, err := sql.Open("sqlite3", "file:"+tmpPath+"?mode=ro")
	if err != nil {
		http.Error(w, "Failed to open SQLite file. Is this a valid Chrome History file?", http.StatusBadRequest)
		return
	}
	defer chromeDB.Close()

	// Verify it's a Chrome History database by checking for the urls table
	var tableName string
	err = chromeDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='urls'").Scan(&tableName)
	if err != nil {
		http.Error(w, "Not a valid Chrome History file: 'urls' table not found", http.StatusBadRequest)
		return
	}

	// Collect existing URLs for deduplication
	existingURLs := make(map[string]bool)
	existingRows, err := db.HistoryDB.Query("SELECT url FROM history WHERE user_id = ?", userID)
	if err == nil {
		defer existingRows.Close()
		for existingRows.Next() {
			var url string
			existingRows.Scan(&url)
			existingURLs[url] = true
		}
	}

	// Chrome's urls table: id, url, title, visit_count, typed_count, last_visit_time, hidden, favicon_id
	// Chrome's visits table: id, url, visit_time, from_visit, transition, segment_id, is_indexed
	//
	// We join urls with visits to get individual visit records with timestamps.
	// Chrome timestamps are microseconds since 1601-01-01.

	query := `SELECT u.url, u.title, v.visit_time
		FROM urls u
		INNER JOIN visits v ON v.url = u.id
		ORDER BY v.visit_time DESC
		LIMIT 50000`

	chromeRows, err := chromeDB.Query(query)
	if err != nil {
		// Fallback: try without joins, just use urls table
		query = `SELECT url, title, last_visit_time FROM urls ORDER BY last_visit_time DESC`
		chromeRows, err = chromeDB.Query(query)
		if err != nil {
			http.Error(w, "Failed to query Chrome History: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	defer chromeRows.Close()

	imported := 0
	skipped := 0
	errors := []string{}

	for chromeRows.Next() {
		var url, title string
		var visitTime int64

		if err := chromeRows.Scan(&url, &title, &visitTime); err != nil {
			continue
		}

		if url == "" {
			continue
		}

		if existingURLs[url] {
			skipped++
			continue
		}

		visitedAt := chromeTimestampToTime(visitTime)
		// Skip entries with obviously wrong timestamps (before year 2000)
		if visitedAt.Year() < 2000 {
			skipped++
			continue
		}

		_, err := db.HistoryDB.Exec(
			"INSERT INTO history (user_id, url, title, visited_at, duration) VALUES (?, ?, ?, ?, ?)",
			userID, url, title, visitedAt, 0,
		)
		if err != nil {
			if len(errors) < 10 {
				errors = append(errors, fmt.Sprintf("Failed to import %s: %v", url, err))
			}
			continue
		}

		existingURLs[url] = true
		imported++
	}

	if err := chromeRows.Err(); err != nil {
		if len(errors) < 10 {
			errors = append(errors, fmt.Sprintf("Error reading Chrome history: %v", err))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.HistoryImportResult{
		Imported: imported,
		Skipped:  skipped,
		Errors:   errors,
	})
}

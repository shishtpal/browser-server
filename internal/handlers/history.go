package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

const defaultGroupedHistoryLimit = 100

// sqliteTimeFormats mirrors the layouts go-sqlite3 uses to (de)serialize
// DATETIME values. Aggregates like MAX(visited_at) lose the column's declared
// type, so the driver hands them back as strings that we parse ourselves.
var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

func parseSQLiteTime(value string) time.Time {
	for _, layout := range sqliteTimeFormats {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetGroupedHistory returns history aggregated by URL, searched and paginated
// entirely on the server so clients never have to load every row at once.
func GetGroupedHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	column := r.URL.Query().Get("column") // "all" (default), "title", or "url"
	limit := helpers.GetLimitFromQuery(r, defaultGroupedHistoryLimit)
	offset := helpers.GetOffsetFromQuery(r)

	where := "WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	// Each whitespace-separated term must match (AND), mirroring the previous
	// client-side search behaviour.
	for _, term := range strings.Fields(search) {
		like := "%" + term + "%"
		switch column {
		case "title":
			where += " AND title LIKE ?"
			args = append(args, like)
		case "url":
			where += " AND url LIKE ?"
			args = append(args, like)
		default:
			where += " AND (title LIKE ? OR url LIKE ?)"
			args = append(args, like, like)
		}
	}

	var total int
	if err := db.HistoryDB.QueryRow("SELECT COUNT(DISTINCT url) FROM history "+where, args...).Scan(&total); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// SQLite's bare-column rule means `title` is taken from the row holding
	// MAX(visited_at), i.e. the most recent visit for that URL.
	query := "SELECT url, title, COUNT(*), COALESCE(SUM(duration), 0), MAX(visited_at) FROM history " +
		where + " GROUP BY url ORDER BY MAX(visited_at) DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.HistoryDB.Query(query, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	entries := []models.GroupedHistoryEntry{}
	for rows.Next() {
		var entry models.GroupedHistoryEntry
		// MAX(visited_at) comes back as a string (aggregates lose the column's
		// DATETIME type), so scan it as text and parse it ourselves.
		var lastVisited sql.NullString
		if err := rows.Scan(&entry.URL, &entry.Title, &entry.Count, &entry.TotalDuration, &lastVisited); err != nil {
			continue
		}
		if lastVisited.Valid {
			entry.LastVisited = parseSQLiteTime(lastVisited.String)
		}
		entries = append(entries, entry)
	}

	json.NewEncoder(w).Encode(models.GroupedHistoryResponse{
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

func GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	urlFilter := r.URL.Query().Get("url")
	limit := helpers.GetLimitFromQuery(r, 0)
	offset := helpers.GetOffsetFromQuery(r)

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

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	rows, err := db.HistoryDB.Query(query, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	history := []models.History{}
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
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", entry.UserID)
	v.URL("url", entry.URL)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	if entry.VisitedAt.IsZero() {
		entry.VisitedAt = time.Now()
	}

	result, err := db.HistoryDB.Exec("INSERT INTO history (user_id, url, title, visited_at, duration) VALUES (?, ?, ?, ?, ?)",
		entry.UserID, entry.URL, entry.Title, entry.VisitedAt, entry.Duration)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
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
		helpers.WriteError(w, http.StatusNotFound, "History entry not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func DeleteHistory(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	result, err := db.HistoryDB.Exec("DELETE FROM history WHERE id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		helpers.WriteError(w, http.StatusNotFound, "History entry not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

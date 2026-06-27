package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

const (
	defaultOmniboxSearchLimit = 6
	maxOmniboxSearchLimit     = 10
)

func clampOmniboxLimit(limit int) int {
	if limit <= 0 {
		return defaultOmniboxSearchLimit
	}
	if limit > maxOmniboxSearchLimit {
		return maxOmniboxSearchLimit
	}
	return limit
}

// SearchOmnibox returns URL suggestions from server-side bookmarks and grouped
// history. It intentionally does not depend on Chrome's local history store.
func SearchOmnibox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := clampOmniboxLimit(helpers.GetLimitFromQuery(r, defaultOmniboxSearchLimit))

	if query == "" {
		json.NewEncoder(w).Encode([]models.OmniboxSearchResult{})
		return
	}

	historyResults, err := searchOmniboxHistory(userID, query, limit)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	bookmarkResults, err := searchOmniboxBookmarks(userID, query, limit)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	results := selectOmniboxResults(historyResults, bookmarkResults, limit)
	json.NewEncoder(w).Encode(results)
}

func selectOmniboxResults(historyResults, bookmarkResults []models.OmniboxSearchResult, limit int) []models.OmniboxSearchResult {
	if len(historyResults) == 0 && len(bookmarkResults) == 0 {
		return []models.OmniboxSearchResult{}
	}
	if len(historyResults) == 0 {
		return trimOmniboxResults(bookmarkResults, limit)
	}
	if len(bookmarkResults) == 0 {
		return trimOmniboxResults(historyResults, limit)
	}

	bookmarkTarget := limit / 2
	historyTarget := limit - bookmarkTarget

	selected := []models.OmniboxSearchResult{}
	selected = append(selected, takeOmniboxResults(historyResults, historyTarget)...)
	selected = append(selected, takeOmniboxResults(bookmarkResults, bookmarkTarget)...)

	if len(selected) < limit {
		selected = append(selected, historyResults[minInt(len(historyResults), historyTarget):]...)
	}
	if len(selected) < limit {
		selected = append(selected, bookmarkResults[minInt(len(bookmarkResults), bookmarkTarget):]...)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		return omniboxResultTime(selected[i]).After(omniboxResultTime(selected[j]))
	})

	return trimOmniboxResults(selected, limit)
}

func takeOmniboxResults(results []models.OmniboxSearchResult, limit int) []models.OmniboxSearchResult {
	return results[:minInt(len(results), limit)]
}

func trimOmniboxResults(results []models.OmniboxSearchResult, limit int) []models.OmniboxSearchResult {
	if len(results) <= limit {
		return results
	}
	return results[:limit]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func searchOmniboxHistory(userID int, search string, limit int) ([]models.OmniboxSearchResult, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	for _, term := range strings.Fields(search) {
		like := "%" + term + "%"
		where += " AND (title LIKE ? OR url LIKE ?)"
		args = append(args, like, like)
	}

	query := "SELECT url, title, COUNT(*), MAX(visited_at) FROM history " +
		where + " GROUP BY url ORDER BY MAX(visited_at) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.HistoryDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.OmniboxSearchResult{}
	for rows.Next() {
		var result models.OmniboxSearchResult
		var lastVisited sql.NullString
		if err := rows.Scan(&result.URL, &result.Title, &result.VisitCount, &lastVisited); err != nil {
			continue
		}
		if lastVisited.Valid {
			parsed := parseSQLiteTime(lastVisited.String)
			result.LastVisited = &parsed
		}
		result.Source = "history"
		results = append(results, result)
	}

	return results, rows.Err()
}

func searchOmniboxBookmarks(userID int, search string, limit int) ([]models.OmniboxSearchResult, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	for _, term := range strings.Fields(search) {
		like := "%" + term + "%"
		where += " AND (title LIKE ? OR url LIKE ? OR description LIKE ? OR folder_path LIKE ? OR tags LIKE ?)"
		args = append(args, like, like, like, like, like)
	}

	query := "SELECT title, url, description, tags, folder_path, updated_at FROM bookmarks " +
		where + " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.BookmarkDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.OmniboxSearchResult{}
	for rows.Next() {
		var result models.OmniboxSearchResult
		var tagsJSON string
		var updatedAt time.Time
		if err := rows.Scan(&result.Title, &result.URL, &result.Description, &tagsJSON, &result.FolderPath, &updatedAt); err != nil {
			continue
		}
		result.Source = "bookmark"
		result.Tags = helpers.ParseTagsFromJSON(tagsJSON)
		result.UpdatedAt = &updatedAt
		results = append(results, result)
	}

	return results, rows.Err()
}

func omniboxResultTime(result models.OmniboxSearchResult) time.Time {
	if result.Source == "history" && result.LastVisited != nil {
		return *result.LastVisited
	}
	if result.UpdatedAt != nil {
		return *result.UpdatedAt
	}
	return time.Time{}
}

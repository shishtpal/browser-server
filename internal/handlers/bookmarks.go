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

func GetBookmarks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	tagsFilter := r.URL.Query().Get("tags")

	query := "SELECT id, user_id, title, url, description, tags, folder_path, created_at, updated_at FROM bookmarks WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	folderPathFilter := r.URL.Query().Get("folder_path")
	if folderPathFilter != "" {
		query += " AND folder_path LIKE ?"
		args = append(args, folderPathFilter+"%")
	}

	rows, err := db.BookmarkDB.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	bookmarks := []models.BookmarkResponse{}
	for rows.Next() {
		var bookmark models.Bookmark
		err := rows.Scan(&bookmark.ID, &bookmark.UserID, &bookmark.Title, &bookmark.URL, &bookmark.Description, &bookmark.Tags, &bookmark.FolderPath, &bookmark.CreatedAt, &bookmark.UpdatedAt)
		if err != nil {
			continue
		}

		response := models.BookmarkResponse{
			ID:          bookmark.ID,
			UserID:      bookmark.UserID,
			Title:       bookmark.Title,
			URL:         bookmark.URL,
			Description: bookmark.Description,
			Tags:        helpers.ParseTagsFromJSON(bookmark.Tags),
			FolderPath:  bookmark.FolderPath,
			CreatedAt:   bookmark.CreatedAt,
			UpdatedAt:   bookmark.UpdatedAt,
		}

		if tagsFilter != "" {
			filterTags := strings.Split(tagsFilter, ",")
			hasTag := false
			for _, filterTag := range filterTags {
				for _, bookmarkTag := range response.Tags {
					if strings.EqualFold(strings.TrimSpace(filterTag), bookmarkTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		bookmarks = append(bookmarks, response)
	}

	json.NewEncoder(w).Encode(bookmarks)
}

func CreateBookmark(w http.ResponseWriter, r *http.Request) {
	var bookmark models.BookmarkResponse
	if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tagsJSON := helpers.TagsToJSON(bookmark.Tags)

	result, err := db.BookmarkDB.Exec("INSERT INTO bookmarks (user_id, title, url, description, tags, folder_path) VALUES (?, ?, ?, ?, ?, ?)",
		bookmark.UserID, bookmark.Title, bookmark.URL, bookmark.Description, tagsJSON, bookmark.FolderPath)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	bookmark.ID = int(id)
	bookmark.CreatedAt = time.Now()
	bookmark.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bookmark)
}

func GetBookmarkByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var bookmark models.Bookmark
	err := db.BookmarkDB.QueryRow("SELECT id, user_id, title, url, description, tags, folder_path, created_at, updated_at FROM bookmarks WHERE id = ?", id).
		Scan(&bookmark.ID, &bookmark.UserID, &bookmark.Title, &bookmark.URL, &bookmark.Description, &bookmark.Tags, &bookmark.FolderPath, &bookmark.CreatedAt, &bookmark.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	response := models.BookmarkResponse{
		ID:          bookmark.ID,
		UserID:      bookmark.UserID,
		Title:       bookmark.Title,
		URL:         bookmark.URL,
		Description: bookmark.Description,
		Tags:        helpers.ParseTagsFromJSON(bookmark.Tags),
		FolderPath:  bookmark.FolderPath,
		CreatedAt:   bookmark.CreatedAt,
		UpdatedAt:   bookmark.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var bookmark models.BookmarkResponse
	if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tagsJSON := helpers.TagsToJSON(bookmark.Tags)

	_, err := db.BookmarkDB.Exec("UPDATE bookmarks SET user_id = ?, title = ?, url = ?, description = ?, tags = ?, folder_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		bookmark.UserID, bookmark.Title, bookmark.URL, bookmark.Description, tagsJSON, bookmark.FolderPath, id)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	bookmark.ID = id
	bookmark.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookmark)
}

func DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	result, err := db.BookmarkDB.Exec("DELETE FROM bookmarks WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

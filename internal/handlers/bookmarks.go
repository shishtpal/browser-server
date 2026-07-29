package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"browser-server/internal/bookmark"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func GetBookmarks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bookmarks, err := bookmark.List(r.Context(), bookmark.ListOptions{
		UserID:           helpers.GetUserIDFromQuery(r),
		TagsFilter:       r.URL.Query().Get("tags"),
		FolderPathPrefix: r.URL.Query().Get("folder_path"),
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(bookmark.Responses(bookmarks))
}

func CreateBookmark(w http.ResponseWriter, r *http.Request) {
	var input models.BookmarkResponse
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if !validateBookmarkInput(w, &input) {
		return
	}

	id, inserted, err := bookmark.Create(r.Context(), bookmark.CreateInput{
		UserID:      input.UserID,
		Title:       input.Title,
		URL:         input.URL,
		Description: input.Description,
		FolderPath:  input.FolderPath,
		CaptureID:   input.CaptureID,
		Tags:        input.Tags,
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if !inserted {
		stored, err := bookmark.GetByCaptureID(r.Context(), input.UserID, input.CaptureID)
		if errors.Is(err, bookmark.ErrNotFound) {
			helpers.WriteError(w, http.StatusNotFound, "Bookmark not found")
			return
		}
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
			return
		}
		response := bookmark.Response(stored)
		response.CaptureID = input.CaptureID
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	input.ID = int(id)
	input.CreatedAt = time.Now()
	input.UpdatedAt = input.CreatedAt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func GetBookmarkByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)
	b, err := bookmark.GetByID(r.Context(), id)
	if errors.Is(err, bookmark.ErrNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "Bookmark not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookmark.Response(b))
}

func UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)
	var input models.BookmarkResponse
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !validateBookmarkInput(w, &input) {
		return
	}

	err := bookmark.Update(r.Context(), id, bookmark.CreateInput{
		UserID:      input.UserID,
		Title:       input.Title,
		URL:         input.URL,
		Description: input.Description,
		FolderPath:  input.FolderPath,
		Tags:        input.Tags,
	})
	if errors.Is(err, bookmark.ErrBookmarkNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "Bookmark not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	input.ID = id
	input.UpdatedAt = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

func DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	deleted, err := bookmark.Delete(r.Context(), helpers.GetIDFromPath(r))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if !deleted {
		helpers.WriteError(w, http.StatusNotFound, "Bookmark not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateBookmarkInput(w http.ResponseWriter, input *models.BookmarkResponse) bool {
	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("title", input.Title)
	v.URL("url", input.URL)
	if title, err := bookmark.ValidateTitle(input.Title); err != nil {
		v.Fields()["title"] = err.Error()
	} else {
		input.Title = title
	}
	input.Tags = bookmark.NormalizeTags(input.Tags)
	if err := bookmark.ValidateDescription(input.Description); err != nil {
		v.Fields()["description"] = err.Error()
	}
	if err := bookmark.ValidateURL(input.URL); err != nil {
		v.Fields()["url"] = err.Error()
	}
	if err := bookmark.ValidateFolderPath(input.FolderPath); err != nil {
		v.Fields()["folder_path"] = err.Error()
	}
	if err := bookmark.ValidateTags(input.Tags); err != nil {
		v.Fields()["tags"] = err.Error()
	}
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return false
	}
	return true
}

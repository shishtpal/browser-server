package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"browser-server/internal/helpers"
	"browser-server/internal/prompt"
)

func GetPrompts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	records, err := prompt.List(r.Context(), prompt.ListQuery{
		UserID: userID,
		Query:  r.URL.Query().Get("q"),
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(prompt.Responses(records))
}

func CreatePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input struct {
		UserID      int      `json:"user_id"`
		Title       string   `json:"title"`
		Content     string   `json:"content"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("title", input.Title)
	v.Required("content", input.Content)
	v.MaxLength("title", input.Title, prompt.MaxTitleLength)
	v.MaxLength("content", input.Content, prompt.MaxContentLength)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	id, _, err := prompt.Create(r.Context(), prompt.CreateInput{
		UserID:      input.UserID,
		Title:       input.Title,
		Content:     input.Content,
		Description: input.Description,
		Tags:        input.Tags,
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rec, err := prompt.GetByID(r.Context(), int(id))
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(prompt.Response(rec))
}

func GetPromptByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	rec, err := prompt.GetByID(r.Context(), id)
	if err == prompt.ErrNotFound {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(prompt.Response(rec))
}

func UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	rec, err := prompt.GetByID(r.Context(), id)
	if err == prompt.ErrNotFound {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	_ = rec

	var input struct {
		Title       *string  `json:"title"`
		Content     *string  `json:"content"`
		Description *string  `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	builder := prompt.NewUpdateBuilder()

	if input.Title != nil {
		v := helpers.NewValidator()
		v.Required("title", *input.Title)
		v.MaxLength("title", *input.Title, prompt.MaxTitleLength)
		if !v.OK() {
			helpers.WriteValidationError(w, v.Fields())
			return
		}
		builder.Set("title", *input.Title)
	}
	if input.Content != nil {
		if err := prompt.ValidateContent(*input.Content); err != nil {
			helpers.WriteValidationError(w, map[string]string{
				"content": "must be " + strconv.Itoa(prompt.MaxContentLength) + " characters or fewer",
			})
			return
		}
		builder.Set("content", *input.Content)
	}
	if input.Description != nil {
		builder.Set("description", *input.Description)
	}
	if input.Tags != nil {
		builder.Set("tags", helpers.TagsToJSON(input.Tags))
	}

	if builder.Empty() {
		helpers.WriteError(w, http.StatusBadRequest, "No updatable fields provided")
		return
	}
	if err := builder.Exec(r.Context(), id); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rec, err = prompt.GetByID(r.Context(), id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	json.NewEncoder(w).Encode(prompt.Response(rec))
}

func DeletePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	deleted, err := prompt.Delete(r.Context(), id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if !deleted {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Fuzzy Search ───────────────────────────────────────

func SearchPrompts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	limit := helpers.GetLimitFromQuery(r, 10)
	if limit > 20 {
		limit = 20
	}

	records, err := prompt.List(r.Context(), prompt.ListQuery{
		UserID:     userID,
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:      limit,
		TitleFirst: true,
	})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(prompt.Responses(records))
}

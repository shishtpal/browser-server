package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/todo"
)

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	showArchived := parseBool(r.URL.Query().Get("archived"))
	parentID := parseInt(r.URL.Query().Get("parent_id"))

	filter := todo.ListFilter{
		UserID:       helpers.GetUserIDFromQuery(r),
		Status:       r.URL.Query().Get("status"),
		Domain:       r.URL.Query().Get("domain"),
		Priority:     r.URL.Query().Get("priority"),
		Tag:          r.URL.Query().Get("tag"),
		ParentID:     parentID,
		SortField:    r.URL.Query().Get("sort"),
		SortOrder:    r.URL.Query().Get("order"),
		ShowArchived: showArchived,
	}

	repo := todo.NewRepository(db.TodoDB)
	todos, err := repo.List(filter)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	json.NewEncoder(w).Encode(todos)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	createInput := todo.CreateInput{
		UserID:      extractInt(input, "user_id"),
		Title:       extractString(input, "title"),
		Description: extractString(input, "description"),
		Domain:      extractString(input, "domain"),
		CaptureID:   extractString(input, "capture_id"),
		ParentID:    int(extractInt(input, "parent_id")),
		Priority:    extractString(input, "priority"),
		Status:      extractString(input, "status"),
		Color:       extractString(input, "color"),
		Tags:        extractStringSlice(input, "tags"),
	}
	if d, ok := input["start_date"].(string); ok && d != "" {
		createInput.StartDate = &d
	}
	if d, ok := input["end_date"].(string); ok && d != "" {
		createInput.EndDate = &d
	}

	repo := todo.NewRepository(db.TodoDB)
	result, err := repo.Create(&createInput)
	if err != nil {
		if validationErr, ok := err.(*todo.ValidationError); ok {
			helpers.WriteValidationError(w, validationErr.Fields)
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	resp := models.TodoResponse{
		Todo: models.Todo{
			ID:          int(result.ID),
			UserID:      result.UserID,
			Title:       result.Title,
			Description: result.Description,
			Status:      result.Status,
			Priority:    result.Priority,
			ParentID:    result.ParentID,
			Position:    result.Position,
			CreatedAt:   result.CreatedAt,
			UpdatedAt:   result.UpdatedAt,
			StartDate:   parseDatePtr(result.StartDate),
			EndDate:     parseDatePtr(result.EndDate),
		},
		Tags: result.Tags,
	}
	if result.Color != nil {
		resp.Todo.Color = *result.Color
	}

	w.Header().Set("Content-Type", "application/json")
	if result.IsDuplicate {
		json.NewEncoder(w).Encode(resp)
	} else {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func GetTodoByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	repo := todo.NewRepository(db.TodoDB)
	resp, err := repo.GetByID(id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if resp == nil {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var input struct {
		UserID         *int            `json:"user_id"`
		Title          *string         `json:"title"`
		Description    *string         `json:"description"`
		Domain         *string         `json:"domain"`
		ScreenshotPath *string         `json:"screenshot_path"`
		Pinned         *bool           `json:"pinned"`
		Status         *string         `json:"status"`
		Priority       *string         `json:"priority"`
		Color          *string         `json:"color"`
		StartDate      json.RawMessage `json:"start_date"`
		EndDate        json.RawMessage `json:"end_date"`
		Rrule          *string         `json:"rrule"`
		Tags           *[]string       `json:"tags"`
		Position       *int            `json:"position"`
		ParentID       json.RawMessage `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Fetch existing todo for user_id/ownership checks
	repo := todo.NewRepository(db.TodoDB)
	existing, err := repo.GetByID(id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing == nil {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}

	updateInput := todo.UpdateInput{ID: id, UserID: existing.Todo.UserID}
	if input.UserID != nil {
		updateInput.UserID = *input.UserID
	}
	updateInput.Title = input.Title
	updateInput.Description = input.Description
	updateInput.Domain = input.Domain
	updateInput.ScreenshotPath = input.ScreenshotPath
	updateInput.Pinned = input.Pinned
	updateInput.Status = input.Status
	updateInput.Priority = input.Priority
	updateInput.Color = input.Color
	updateInput.Rrule = input.Rrule
	updateInput.Tags = input.Tags
	updateInput.Position = input.Position

	if input.ParentID != nil {
		if string(input.ParentID) == "null" {
			zero := 0
			updateInput.ParentID = &zero
		} else {
			var parentID int
			if err := json.Unmarshal(input.ParentID, &parentID); err != nil {
				helpers.WriteValidationError(w, map[string]string{"parent_id": "must be a positive integer or null"})
				return
			}
			updateInput.ParentID = &parentID
		}
	}
	if input.StartDate != nil {
		if string(input.StartDate) == "null" {
			clear := ""
			updateInput.StartDate = &clear
		} else {
			var raw string
			if err := json.Unmarshal(input.StartDate, &raw); err != nil {
				helpers.WriteValidationError(w, map[string]string{"start_date": "must be a date string or null"})
				return
			}
			updateInput.StartDate = &raw
		}
	}
	if input.EndDate != nil {
		if string(input.EndDate) == "null" {
			clear := ""
			updateInput.EndDate = &clear
		} else {
			var raw string
			if err := json.Unmarshal(input.EndDate, &raw); err != nil {
				helpers.WriteValidationError(w, map[string]string{"end_date": "must be a date string or null"})
				return
			}
			updateInput.EndDate = &raw
		}
	}

	result, err := repo.Update(&updateInput)
	if err != nil {
		if validationErr, ok := err.(*todo.ValidationError); ok {
			helpers.WriteValidationError(w, validationErr.Fields)
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	resp := models.TodoResponse{
		Todo: models.Todo{
			ID:             int(result.ID),
			UserID:         result.UserID,
			Title:          result.Title,
			Description:    result.Description,
			Domain:         "",
			ScreenshotPath: "",
			Pinned:         result.Pinned,
			Status:         result.Status,
			Priority:       result.Priority,
			ParentID:       result.ParentID,
			Position:       result.Position,
			Rrule:          "",
			CreatedAt:      result.CreatedAt,
			UpdatedAt:      result.UpdatedAt,
			StartDate:      parseDatePtr(result.StartDate),
			EndDate:        parseDatePtr(result.EndDate),
		},
		Tags: result.Tags,
	}
	if result.Color != nil {
		resp.Todo.Color = *result.Color
	}
	if result.Domain != nil {
		resp.Todo.Domain = *result.Domain
	}
	if result.ScreenshotPath != nil {
		resp.Todo.ScreenshotPath = *result.ScreenshotPath
	}
	if result.Rrule != nil {
		resp.Todo.Rrule = *result.Rrule
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	repo := todo.NewRepository(db.TodoDB)
	err := repo.Delete(id)
	if err != nil {
		if err.Error() == "not found" {
			helpers.WriteError(w, http.StatusNotFound, "Todo not found")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractString safely extracts a string value from a map.
func extractString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractInt safely extracts an int value from a map.
func extractInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch f := v.(type) {
		case int:
			return f
		case int64:
			return int(f)
		case float64:
			return int(f)
		case float32:
			return int(f)
		}
	}
	return 0
}

// extractStringSlice safely extracts a string slice from a map.
func extractStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case []any:
			result := []string{}
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		default:
			return []string{}
		}
	}
	return []string{}
}

// parseDatePtr converts a *string date to *time.Time using the todo package's date parser.
func parseDatePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return todo.ParseDate(*s)
}

// parseBool safely parses a string as a bool.
func parseBool(s string) bool {
	if s == "" {
		return false
	}
	return s == "true" || s == "1"
}

// parseInt safely parses a string as an int.
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int(c-'0')
	}
	return v
}

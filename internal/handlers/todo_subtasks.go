package handlers

import (
	"encoding/json"
	"net/http"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/todo"
)

func GetSubtasks(w http.ResponseWriter, r *http.Request) {
	parentID := helpers.GetIDFromPath(r)

	repo := todo.NewRepository(db.TodoDB)
	subtasks, err := repo.List(todo.ListFilter{ParentID: parentID, SortField: "position"})
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subtasks)
}

func CreateSubtask(w http.ResponseWriter, r *http.Request) {
	parentID := helpers.GetIDFromPath(r)

	var input struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Domain      string   `json:"domain"`
		UserID      int      `json:"user_id"`
		Priority    string   `json:"priority"`
		StartDate   *string  `json:"start_date"`
		EndDate     *string  `json:"end_date"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	createInput := todo.CreateInput{
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		Domain:      input.Domain,
		ParentID:    parentID,
		Priority:    input.Priority,
		Status:      "pending",
		Tags:        input.Tags,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

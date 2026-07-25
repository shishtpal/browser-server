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

func GetSubtasks(w http.ResponseWriter, r *http.Request) {
	parentID := helpers.GetIDFromPath(r)

	w.Header().Set("Content-Type", "application/json")

	rows, err := db.TodoDB.Query("SELECT "+todoColumns+" FROM todos WHERE parent_id = ? ORDER BY pinned DESC, position ASC", parentID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	subtasks := make([]models.TodoResponse, 0)
	for rows.Next() {
		todo, tagsJSON, err := scanTodo(rows)
		if err != nil {
			continue
		}
		subtasks = append(subtasks, todoToResponse(todo, tagsJSON))
	}

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

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("title", input.Title)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	if input.Priority == "" {
		input.Priority = "medium"
	}
	validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
	if !validPriorities[input.Priority] {
		input.Priority = "medium"
	}

	tagsJSON := helpers.TagsToJSON(input.Tags)
	var startDateDB *time.Time
	if input.StartDate != nil && *input.StartDate != "" {
		startDateDB = parseDate(*input.StartDate)
	}
	var endDateDB *time.Time
	if input.EndDate != nil && *input.EndDate != "" {
		endDateDB = parseDate(*input.EndDate)
	}

	var maxPos sql.NullInt64
	db.TodoDB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM todos WHERE parent_id = ? AND user_id = ?", parentID, input.UserID).Scan(&maxPos)
	position := int(maxPos.Int64) + 1

	pid := parentID
	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, domain, status, priority, start_date, end_date, tags, parent_id, position)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Title, input.Description, input.Domain, input.Priority, startDateDB, endDateDB, tagsJSON, &pid, position)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	id, _ := result.LastInsertId()
	now := time.Now()
	todo := models.Todo{
		ID:          int(id),
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		Domain:      input.Domain,
		Status:      "pending",
		Priority:    input.Priority,
		StartDate:   startDateDB,
		EndDate:     endDateDB,
		ParentID:    &pid,
		Position:    position,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	resp := todoToResponse(todo, tagsJSON)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

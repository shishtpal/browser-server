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

	rows, err := db.TodoDB.Query("SELECT id, user_id, title, description, domain, screenshot_path, completed, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE parent_id = ? ORDER BY position ASC", parentID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var subtasks []models.TodoResponse
	for rows.Next() {
		var todo models.Todo
		var tagsDB string
		var dueDate sql.NullTime
		var pid sql.NullInt64
		err := rows.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.Priority, &dueDate, &tagsDB, &pid, &todo.Position, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			continue
		}
		if dueDate.Valid {
			dt := dueDate.Time
			todo.DueDate = &dt
		}
		if pid.Valid {
			cpid := int(pid.Int64)
			todo.ParentID = &cpid
		}
		subtasks = append(subtasks, models.TodoResponse{
			Todo:  todo,
			Tags:  helpers.ParseTagsFromJSON(tagsDB),
		})
	}

	json.NewEncoder(w).Encode(subtasks)
}

func CreateSubtask(w http.ResponseWriter, r *http.Request) {
	parentID := helpers.GetIDFromPath(r)

	var input struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		Domain      string     `json:"domain"`
		UserID      int        `json:"user_id"`
		Priority    string     `json:"priority"`
		DueDate     *string    `json:"due_date"`
		Tags        []string   `json:"tags"`
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
	var dueDateDB *time.Time
	if input.DueDate != nil && *input.DueDate != "" {
		parsed := parseDueDate(*input.DueDate)
		dueDateDB = parsed
	}

	var maxPos sql.NullInt64
	db.TodoDB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM todos WHERE parent_id = ? AND user_id = ?", parentID, input.UserID).Scan(&maxPos)
	position := int(maxPos.Int64) + 1

	pid := parentID
	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, domain, completed, priority, due_date, tags, parent_id, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Title, input.Description, input.Domain, false, input.Priority, dueDateDB, tagsJSON, &pid, position)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	id, _ := result.LastInsertId()
	now := time.Now()
	todo := models.Todo{
		ID:         int(id),
		UserID:     input.UserID,
		Title:      input.Title,
		Description: input.Description,
		Domain:     input.Domain,
		Completed:  false,
		Priority:   input.Priority,
		DueDate:    dueDateDB,
		ParentID:   &pid,
		Position:   position,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	resp := models.TodoResponse{Todo: todo, Tags: helpers.ParseTagsFromJSON(tagsJSON)}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

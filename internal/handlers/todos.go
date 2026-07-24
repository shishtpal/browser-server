package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func parseDueDate(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t2, err2 := time.Parse("2006-01-02", raw)
		if err2 != nil {
			return nil
		}
		return &t2
	}
	return &t
}

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	completedStr := r.URL.Query().Get("completed")
	domain := r.URL.Query().Get("domain")
	priority := r.URL.Query().Get("priority")
	tagFilter := r.URL.Query().Get("tag")
	parentIDStr := r.URL.Query().Get("parent_id")
	archived, _ := strconv.ParseBool(r.URL.Query().Get("archived"))
	sortField := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")

	query := "SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE archived = ?"
	args := []interface{}{archived}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if completedStr != "" {
		completed, _ := strconv.ParseBool(completedStr)
		query += " AND completed = ?"
		args = append(args, completed)
	}

	if domain != "" {
		query += " AND domain = ?"
		args = append(args, domain)
	}

	if priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}

	if tagFilter != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%"+tagFilter+"%")
	}

	if parentIDStr != "" {
		pid, _ := strconv.Atoi(parentIDStr)
		if pid == 0 {
			query += " AND parent_id IS NULL"
		} else {
			query += " AND parent_id = ?"
			args = append(args, pid)
		}
	} else {
		query += " AND parent_id IS NULL"
	}

	if sortField != "" {
		switch sortField {
		case "position", "due_date", "created_at":
			query += " ORDER BY pinned DESC, " + sortField
		case "priority":
			query += " ORDER BY pinned DESC, CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END"
		case "title":
			query += " ORDER BY pinned DESC, title"
		default:
			query += " ORDER BY pinned DESC, position"
		}
	} else {
		query += " ORDER BY pinned DESC, position"
	}

	if sortOrder == "desc" {
		query += " DESC"
	} else {
		query += " ASC"
	}

	rows, err := db.TodoDB.Query(query, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	todos := make([]models.TodoResponse, 0)
	for rows.Next() {
		var todo models.Todo
		var tagsJSON string
		var dueDate sql.NullTime
		var parentID sql.NullInt64
		err := rows.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.Pinned, &todo.Archived, &todo.Priority, &dueDate, &tagsJSON, &parentID, &todo.Position, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			continue
		}
		if dueDate.Valid {
			t := dueDate.Time
			todo.DueDate = &t
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			todo.ParentID = &pid
		}
		resp := models.TodoResponse{
			Todo: todo,
			Tags: helpers.ParseTagsFromJSON(tagsJSON),
		}
		if tagFilter == "" {
			childRows, err := db.TodoDB.Query("SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE parent_id = ? AND archived = ? ORDER BY pinned DESC, position ASC", todo.ID, archived)
			if err == nil {
				var children []models.TodoResponse
				for childRows.Next() {
					var child models.Todo
					var childTags string
					var childDue sql.NullTime
					var childParent sql.NullInt64
					err := childRows.Scan(&child.ID, &child.UserID, &child.Title, &child.Description, &child.Domain, &child.ScreenshotPath, &child.Completed, &child.Pinned, &child.Archived, &child.Priority, &childDue, &childTags, &childParent, &child.Position, &child.CreatedAt, &child.UpdatedAt)
					if err == nil {
						if childDue.Valid {
							ct := childDue.Time
							child.DueDate = &ct
						}
						if childParent.Valid {
							cpid := int(childParent.Int64)
							child.ParentID = &cpid
						}
						children = append(children, models.TodoResponse{
							Todo: child,
							Tags: helpers.ParseTagsFromJSON(childTags),
						})
					}
				}
				childRows.Close()
				if len(children) > 0 {
					resp.Subtasks = children
				}
			}
		}
		todos = append(todos, resp)
	}

	json.NewEncoder(w).Encode(todos)
}

func extractString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}

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

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	userID := extractInt(input, "user_id")
	title := extractString(input, "title")
	description := extractString(input, "description")
	domain := extractString(input, "domain")
	captureID := extractString(input, "capture_id")

	v := helpers.NewValidator()
	v.PositiveID("user_id", userID)
	v.Required("title", title)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	priority := "medium"
	if p, ok := input["priority"].(string); ok && p != "" {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if validPriorities[p] {
			priority = p
		}
	}

	tagsJSON := helpers.TagsToJSON(extractStringSlice(input, "tags"))

	var dueDateDB *time.Time
	if d, ok := input["due_date"].(string); ok && d != "" {
		parsed := parseDueDate(d)
		dueDateDB = parsed
	}

	var completed bool
	if c, ok := input["completed"].(bool); ok {
		completed = c
	}

	var parentID *int
	if p, ok := input["parent_id"].(float64); ok {
		pid := int(p)
		parentID = &pid
	}

	var position int
	if p, ok := input["position"].(float64); ok {
		position = int(p)
	} else {
		var parentWhere string
		var parentArgs []interface{}
		if parentID != nil {
			parentWhere = "WHERE parent_id = ? AND user_id = ?"
			parentArgs = []interface{}{*parentID, userID}
		} else {
			parentWhere = "WHERE parent_id IS NULL AND user_id = ?"
			parentArgs = []interface{}{userID}
		}
		var maxPos sql.NullInt64
		db.TodoDB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM todos "+parentWhere, parentArgs...).Scan(&maxPos)
		position = int(maxPos.Int64) + 1
	}

	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, domain, capture_id, completed, priority, due_date, tags, parent_id, position)
		VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, capture_id) DO NOTHING`,
		userID, title, description, domain, captureID, completed, priority, dueDateDB, tagsJSON, parentID, position)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 && captureID != "" {
		var todo models.Todo
		var tagsDB string
		var due sql.NullTime
		var pid sql.NullInt64
		err = db.TodoDB.QueryRow(`
			SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at
			FROM todos WHERE user_id = ? AND capture_id = ?`,
			userID, captureID,
		).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.Pinned, &todo.Archived, &todo.Priority, &due, &tagsDB, &pid, &todo.Position, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if due.Valid {
			dt := due.Time
			todo.DueDate = &dt
		}
		if pid.Valid {
			cpid := int(pid.Int64)
			todo.ParentID = &cpid
		}
		resp := models.TodoResponse{Todo: todo, Tags: helpers.ParseTagsFromJSON(tagsDB)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	todoID, _ := result.LastInsertId()
	now := time.Now()
	todo := models.Todo{
		ID:          int(todoID),
		UserID:      userID,
		Title:       title,
		Description: description,
		Domain:      domain,
		Completed:   completed,
		Priority:    priority,
		DueDate:     dueDateDB,
		ParentID:    parentID,
		Position:    position,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	resp := models.TodoResponse{Todo: todo, Tags: helpers.ParseTagsFromJSON(tagsJSON)}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func GetTodoByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var todo models.Todo
	var tagsDB string
	var dueDate sql.NullTime
	var parentID sql.NullInt64
	err := db.TodoDB.QueryRow("SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE id = ?", id).
		Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.Pinned, &todo.Archived, &todo.Priority, &dueDate, &tagsDB, &parentID, &todo.Position, &todo.CreatedAt, &todo.UpdatedAt)

	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dueDate.Valid {
		dt := dueDate.Time
		todo.DueDate = &dt
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		todo.ParentID = &pid
	}

	resp := models.TodoResponse{Todo: todo, Tags: helpers.ParseTagsFromJSON(tagsDB)}

	childRows, err := db.TodoDB.Query("SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE parent_id = ? ORDER BY pinned DESC, position ASC", todo.ID)
	if err == nil {
		var children []models.TodoResponse
		for childRows.Next() {
			var child models.Todo
			var childTags string
			var childDue sql.NullTime
			var childParent sql.NullInt64
			err := childRows.Scan(&child.ID, &child.UserID, &child.Title, &child.Description, &child.Domain, &child.ScreenshotPath, &child.Completed, &child.Pinned, &child.Archived, &child.Priority, &childDue, &childTags, &childParent, &child.Position, &child.CreatedAt, &child.UpdatedAt)
			if err == nil {
				if childDue.Valid {
					ct := childDue.Time
					child.DueDate = &ct
				}
				if childParent.Valid {
					cpid := int(childParent.Int64)
					child.ParentID = &cpid
				}
				children = append(children, models.TodoResponse{
					Todo: child,
					Tags: helpers.ParseTagsFromJSON(childTags),
				})
			}
		}
		childRows.Close()
		if len(children) > 0 {
			resp.Subtasks = children
		}
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
		Completed      *bool           `json:"completed"`
		Pinned         *bool           `json:"pinned"`
		Archived       *bool           `json:"archived"`
		Priority       *string         `json:"priority"`
		DueDate        json.RawMessage `json:"due_date"`
		Tags           *[]string       `json:"tags"`
		Position       *int            `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var todo models.Todo
	var tagsJSON string
	var dueDate sql.NullTime
	var parentID sql.NullInt64
	err := db.TodoDB.QueryRow("SELECT id, user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, due_date, tags, parent_id, position, created_at, updated_at FROM todos WHERE id = ?", id).
		Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.Pinned, &todo.Archived, &todo.Priority, &dueDate, &tagsJSON, &parentID, &todo.Position, &todo.CreatedAt, &todo.UpdatedAt)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if dueDate.Valid {
		dt := dueDate.Time
		todo.DueDate = &dt
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		todo.ParentID = &pid
	}

	if input.UserID != nil {
		todo.UserID = *input.UserID
	}
	if input.Title != nil {
		v := helpers.NewValidator()
		v.Required("title", *input.Title)
		if !v.OK() {
			helpers.WriteValidationError(w, v.Fields())
			return
		}
		todo.Title = *input.Title
	}
	if input.Description != nil {
		todo.Description = *input.Description
	}
	if input.Domain != nil {
		todo.Domain = *input.Domain
	}
	if input.ScreenshotPath != nil {
		todo.ScreenshotPath = *input.ScreenshotPath
	}
	if input.Completed != nil {
		todo.Completed = *input.Completed
	}
	if input.Pinned != nil {
		todo.Pinned = *input.Pinned
	}
	if input.Archived != nil {
		todo.Archived = *input.Archived
	}
	if input.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*input.Priority] {
			helpers.WriteValidationError(w, map[string]string{"priority": "must be low, medium, high, or urgent"})
			return
		}
		todo.Priority = *input.Priority
	}
	if input.DueDate != nil {
		todo.DueDate = nil
		if string(input.DueDate) != "null" {
			var rawDueDate string
			if err := json.Unmarshal(input.DueDate, &rawDueDate); err != nil {
				helpers.WriteValidationError(w, map[string]string{"due_date": "must be a date string or null"})
				return
			}
			if rawDueDate != "" {
				todo.DueDate = parseDueDate(rawDueDate)
				if todo.DueDate == nil {
					helpers.WriteValidationError(w, map[string]string{"due_date": "must be a valid date"})
					return
				}
			}
		}
	}
	if input.Tags != nil {
		tagsJSON = helpers.TagsToJSON(*input.Tags)
	}
	if input.Position != nil {
		todo.Position = *input.Position
	}

	_, err = db.TodoDB.Exec("UPDATE todos SET user_id = ?, title = ?, description = ?, domain = ?, screenshot_path = ?, completed = ?, pinned = ?, archived = ?, priority = ?, due_date = ?, tags = ?, position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		todo.UserID, todo.Title, todo.Description, todo.Domain, todo.ScreenshotPath, todo.Completed, todo.Pinned, todo.Archived, todo.Priority, todo.DueDate, tagsJSON, todo.Position, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	todo.UpdatedAt = time.Now()
	resp := models.TodoResponse{Todo: todo, Tags: helpers.ParseTagsFromJSON(tagsJSON)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	tx, err := db.TodoDB.Begin()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer tx.Rollback()

	// Delete subtasks first
	if _, err := tx.Exec("DELETE FROM todos WHERE parent_id = ?", id); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	result, err := tx.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}

	if err := tx.Commit(); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

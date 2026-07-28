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

// Column list used in all SELECT queries for todos.
const todoColumns = "id, user_id, title, description, domain, screenshot_path, pinned, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at"

func parseDate(raw string) *time.Time {
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

func scanTodo(scanner interface{ Scan(...any) error }) (models.Todo, string, error) {
	var todo models.Todo
	var tagsJSON string
	var startDate sql.NullTime
	var endDate sql.NullTime
	var parentID sql.NullInt64
	err := scanner.Scan(
		&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain,
		&todo.ScreenshotPath, &todo.Pinned, &todo.Status, &todo.Priority, &todo.Color,
		&startDate, &endDate, &todo.Rrule, &tagsJSON, &parentID,
		&todo.Position, &todo.CreatedAt, &todo.UpdatedAt,
	)
	if err != nil {
		return todo, tagsJSON, err
	}
	if startDate.Valid {
		t := startDate.Time
		todo.StartDate = &t
	}
	if endDate.Valid {
		t := endDate.Time
		todo.EndDate = &t
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		todo.ParentID = &pid
	}
	return todo, tagsJSON, nil
}

func todoToResponse(todo models.Todo, tagsJSON string) models.TodoResponse {
	return models.TodoResponse{
		Todo: todo,
		Tags: helpers.ParseTagsFromJSON(tagsJSON),
	}
}

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	statusFilter := r.URL.Query().Get("status")
	domain := r.URL.Query().Get("domain")
	priority := r.URL.Query().Get("priority")
	tagFilter := r.URL.Query().Get("tag")
	parentIDStr := r.URL.Query().Get("parent_id")
	sortField := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")

	// Default: exclude archived unless explicitly requested
	showArchived, _ := strconv.ParseBool(r.URL.Query().Get("archived"))

	query := "SELECT " + todoColumns + " FROM todos WHERE 1=1"
	args := []interface{}{}

	if !showArchived {
		query += " AND status != 'archived'"
	}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
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
		case "position", "start_date", "created_at":
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
		todo, tagsJSON, err := scanTodo(rows)
		if err != nil {
			continue
		}
		resp := todoToResponse(todo, tagsJSON)

		if tagFilter == "" {
			childRows, cerr := db.TodoDB.Query("SELECT "+todoColumns+" FROM todos WHERE parent_id = ? AND status != 'archived' ORDER BY pinned DESC, position ASC", todo.ID)
			if cerr == nil {
				var children []models.TodoResponse
				for childRows.Next() {
					child, childTags, err := scanTodo(childRows)
					if err == nil {
						children = append(children, todoToResponse(child, childTags))
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

	status := "pending"
	if s, ok := input["status"].(string); ok && s != "" {
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "archived": true}
		if validStatuses[s] {
			status = s
		}
	}

	color := extractString(input, "color")
	rrule := extractString(input, "rrule")
	tagsJSON := helpers.TagsToJSON(extractStringSlice(input, "tags"))

	var startDateDB *time.Time
	if d, ok := input["start_date"].(string); ok && d != "" {
		startDateDB = parseDate(d)
	}

	var endDateDB *time.Time
	if d, ok := input["end_date"].(string); ok && d != "" {
		endDateDB = parseDate(d)
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
		INSERT INTO todos (user_id, title, description, domain, capture_id, status, priority, color, start_date, end_date, rrule, tags, parent_id, position)
		VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, capture_id) DO NOTHING`,
		userID, title, description, domain, captureID, status, priority, color, startDateDB, endDateDB, rrule, tagsJSON, parentID, position)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 && captureID != "" {
		row := db.TodoDB.QueryRow("SELECT "+todoColumns+" FROM todos WHERE user_id = ? AND capture_id = ?", userID, captureID)
		todo, tagsDB, err := scanTodo(row)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
			return
		}
		resp := todoToResponse(todo, tagsDB)
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
		Status:      status,
		Priority:    priority,
		Color:       color,
		StartDate:   startDateDB,
		EndDate:     endDateDB,
		Rrule:       rrule,
		ParentID:    parentID,
		Position:    position,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	resp := todoToResponse(todo, tagsJSON)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func GetTodoByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	row := db.TodoDB.QueryRow("SELECT "+todoColumns+" FROM todos WHERE id = ?", id)
	todo, tagsDB, err := scanTodo(row)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	resp := todoToResponse(todo, tagsDB)

	childRows, err := db.TodoDB.Query("SELECT "+todoColumns+" FROM todos WHERE parent_id = ? ORDER BY pinned DESC, position ASC", todo.ID)
	if err == nil {
		var children []models.TodoResponse
		for childRows.Next() {
			child, childTags, err := scanTodo(childRows)
			if err == nil {
				children = append(children, todoToResponse(child, childTags))
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
		Pinned         *bool           `json:"pinned"`
		Status         *string         `json:"status"`
		Priority       *string         `json:"priority"`
		Color          *string         `json:"color"`
		StartDate      json.RawMessage `json:"start_date"`
		EndDate        json.RawMessage `json:"end_date"`
		Rrule          *string         `json:"rrule"`
		Tags           *[]string       `json:"tags"`
		Position       *int            `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	row := db.TodoDB.QueryRow("SELECT "+todoColumns+" FROM todos WHERE id = ?", id)
	todo, tagsJSON, err := scanTodo(row)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
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
	if input.Pinned != nil {
		todo.Pinned = *input.Pinned
	}
	if input.Status != nil {
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "archived": true}
		if !validStatuses[*input.Status] {
			helpers.WriteValidationError(w, map[string]string{"status": "must be pending, in_progress, completed, or archived"})
			return
		}
		todo.Status = *input.Status
	}
	if input.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*input.Priority] {
			helpers.WriteValidationError(w, map[string]string{"priority": "must be low, medium, high, or urgent"})
			return
		}
		todo.Priority = *input.Priority
	}
	if input.Color != nil {
		todo.Color = *input.Color
	}
	if input.Rrule != nil {
		todo.Rrule = *input.Rrule
	}
	if input.StartDate != nil {
		todo.StartDate = nil
		if string(input.StartDate) != "null" {
			var raw string
			if err := json.Unmarshal(input.StartDate, &raw); err != nil {
				helpers.WriteValidationError(w, map[string]string{"start_date": "must be a date string or null"})
				return
			}
			if raw != "" {
				todo.StartDate = parseDate(raw)
				if todo.StartDate == nil {
					helpers.WriteValidationError(w, map[string]string{"start_date": "must be a valid date"})
					return
				}
			}
		}
	}
	if input.EndDate != nil {
		todo.EndDate = nil
		if string(input.EndDate) != "null" {
			var raw string
			if err := json.Unmarshal(input.EndDate, &raw); err != nil {
				helpers.WriteValidationError(w, map[string]string{"end_date": "must be a date string or null"})
				return
			}
			if raw != "" {
				todo.EndDate = parseDate(raw)
				if todo.EndDate == nil {
					helpers.WriteValidationError(w, map[string]string{"end_date": "must be a valid date"})
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

	_, err = db.TodoDB.Exec("UPDATE todos SET user_id = ?, title = ?, description = ?, domain = ?, screenshot_path = ?, pinned = ?, status = ?, priority = ?, color = ?, start_date = ?, end_date = ?, rrule = ?, tags = ?, position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		todo.UserID, todo.Title, todo.Description, todo.Domain, todo.ScreenshotPath, todo.Pinned, todo.Status, todo.Priority, todo.Color, todo.StartDate, todo.EndDate, todo.Rrule, tagsJSON, todo.Position, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	todo.UpdatedAt = time.Now()
	resp := todoToResponse(todo, tagsJSON)
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

package tools

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

//go:embed schemas/update_todo_record.json
var updateTodoRecordSchema []byte

func registerUpdateTodoRecord(r *Registry) {
	r.add(Tool{
		Name:        "update_todo_record",
		Category:    "General",
		Description: "Update an existing todo item (including sub-tasks) by ID. Requires user_id and id. All other fields are optional - only provided fields are updated. Use null for parent_id, start_date, or end_date to clear them.",
		Schema:      json.RawMessage(updateTodoRecordSchema),
		Execute:     updateTodoRecord,
	})
}

func updateTodoRecord(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID         int             `json:"user_id"`
		ID             int             `json:"id"`
		Title          *string         `json:"title"`
		Description    *string         `json:"description"`
		Status         *string         `json:"status"`
		Priority       *string         `json:"priority"`
		Color          *string         `json:"color"`
		Tags           *[]string       `json:"tags"`
		Position       *int            `json:"position"`
		ParentID       json.RawMessage `json:"parent_id"`
		StartDate      json.RawMessage `json:"start_date"`
		EndDate        json.RawMessage `json:"end_date"`
		Rrule          *string         `json:"rrule"`
		Domain         *string         `json:"domain"`
		ScreenshotPath *string         `json:"screenshot_path"`
		Pinned         *bool           `json:"pinned"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "id": true,
		"title": true, "description": true,
		"status": true, "priority": true,
		"color": true, "tags": true,
		"position": true, "parent_id": true,
		"start_date": true, "end_date": true,
		"rrule": true, "domain": true,
		"screenshot_path": true, "pinned": true,
	}); err != nil {
		return nil, err
	}

	// Validate required fields
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}
	if a.ID < 1 {
		return nil, fmt.Errorf("id is required and must be a positive integer")
	}

	// Validate provided field values
	if a.Title != nil {
		title := strings.TrimSpace(*a.Title)
		if title == "" {
			return nil, fmt.Errorf("title must not be empty")
		}
		if len(title) > 500 {
			return nil, fmt.Errorf("title must be 500 characters or fewer")
		}
		*a.Title = title
	}
	if a.Description != nil {
		if len(*a.Description) > 2000 {
			return nil, fmt.Errorf("description must be 2000 characters or fewer")
		}
		*a.Description = strings.TrimSpace(*a.Description)
	}
	if a.Tags != nil {
		for i, tag := range *a.Tags {
			if len(tag) > 100 {
				return nil, fmt.Errorf("tags[%d] must be 100 characters or fewer", i)
			}
		}
	}
	if a.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*a.Priority] {
			return nil, fmt.Errorf("priority must be one of: low, medium, high, urgent")
		}
	}
	if a.Status != nil {
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "done": true, "cancelled": true, "archived": true}
		if !validStatuses[*a.Status] {
			return nil, fmt.Errorf("status must be one of: pending, in_progress, completed, done, cancelled, archived")
		}
	}
	if a.Color != nil {
		*a.Color = strings.TrimSpace(*a.Color)
		if *a.Color != "" && !isValidColor(*a.Color) {
			return nil, fmt.Errorf("color must be empty or a valid hex color code (e.g., #FF5733 or #33FF57)")
		}
	}
	if a.Position != nil && *a.Position < 0 {
		return nil, fmt.Errorf("position must be a non-negative integer")
	}

	// Parse parent_id without accessing the database. Ownership and relationship
	// checks happen after all input validation has completed.
	var parentID *int
	if a.ParentID != nil {
		if string(a.ParentID) == "null" {
			parentID = nil // explicit clear
		} else {
			var pid int
			if err := json.Unmarshal(a.ParentID, &pid); err != nil || pid < 1 {
				return nil, fmt.Errorf("parent_id must be a positive integer or null")
			}
			parentID = &pid
		}
	}

	// Parse dates (null clears the field)
	var startDate *time.Time
	if a.StartDate != nil && string(a.StartDate) != "null" {
		var raw string
		if err := json.Unmarshal(a.StartDate, &raw); err != nil {
			return nil, fmt.Errorf("start_date must be a date string or null")
		}
		if raw != "" {
			parsed := parseDate(raw)
			if parsed == nil {
				return nil, fmt.Errorf("start_date must be a valid date (YYYY-MM-DD or RFC3339)")
			}
			startDate = parsed
		}
	}
	var endDate *time.Time
	if a.EndDate != nil && string(a.EndDate) != "null" {
		var raw string
		if err := json.Unmarshal(a.EndDate, &raw); err != nil {
			return nil, fmt.Errorf("end_date must be a date string or null")
		}
		if raw != "" {
			parsed := parseDate(raw)
			if parsed == nil {
				return nil, fmt.Errorf("end_date must be a valid date (YYYY-MM-DD or RFC3339)")
			}
			endDate = parsed
		}
	}

	// Convert tags to JSON
	var tagsJSON string
	if a.Tags != nil {
		// Store empty tags as a JSON array for consistent response types.
		if len(*a.Tags) == 0 {
			tagsJSON = "[]"
		} else {
			tagsJSON = helpers.TagsToJSON(*a.Tags)
		}
	}

	// Check if any updatable field was provided
	hasUpdate := a.Title != nil || a.Description != nil || a.Status != nil ||
		a.Priority != nil || a.Color != nil || a.Tags != nil || a.Position != nil ||
		a.ParentID != nil || a.StartDate != nil || a.EndDate != nil || a.Rrule != nil ||
		a.Domain != nil || a.ScreenshotPath != nil || a.Pinned != nil
	if !hasUpdate {
		return nil, fmt.Errorf("no updatable fields provided")
	}

	// Verify ownership only after input validation, so malformed requests do not
	// depend on database availability and consistently return validation errors.
	var existingUserID int
	err := db.TodoDB.QueryRow(
		"SELECT user_id FROM todos WHERE id = ?",
		a.ID,
	).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo %d not found", a.ID)
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != a.UserID {
		return nil, fmt.Errorf("todo %d does not belong to user %d", a.ID, a.UserID)
	}

	if parentID != nil {
		if *parentID == a.ID {
			return nil, fmt.Errorf("parent_id must not equal the todo's own id")
		}
		// Verify the parent belongs to the same user and is not itself a subtask.
		var parentUserID int
		var grandparentID sql.NullInt64
		perr := db.TodoDB.QueryRow(
			"SELECT user_id, parent_id FROM todos WHERE id = ?",
			*parentID,
		).Scan(&parentUserID, &grandparentID)
		if perr == sql.ErrNoRows {
			return nil, fmt.Errorf("parent todo %d not found", *parentID)
		}
		if perr != nil {
			return nil, perr
		}
		if parentUserID != a.UserID {
			return nil, fmt.Errorf("parent todo %d does not belong to user %d", *parentID, a.UserID)
		}
		if grandparentID.Valid {
			return nil, fmt.Errorf("parent todo %d is itself a subtask; nested subtasks are not allowed", *parentID)
		}
	}

	// Build dynamic UPDATE SET clause
	setClauses := []string{}
	args := []any{}

	if a.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *a.Title)
	}
	if a.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *a.Description)
	}
	if a.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, *a.Status)
	}
	if a.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *a.Priority)
	}
	if a.Color != nil {
		setClauses = append(setClauses, "color = ?")
		args = append(args, nullIfEmpty(*a.Color))
	}
	if a.Tags != nil {
		setClauses = append(setClauses, "tags = ?")
		args = append(args, tagsJSON)
	}
	if a.Position != nil {
		setClauses = append(setClauses, "position = ?")
		args = append(args, *a.Position)
	}
	if a.ParentID != nil {
		setClauses = append(setClauses, "parent_id = ?")
		args = append(args, parentID)
	}
	if a.StartDate != nil {
		setClauses = append(setClauses, "start_date = ?")
		if startDate != nil {
			args = append(args, startDate)
		} else {
			args = append(args, nil)
		}
	}
	if a.EndDate != nil {
		setClauses = append(setClauses, "end_date = ?")
		if endDate != nil {
			args = append(args, endDate)
		} else {
			args = append(args, nil)
		}
	}
	if a.Rrule != nil {
		setClauses = append(setClauses, "rrule = ?")
		args = append(args, *a.Rrule)
	}
	if a.Domain != nil {
		setClauses = append(setClauses, "domain = ?")
		args = append(args, *a.Domain)
	}
	if a.ScreenshotPath != nil {
		setClauses = append(setClauses, "screenshot_path = ?")
		args = append(args, *a.ScreenshotPath)
	}
	if a.Pinned != nil {
		setClauses = append(setClauses, "pinned = ?")
		args = append(args, *a.Pinned)
	}

	setClause := strings.Join(setClauses, ", ")
	args = append(args, a.ID)

	_, err = db.TodoDB.Exec(
		"UPDATE todos SET "+setClause+", updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		args...,
	)
	if err != nil {
		return nil, err
	}

	// Fetch the updated todo
	row := db.TodoDB.QueryRow(
		"SELECT id, user_id, title, description, domain, screenshot_path, pinned, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at FROM todos WHERE id = ?",
		a.ID,
	)
	todo, tagsJSONOut, err := scanTodo(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo %d not found after update", a.ID)
	}
	if err != nil {
		return nil, err
	}

	// Fetch subtasks
	childRows, err := db.TodoDB.Query(
		"SELECT id, user_id, title, description, domain, screenshot_path, pinned, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at FROM todos WHERE parent_id = ? ORDER BY position ASC",
		a.ID,
	)
	if err == nil {
		var subtasks []any
		for childRows.Next() {
			childTodo, childTagsJSON, cerr := scanTodo(childRows)
			if cerr != nil {
				continue
			}
			subtask := map[string]any{
				"id":              childTodo.ID,
				"user_id":         childTodo.UserID,
				"title":           childTodo.Title,
				"description":     childTodo.Description,
				"domain":          nullIfEmpty(childTodo.Domain),
				"screenshot_path": nullIfEmpty(childTodo.ScreenshotPath),
				"pinned":          childTodo.Pinned,
				"status":          childTodo.Status,
				"priority":        childTodo.Priority,
				"color":           nullIfEmpty(childTodo.Color),
				"start_date":      nil,
				"end_date":        nil,
				"rrule":           nullIfEmpty(childTodo.Rrule),
				"tags":            emptyIfEmptySlice(helpers.ParseTagsFromJSON(childTagsJSON)),
				"parent_id":       childTodo.ParentID,
				"position":        childTodo.Position,
				"created_at":      childTodo.CreatedAt,
				"updated_at":      childTodo.UpdatedAt,
			}
			if childTodo.StartDate != nil {
				subtask["start_date"] = childTodo.StartDate.Format("2006-01-02")
			}
			if childTodo.EndDate != nil {
				subtask["end_date"] = childTodo.EndDate.Format("2006-01-02")
			}
			subtasks = append(subtasks, subtask)
		}
		childRows.Close()
		if len(subtasks) > 0 {
			result := todoToMap(todo, tagsJSONOut)
			result["subtasks"] = subtasks
			return result, nil
		}
	}

	return todoToMap(todo, tagsJSONOut), nil
}

// scanTodo scans a database row into a models.Todo and tags JSON string.
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

// todoToMap converts a models.Todo and tags JSON string into a map for AI tool responses.
func todoToMap(todo models.Todo, tagsJSON string) map[string]any {
	m := map[string]any{
		"id":              todo.ID,
		"user_id":         todo.UserID,
		"title":           todo.Title,
		"description":     todo.Description,
		"domain":          nullIfEmpty(todo.Domain),
		"screenshot_path": nullIfEmpty(todo.ScreenshotPath),
		"pinned":          todo.Pinned,
		"status":          todo.Status,
		"priority":        todo.Priority,
		"color":           nullIfEmpty(todo.Color),
		"rrule":           nullIfEmpty(todo.Rrule),
		"tags":            emptyIfEmptySlice(helpers.ParseTagsFromJSON(tagsJSON)),
		"parent_id":       todo.ParentID,
		"position":        todo.Position,
		"created_at":      todo.CreatedAt,
		"updated_at":      todo.UpdatedAt,
	}
	if todo.StartDate != nil {
		m["start_date"] = todo.StartDate.Format("2006-01-02")
	}
	if todo.EndDate != nil {
		m["end_date"] = todo.EndDate.Format("2006-01-02")
	}
	return m
}

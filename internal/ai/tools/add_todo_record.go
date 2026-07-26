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
)

//go:embed schemas/add_todo_record.json
var addTodoRecordSchema []byte

func registerAddTodoRecord(r *Registry) {
	r.add(Tool{
		Name:        "add_todo_record",
		Category:    "General",
		Description: "Create a todo item with optional sub-tasks in a single call. Requires user_id and title. Optional fields: description, parent_id, priority, status, color, tags, subtasks. Subtask titles default to 'Subtask N' if omitted.",
		Schema:      json.RawMessage(addTodoRecordSchema),
		Execute:     addTodoRecord,
	})
}

// subtaskInput represents a single subtask in the input array.
type subtaskInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Color       string   `json:"color"`
	Tags        []string `json:"tags"`
}

func addTodoRecord(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID      int            `json:"user_id"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		ParentID    int            `json:"parent_id"`
		Priority    string         `json:"priority"`
		Status      string         `json:"status"`
		Color       string         `json:"color"`
		Tags        []string       `json:"tags"`
		Subtasks    []subtaskInput `json:"subtasks"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "title": true, "description": true,
		"parent_id": true, "priority": true, "status": true,
		"color": true, "tags": true, "subtasks": true,
	}); err != nil {
		return nil, err
	}

	// Validate required fields
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}
	a.Title = strings.TrimSpace(a.Title)
	if a.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(a.Title) > 500 {
		return nil, fmt.Errorf("title must be 500 characters or fewer")
	}
	if len(a.Description) > 2000 {
		return nil, fmt.Errorf("description must be 2000 characters or fewer")
	}

	// Validate priority
	priority := "medium"
	if a.Priority != "" {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[a.Priority] {
			return nil, fmt.Errorf("priority must be one of: low, medium, high, urgent")
		}
		priority = a.Priority
	}

	// Validate status
	status := "pending"
	if a.Status != "" {
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "done": true, "cancelled": true, "archived": true}
		if !validStatuses[a.Status] {
			return nil, fmt.Errorf("status must be one of: pending, in_progress, completed, done, cancelled, archived")
		}
		status = a.Status
	}

	// Validate color
	color := strings.TrimSpace(a.Color)

	// Validate subtasks
	if len(a.Subtasks) > 20 {
		return nil, fmt.Errorf("subtasks must have 20 items or fewer")
	}
	for i, st := range a.Subtasks {
		st.Title = strings.TrimSpace(st.Title)
		if len(st.Title) > 500 {
			return nil, fmt.Errorf("subtasks[%d].title must be 500 characters or fewer", i)
		}
		if st.Title == "" {
			st.Title = fmt.Sprintf("Subtask %d", i+1)
		}
		if len(st.Description) > 2000 {
			return nil, fmt.Errorf("subtasks[%d].description must be 2000 characters or fewer", i)
		}
		if st.Priority != "" {
			validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
			if !validPriorities[st.Priority] {
				return nil, fmt.Errorf("subtasks[%d].priority must be one of: low, medium, high, urgent", i)
			}
		}
		if st.Status != "" {
			validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "done": true, "cancelled": true, "archived": true}
			if !validStatuses[st.Status] {
				return nil, fmt.Errorf("subtasks[%d].status must be one of: pending, in_progress, completed, done, cancelled, archived", i)
			}
		}
		a.Subtasks[i] = st
	}

	// Validate parent_id if provided
	var parentID *int
	if a.ParentID > 0 {
		// Verify parent exists and belongs to the same user
		var parentUserID int
		err := db.TodoDB.QueryRow(
			"SELECT user_id FROM todos WHERE id = ?",
			a.ParentID,
		).Scan(&parentUserID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("parent todo %d not found", a.ParentID)
		}
		if err != nil {
			return nil, err
		}
		if parentUserID != a.UserID {
			return nil, fmt.Errorf("parent todo %d does not belong to user %d", a.ParentID, a.UserID)
		}
		parentID = &a.ParentID
	}

	// Convert tags to JSON
	tagsJSON := helpers.TagsToJSON(a.Tags)

	// Determine position (scoped to parent_id, matching CreateSubtask pattern)
	var whereClause string
	var posArgs []any
	if parentID != nil {
		whereClause = "WHERE parent_id = ? AND user_id = ?"
		posArgs = []any{*parentID, a.UserID}
	} else {
		whereClause = "WHERE parent_id IS NULL AND user_id = ?"
		posArgs = []any{a.UserID}
	}

	var maxPos sql.NullInt64
	db.TodoDB.QueryRow(
		"SELECT COALESCE(MAX(position), -1) FROM todos "+whereClause,
		posArgs...,
	).Scan(&maxPos)
	position := int(maxPos.Int64) + 1

	now := time.Now()
	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, status, priority, color, tags, parent_id, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		a.UserID, a.Title, a.Description, status, priority, color, tagsJSON, parentID, position)
	if err != nil {
		return nil, err
	}

	todoID, _ := result.LastInsertId()
	todo := map[string]any{
		"id":          todoID,
		"user_id":     a.UserID,
		"title":       a.Title,
		"description": a.Description,
		"status":      status,
		"priority":    priority,
		"color":       color,
		"tags":        a.Tags,
		"parent_id":   a.ParentID,
		"position":    position,
		"created_at":  now,
		"updated_at":  now,
	}

	// Create subtasks if provided
	if len(a.Subtasks) > 0 {
		subtasks := make([]map[string]any, 0, len(a.Subtasks))
		for i, st := range a.Subtasks {
			stPriority := "medium"
			if st.Priority != "" {
				stPriority = st.Priority
			}
			stStatus := "pending"
			if st.Status != "" {
				stStatus = st.Status
			}
			stColor := strings.TrimSpace(st.Color)
			stTagsJSON := helpers.TagsToJSON(st.Tags)

			stResult, err := db.TodoDB.Exec(`
				INSERT INTO todos (user_id, title, description, status, priority, color, tags, parent_id, position, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
				a.UserID, st.Title, st.Description, stStatus, stPriority, stColor, stTagsJSON, todoID, i)
			if err != nil {
				return nil, fmt.Errorf("failed to create subtask %d: %w", i, err)
			}

			stID, _ := stResult.LastInsertId()
			subtasks = append(subtasks, map[string]any{
				"id":          stID,
				"user_id":     a.UserID,
				"title":       st.Title,
				"description": st.Description,
				"status":      stStatus,
				"priority":    stPriority,
				"color":       stColor,
				"tags":        st.Tags,
				"parent_id":   todoID,
				"position":    i,
				"created_at":  now,
				"updated_at":  now,
			})
		}
		todo["subtasks"] = subtasks
	}

	return todo, nil
}

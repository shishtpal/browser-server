package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"browser-server/internal/db"
	"browser-server/internal/todo"
)

//go:embed schemas/add_todo_record.json
var addTodoRecordSchema []byte

func registerAddTodoRecord(r *Registry) {
	r.add(Tool{
		Name:        "add_todo_record",
		Category:    "General",
		Description: "Create a todo item with optional sub-tasks in a single call. Requires user_id and title. Optional fields: description, parent_id, priority, status, color, tags, start_date, end_date, and subtasks. Subtask titles default to 'Subtask N' if omitted.",
		Schema:      json.RawMessage(addTodoRecordSchema),
		Execute:     addTodoRecord,
	})
}

func addTodoRecord(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID      int                 `json:"user_id"`
		Title       string              `json:"title"`
		Description string              `json:"description"`
		Domain      string              `json:"domain"`
		ParentID    int                 `json:"parent_id"`
		Priority    string              `json:"priority"`
		Status      string              `json:"status"`
		Color       string              `json:"color"`
		Tags        []string            `json:"tags"`
		Subtasks    []todo.SubtaskInput `json:"subtasks"`
		StartDate   *string             `json:"start_date"`
		EndDate     *string             `json:"end_date"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "title": true, "description": true, "domain": true,
		"parent_id": true, "priority": true, "status": true,
		"color": true, "tags": true, "subtasks": true,
		"start_date": true, "end_date": true,
	}); err != nil {
		return nil, err
	}
	if a.Subtasks == nil {
		a.Subtasks = []todo.SubtaskInput{}
	}

	// Append the chat-origin tag so every todo created via this AI tool is
	// easy to filter/search/audit. Dedupe first: if the model already
	// supplied the tag, don't add a duplicate.
	const chatOriginTag = "browser-server-chat"
	hasTag := false
	for _, t := range a.Tags {
		if t == chatOriginTag {
			hasTag = true
			break
		}
	}
	if !hasTag {
		a.Tags = append(a.Tags, chatOriginTag)
	}

	createInput := todo.CreateInput{
		UserID:      a.UserID,
		Title:       a.Title,
		Description: a.Description,
		Domain:      a.Domain,
		ParentID:    a.ParentID,
		Priority:    a.Priority,
		Status:      a.Status,
		Color:       a.Color,
		Tags:        a.Tags,
		Subtasks:    a.Subtasks,
		StartDate:   a.StartDate,
		EndDate:     a.EndDate,
	}

	repo := todo.NewRepository(db.TodoDB)
	result, err := repo.Create(&createInput)
	if err != nil {
		if validationErr, ok := err.(*todo.ValidationError); ok {
			return nil, fmt.Errorf("validation failed: %s", validationErr.Error())
		}
		return nil, err
	}

	return result, nil
}

package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"browser-server/internal/db"
	"browser-server/internal/todo"
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

	updateInput := todo.UpdateInput{UserID: a.UserID, ID: a.ID}
	updateInput.Title = a.Title
	updateInput.Description = a.Description
	updateInput.Status = a.Status
	updateInput.Priority = a.Priority
	updateInput.Color = a.Color
	updateInput.Tags = a.Tags
	updateInput.Position = a.Position
	updateInput.Rrule = a.Rrule
	updateInput.Domain = a.Domain
	updateInput.ScreenshotPath = a.ScreenshotPath
	updateInput.Pinned = a.Pinned

	if a.ParentID != nil {
		if string(a.ParentID) == "null" {
			zero := 0
			updateInput.ParentID = &zero
		} else {
			var parentID int
			if err := json.Unmarshal(a.ParentID, &parentID); err != nil {
				return nil, fmt.Errorf("parent_id must be a positive integer or null")
			}
			updateInput.ParentID = &parentID
		}
	}
	if a.StartDate != nil {
		if string(a.StartDate) == "null" {
			clear := ""
			updateInput.StartDate = &clear
		} else {
			var raw string
			if err := json.Unmarshal(a.StartDate, &raw); err != nil {
				return nil, fmt.Errorf("start_date must be a date string or null")
			}
			updateInput.StartDate = &raw
		}
	}
	if a.EndDate != nil {
		if string(a.EndDate) == "null" {
			clear := ""
			updateInput.EndDate = &clear
		} else {
			var raw string
			if err := json.Unmarshal(a.EndDate, &raw); err != nil {
				return nil, fmt.Errorf("end_date must be a date string or null")
			}
			updateInput.EndDate = &raw
		}
	}

	repo := todo.NewRepository(db.TodoDB)
	// Validate before DB access so validation-only tests don't panic.
	if err := todo.ValidateUpdateInput(&updateInput); err != nil {
		return nil, fmt.Errorf("validation failed: %s", err.Error())
	}
	result, err := repo.Update(&updateInput)
	if err != nil {
		if validationErr, ok := err.(*todo.ValidationError); ok {
			return nil, fmt.Errorf("validation failed: %s", validationErr.Error())
		}
		return nil, err
	}

	return result, nil
}

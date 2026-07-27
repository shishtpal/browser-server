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

//go:embed schemas/add_calendar_event.json
var addCalendarEventSchema []byte

func registerAddCalendarEvent(r *Registry) {
	r.add(Tool{
		Name:        "add_calendar_event",
		Category:    "General",
		Description: "Create a calendar event (todo with a scheduled date). Requires user_id, title, and start_date. Optional fields: description, end_date, rrule, priority, status, color, tags.",
		Schema:      json.RawMessage(addCalendarEventSchema),
		Execute:     addCalendarEvent,
	})
}

func addCalendarEvent(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID      int      `json:"user_id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		StartDate   string   `json:"start_date"`
		EndDate     string   `json:"end_date"`
		Rrule       string   `json:"rrule"`
		Priority    string   `json:"priority"`
		Status      string   `json:"status"`
		Color       string   `json:"color"`
		Tags        []string `json:"tags"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "title": true, "description": true,
		"start_date": true, "end_date": true, "rrule": true,
		"priority": true, "status": true, "color": true, "tags": true,
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

	// Parse start_date (required)
	startDate := parseDate(a.StartDate)
	if startDate == nil {
		return nil, fmt.Errorf("start_date is required and must be a valid date (YYYY-MM-DD or RFC3339)")
	}

	// Parse end_date (optional)
	var endDate *time.Time
	if a.EndDate != "" {
		endDate = parseDate(a.EndDate)
		if endDate == nil {
			return nil, fmt.Errorf("end_date must be a valid date (YYYY-MM-DD or RFC3339)")
		}
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
		validStatuses := map[string]bool{"pending": true, "completed": true, "archived": true}
		if !validStatuses[a.Status] {
			return nil, fmt.Errorf("status must be one of: pending, completed, archived")
		}
		status = a.Status
	}

	// Validate color
	color := strings.TrimSpace(a.Color)

	// Validate rrule
	rrule := strings.TrimSpace(a.Rrule)

	// Convert tags to JSON
	tagsJSON := helpers.TagsToJSON(a.Tags)

	// Determine position (same logic as CreateTodo handler)
	var maxPos sql.NullInt64
	db.TodoDB.QueryRow(
		"SELECT COALESCE(MAX(position), -1) FROM todos WHERE parent_id IS NULL AND user_id = ?",
		a.UserID,
	).Scan(&maxPos)
	position := int(maxPos.Int64) + 1

	now := time.Now()
	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		a.UserID, a.Title, a.Description, status, priority, color, startDate, endDate, rrule, tagsJSON, position)
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
		"start_date":  startDate.Format("2006-01-02"),
		"rrule":       rrule,
		"tags":        a.Tags,
		"position":    position,
		"created_at":  now,
		"updated_at":  now,
	}
	if endDate != nil {
		todo["end_date"] = endDate.Format("2006-01-02")
	}

	return todo, nil
}

// parseDate parses a date string in YYYY-MM-DD or RFC3339 format.
// Returns nil for empty or unparsable strings.
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

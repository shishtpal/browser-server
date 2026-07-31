package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/todo"
)

//go:embed schemas/manage_calendar.json
var manageCalendarSchema []byte

func registerManageCalendar(r *Registry) {
	r.add(Tool{
		Name:        "manage_calendar",
		Category:    "General",
		Description: "Manage calendar events (todos with scheduled dates). Supports add, edit, remove, and get actions. Requires user_id and action.",
		Schema:      json.RawMessage(manageCalendarSchema),
		Execute:     manageCalendar,
	})
}

func manageCalendar(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID      int             `json:"user_id"`
		Action      string          `json:"action"`
		ID          json.RawMessage `json:"id"`
		Title       *string         `json:"title"`
		Description *string         `json:"description"`
		StartDate   *string         `json:"start_date"`
		EndDate     *string         `json:"end_date"`
		Rrule       *string         `json:"rrule"`
		Priority    *string         `json:"priority"`
		Status      *string         `json:"status"`
		Color       *string         `json:"color"`
		Domain      *string         `json:"domain"`
		Tags        *[]string       `json:"tags"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "action": true,
		"id": true, "title": true, "description": true,
		"start_date": true, "end_date": true, "rrule": true,
		"priority": true, "status": true, "color": true, "domain": true, "tags": true,
	}); err != nil {
		return nil, err
	}

	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}

	switch a.Action {
	case "add", "edit", "remove", "get":
	default:
		return nil, fmt.Errorf("action must be one of: add, edit, remove, get")
	}

	if a.Action == "add" {
		if a.Title == nil {
			return nil, fmt.Errorf("title is required for add action")
		}
		if a.StartDate == nil {
			return nil, fmt.Errorf("start_date is required for add action")
		}
	}

	if a.Action == "edit" {
		hasField := a.Title != nil || a.Description != nil || a.StartDate != nil || a.EndDate != nil ||
			a.Rrule != nil || a.Priority != nil || a.Status != nil || a.Color != nil || a.Tags != nil
		if !hasField {
			return nil, fmt.Errorf("no updatable fields provided")
		}
	}

	if a.Priority != nil && !todo.IsValidPriority(*a.Priority) {
		return nil, fmt.Errorf("priority must be one of: low, medium, high, urgent")
	}
	if a.Status != nil && !todo.IsValidStatus(*a.Status) {
		return nil, fmt.Errorf("status must be one of: pending, in_progress, completed, done, cancelled, archived")
	}

	repo := todo.NewRepository(db.TodoDB)
	switch a.Action {
	case "add":
		return manageCalendarAdd(ctx, repo, a.UserID, a.Title, a.Description, a.StartDate, a.EndDate, a.Rrule, a.Priority, a.Status, a.Color, a.Domain, a.Tags)
	case "edit":
		return manageCalendarEdit(ctx, repo, a.UserID, a.ID, a.Title, a.Description, a.StartDate, a.EndDate, a.Rrule, a.Priority, a.Status, a.Color, a.Domain, a.Tags)
	case "remove":
		return manageCalendarRemove(ctx, repo, a.UserID, a.ID)
	case "get":
		return manageCalendarGet(ctx, repo, a.UserID, a.ID)
	default:
		return nil, fmt.Errorf("unknown action: %s", a.Action)
	}
}

func manageCalendarAdd(ctx context.Context, repo *todo.SQLRepository, userID int, title, description, startDate, endDate, rrule, priority, status, color, domain *string, tags *[]string) (any, error) {
	input := todo.CreateInput{
		UserID:      userID,
		Title:       strings.TrimSpace(*title),
		Description: coalesceString(description),
		StartDate:   startDate,
		EndDate:     endDate,
		Rrule:       coalesceString(rrule),
		Priority:    coalesceString(priority),
		Status:      coalesceString(status),
		Domain:      coalesceString(domain),
	}
	if color != nil && *color != "" {
		input.Color = strings.TrimSpace(*color)
	} else {
		input.Color = "#3366FF"
	}
	if tags != nil {
		input.Tags = *tags
	} else {
		input.Tags = []string{}
	}

	result, err := repo.Create(&input)
	if err != nil {
		return nil, err
	}

	colorValue := ""
	if result.Color != nil {
		colorValue = *result.Color
	}
	out := map[string]any{
		"id":          result.ID,
		"user_id":     result.UserID,
		"title":       result.Title,
		"description": result.Description,
		"status":      result.Status,
		"priority":    result.Priority,
		"color":       colorValue,
		"tags":        result.Tags,
		"position":    result.Position,
		"created_at":  result.CreatedAt,
		"updated_at":  result.UpdatedAt,
	}
	if result.StartDate != nil {
		out["start_date"] = *result.StartDate
	}
	if result.EndDate != nil {
		out["end_date"] = *result.EndDate
	}
	if result.ParentID != nil {
		out["parent_id"] = *result.ParentID
	}
	out["rrule"] = input.Rrule
	return out, nil
}

func manageCalendarEdit(ctx context.Context, repo *todo.SQLRepository, userID int, idRaw json.RawMessage, title, description, startDate, endDate, rrule, priority, status, color, domain *string, tags *[]string) (any, error) {
	id, err := calendarIDFromRaw(idRaw, "edit")
	if err != nil {
		return nil, err
	}

	rec, err := repo.GetByID(id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("calendar event %d not found", id)
	}
	if rec.UserID != userID {
		return nil, fmt.Errorf("calendar event %d does not belong to user %d", id, userID)
	}

	input := todo.UpdateInput{UserID: userID, ID: id}
	input.Title = title
	input.Description = description
	input.StartDate = startDate
	input.EndDate = endDate
	input.Rrule = rrule
	input.Priority = priority
	input.Status = status
	input.Color = color
	input.Domain = domain
	input.Tags = tags

	result, err := repo.Update(&input)
	if err != nil {
		if verr, ok := err.(*todo.ValidationError); ok {
			return nil, fmt.Errorf("validation failed: %s", verr.Error())
		}
		return nil, err
	}

	return result, nil
}

func manageCalendarRemove(ctx context.Context, repo *todo.SQLRepository, userID int, idRaw json.RawMessage) (any, error) {
	id, err := calendarIDFromRaw(idRaw, "remove")
	if err != nil {
		return nil, err
	}

	rec, err := repo.GetByID(id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("calendar event %d not found", id)
	}
	if rec.UserID != userID {
		return nil, fmt.Errorf("calendar event %d does not belong to user %d", id, userID)
	}

	if err := repo.Delete(id); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":      id,
		"user_id": userID,
		"removed": true,
	}, nil
}

func manageCalendarGet(ctx context.Context, repo *todo.SQLRepository, userID int, idRaw json.RawMessage) (any, error) {
	id, err := calendarIDFromRaw(idRaw, "get")
	if err != nil {
		return nil, err
	}

	rec, err := repo.GetByID(id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("calendar event %d not found", id)
	}
	if rec.UserID != userID {
		return nil, fmt.Errorf("calendar event %d does not belong to user %d", id, userID)
	}

	tags := rec.Tags
	if tags == nil {
		tags = []string{}
	}

	out := map[string]any{
		"id":          rec.ID,
		"user_id":     rec.UserID,
		"title":       rec.Title,
		"description": rec.Description,
		"status":      rec.Status,
		"priority":    rec.Priority,
		"color":       rec.Color,
		"rrule":       rec.Rrule,
		"tags":        tags,
		"parent_id":   rec.ParentID,
		"position":    rec.Position,
		"created_at":  rec.CreatedAt,
		"updated_at":  rec.UpdatedAt,
	}
	if rec.StartDate != nil {
		out["start_date"] = rec.StartDate.Format("2006-01-02")
	}
	if rec.EndDate != nil {
		out["end_date"] = rec.EndDate.Format("2006-01-02")
	}
	return out, nil
}

func calendarIDFromRaw(raw json.RawMessage, action string) (int, error) {
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("id must be a valid integer for %s action", action)
	}
	if id < 1 {
		return 0, fmt.Errorf("id is required and must be a positive integer for %s action", action)
	}
	return id, nil
}

func coalesceString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

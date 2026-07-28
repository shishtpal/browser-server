package todo

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var colorPattern = regexp.MustCompile(`^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

func ValidateCreateInput(input *CreateInput) error {
	fields := map[string]string{}

	if input.UserID < 1 {
		fields["user_id"] = "is required and must be a positive integer"
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		fields["title"] = "is required"
	} else if len(input.Title) > 500 {
		fields["title"] = "must be 500 characters or fewer"
	}
	if len(input.Description) > 2000 {
		fields["description"] = "must be 2000 characters or fewer"
	}

	if input.Color != "" {
		color := strings.TrimSpace(input.Color)
		if !isValidColor(color) {
			fields["color"] = "must be empty or a valid hex color code"
		} else {
			input.Color = color
		}
	}

	for i, tag := range input.Tags {
		if len(tag) > 100 {
			fields[fmt.Sprintf("tags[%d]", i)] = "must be 100 characters or fewer"
		}
	}

	priority := "medium"
	if input.Priority != "" {
		if !IsValidPriority(input.Priority) {
			fields["priority"] = "must be one of: low, medium, high, urgent"
		} else {
			priority = input.Priority
		}
	}
	input.Priority = priority

	status := "pending"
	if input.Status != "" {
		if !IsValidStatus(input.Status) {
			fields["status"] = "must be one of: pending, in_progress, completed, done, cancelled, archived"
		} else {
			status = input.Status
		}
	}
	input.Status = status

	if len(input.Subtasks) > 20 {
		fields["subtasks"] = "must have 20 items or fewer"
	}
	for i := range input.Subtasks {
		st := &input.Subtasks[i]
		st.Title = strings.TrimSpace(st.Title)
		if st.Title == "" {
			st.Title = fmt.Sprintf("Subtask %d", i+1)
		}
		if len(st.Title) > 500 {
			fields[fmt.Sprintf("subtasks[%d].title", i)] = "must be 500 characters or fewer"
		}
		if len(st.Description) > 2000 {
			fields[fmt.Sprintf("subtasks[%d].description", i)] = "must be 2000 characters or fewer"
		}
		if st.Priority != "" && !IsValidPriority(st.Priority) {
			fields[fmt.Sprintf("subtasks[%d].priority", i)] = "must be one of: low, medium, high, urgent"
		}
		if st.Status != "" && !IsValidStatus(st.Status) {
			fields[fmt.Sprintf("subtasks[%d].status", i)] = "must be one of: pending, in_progress, completed, done, cancelled, archived"
		}
		if st.Color != "" {
			st.Color = strings.TrimSpace(st.Color)
			if !isValidColor(st.Color) {
				fields[fmt.Sprintf("subtasks[%d].color", i)] = "must be empty or a valid hex color code"
			}
		}
		for j, tag := range st.Tags {
			if len(tag) > 100 {
				fields[fmt.Sprintf("subtasks[%d].tags[%d]", i, j)] = "must be 100 characters or fewer"
			}
		}
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func ValidateUpdateInput(input *UpdateInput) error {
	fields := map[string]string{}

	if input.UserID < 1 {
		fields["user_id"] = "is required and must be a positive integer"
	}
	if input.ID < 1 {
		fields["id"] = "is required and must be a positive integer"
	}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			fields["title"] = "must not be empty"
		} else if len(title) > 500 {
			fields["title"] = "must be 500 characters or fewer"
		}
		*input.Title = title
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if len(description) > 2000 {
			fields["description"] = "must be 2000 characters or fewer"
		}
		*input.Description = description
	}
	if input.Tags != nil {
		for i, tag := range *input.Tags {
			if len(tag) > 100 {
				fields[fmt.Sprintf("tags[%d]", i)] = "must be 100 characters or fewer"
			}
		}
	}
	if input.Priority != nil {
		if !IsValidPriority(*input.Priority) {
			fields["priority"] = "must be one of: low, medium, high, urgent"
		}
	}
	if input.Status != nil {
		if !IsValidStatus(*input.Status) {
			fields["status"] = "must be one of: pending, in_progress, completed, done, cancelled, archived"
		}
	}
	if input.Color != nil {
		color := strings.TrimSpace(*input.Color)
		if color != "" && !isValidColor(color) {
			fields["color"] = "must be empty or a valid hex color code"
		} else {
			*input.Color = color
		}
	}
	if input.Position != nil && *input.Position < 0 {
		fields["position"] = "must be a non-negative integer"
	}
	if input.StartDate != nil && *input.StartDate != "" {
		if ParseDate(*input.StartDate) == nil {
			fields["start_date"] = "must be a valid date (YYYY-MM-DD or RFC3339)"
		}
	}
	if input.EndDate != nil && *input.EndDate != "" {
		if ParseDate(*input.EndDate) == nil {
			fields["end_date"] = "must be a valid date (YYYY-MM-DD or RFC3339)"
		}
	}
	if input.Title == nil && input.Description == nil && input.Status == nil && input.Priority == nil &&
		input.Color == nil && input.Tags == nil && input.Position == nil && input.ParentID == nil &&
		input.StartDate == nil && input.EndDate == nil && input.Rrule == nil && input.Domain == nil &&
		input.ScreenshotPath == nil && input.Pinned == nil {
		fields["_body"] = "no updatable fields provided"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func ParseDate(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return &t
	}
	return nil
}

func IsValidPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "urgent":
		return true
	default:
		return false
	}
}

func IsValidStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "completed", "done", "cancelled", "archived":
		return true
	default:
		return false
	}
}

func isValidColor(color string) bool {
	return colorPattern.MatchString(color)
}

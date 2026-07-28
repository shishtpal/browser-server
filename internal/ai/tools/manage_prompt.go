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

//go:embed schemas/manage_prompt.json
var managePromptSchema []byte

func registerManagePrompt(r *Registry) {
	r.add(Tool{
		Name:        "manage_prompt",
		Category:    "General",
		Description: "Add, edit, or remove a prompt. Requires user_id and action. For add: title and content are required. For edit/remove: id is required.",
		Schema:      json.RawMessage(managePromptSchema),
		Execute:     managePrompt,
	})
}

func managePrompt(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID      int             `json:"user_id"`
		Action      string          `json:"action"`
		ID          json.RawMessage `json:"id"`
		Title       *string         `json:"title"`
		Content     *string         `json:"content"`
		Description *string         `json:"description"`
		FolderID    json.RawMessage `json:"folder_id"`
		Tags        *[]string       `json:"tags"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "action": true,
		"id": true, "title": true, "content": true,
		"description": true, "folder_id": true, "tags": true,
	}); err != nil {
		return nil, err
	}

	// Validate user_id
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}

	// Validate action
	switch a.Action {
	case "add", "edit", "remove":
	default:
		return nil, fmt.Errorf("action must be one of: add, edit, remove")
	}

	// Parse folder_id (nullable). We need to distinguish an explicit null from an omitted field.
	var folderID *int
	var folderIDSet bool
	if a.FolderID != nil {
		folderIDSet = true
		if string(a.FolderID) == "null" {
			folderID = nil // explicit clear
		} else {
			var fid int
			if err := json.Unmarshal(a.FolderID, &fid); err != nil || fid < 1 {
				return nil, fmt.Errorf("folder_id must be a positive integer or null")
			}
			folderID = &fid
		}
	}

	// Validate title/content/description for add action
	if a.Action == "add" {
		if a.Title == nil || strings.TrimSpace(*a.Title) == "" {
			return nil, fmt.Errorf("title is required for add action")
		}
		if len(strings.TrimSpace(*a.Title)) > 200 {
			return nil, fmt.Errorf("title must be 200 characters or fewer")
		}
		if a.Content == nil {
			return nil, fmt.Errorf("content is required for add action")
		}
		if len(*a.Content) > 10000 {
			return nil, fmt.Errorf("content must be 10000 characters or fewer")
		}
		if a.Description != nil && len(*a.Description) > 2000 {
			return nil, fmt.Errorf("description must be 2000 characters or fewer")
		}
	}

	// Validate title/description lengths for edit action
	if a.Action == "edit" {
		if a.Title != nil {
			title := strings.TrimSpace(*a.Title)
			if title == "" {
				return nil, fmt.Errorf("title must not be empty")
			}
			if len(title) > 200 {
				return nil, fmt.Errorf("title must be 200 characters or fewer")
			}
			*a.Title = title
		}
		if a.Content != nil && len(*a.Content) > 10000 {
			return nil, fmt.Errorf("content must be 10000 characters or fewer")
		}
		if a.Description != nil && len(*a.Description) > 2000 {
			return nil, fmt.Errorf("description must be 2000 characters or fewer")
		}
	}

	// Validate tags
	if a.Tags != nil {
		for i, tag := range *a.Tags {
			if len(tag) > 100 {
				return nil, fmt.Errorf("tags[%d] must be 100 characters or fewer", i)
			}
		}
	}

	// Validate folder ownership if folder_id is provided
	if folderID != nil {
		var folderUserID int
		err := db.PromptDB.QueryRow(
			"SELECT user_id FROM prompt_folders WHERE id = ?",
			*folderID,
		).Scan(&folderUserID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("folder not found")
		}
		if err != nil {
			return nil, err
		}
		if folderUserID != a.UserID {
			return nil, fmt.Errorf("folder does not belong to user")
		}
	}

	switch a.Action {
	case "add":
		return managePromptAdd(ctx, a.UserID, a.Title, a.Content, a.Description, folderID, a.Tags)
	case "edit":
		return managePromptEdit(ctx, a.UserID, a.ID, a.Title, a.Content, a.Description, folderID, folderIDSet, a.Tags)
	case "remove":
		return managePromptRemove(ctx, a.UserID, a.ID)
	default:
		return nil, fmt.Errorf("unknown action: %s", a.Action)
	}
}

func managePromptAdd(ctx context.Context, userID int, title, content *string, description *string, folderID *int, tags *[]string) (any, error) {
	titleStr := strings.TrimSpace(*title)
	contentStr := *content
	descriptionStr := ""
	if description != nil {
		descriptionStr = strings.TrimSpace(*description)
	}

	tagsJSON := "[]"
	if tags != nil {
		tagsJSON = helpers.TagsToJSON(*tags)
	}

	result, err := db.PromptDB.ExecContext(ctx, `
		INSERT INTO prompts (user_id, folder_id, title, content, description, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		userID, folderID, titleStr, contentStr, descriptionStr, tagsJSON)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return map[string]any{
		"id":          id,
		"user_id":     userID,
		"folder_id":   folderID,
		"title":       titleStr,
		"content":     contentStr,
		"description": descriptionStr,
		"tags":        helpers.ParseTagsFromJSON(tagsJSON),
		"created_at":  timeNow(),
		"updated_at":  timeNow(),
	}, nil
}

func managePromptEdit(ctx context.Context, userID int, idRaw json.RawMessage, title, content, description *string, folderID *int, folderIDSet bool, tags *[]string) (any, error) {
	// Parse and validate id
	var id int
	if err := json.Unmarshal(idRaw, &id); err != nil || id < 1 {
		return nil, fmt.Errorf("id is required for edit action")
	}

	// Verify ownership
	var existingUserID int
	err := db.PromptDB.QueryRow(
		"SELECT user_id FROM prompts WHERE id = ?",
		id,
	).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Prompt not found")
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != userID {
		return nil, fmt.Errorf("Prompt does not belong to user")
	}

	// Check if any updatable field was provided. An explicit null folder_id is a valid update.
	hasUpdate := title != nil || content != nil || description != nil || folderIDSet || tags != nil
	if !hasUpdate {
		return nil, fmt.Errorf("no updatable fields provided")
	}

	// Build dynamic UPDATE SET clause
	setClauses := []string{}
	args := []any{}

	if title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *title)
	}
	if content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *content)
	}
	if description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *description)
	}
	if folderIDSet {
		setClauses = append(setClauses, "folder_id = ?")
		args = append(args, folderID)
	}
	if tags != nil {
		setClauses = append(setClauses, "tags = ?")
		if len(*tags) == 0 {
			args = append(args, "[]")
		} else {
			args = append(args, helpers.TagsToJSON(*tags))
		}
	}

	setClause := strings.Join(setClauses, ", ")
	args = append(args, id)

	_, err = db.PromptDB.ExecContext(ctx,
		"UPDATE prompts SET "+setClause+", updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		args...,
	)
	if err != nil {
		return nil, err
	}

	// Fetch the updated prompt
	return fetchPromptByID(ctx, id)
}

func managePromptRemove(ctx context.Context, userID int, idRaw json.RawMessage) (any, error) {
	// Parse and validate id
	var id int
	if err := json.Unmarshal(idRaw, &id); err != nil || id < 1 {
		return nil, fmt.Errorf("id is required for remove action")
	}

	// Verify ownership
	var existingUserID int
	err := db.PromptDB.QueryRow(
		"SELECT user_id FROM prompts WHERE id = ?",
		id,
	).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Prompt not found")
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != userID {
		return nil, fmt.Errorf("Prompt does not belong to user")
	}

	_, err = db.PromptDB.ExecContext(ctx, "DELETE FROM prompts WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":      id,
		"user_id": userID,
		"removed": true,
	}, nil
}

func fetchPromptByID(ctx context.Context, id int) (map[string]any, error) {
	var (
		promptID    int
		userID      int
		folderID    sql.NullInt64
		title       string
		content     string
		description string
		tagsJSON    string
		createdAt   string
		updatedAt   string
	)
	err := db.PromptDB.QueryRowContext(ctx,
		"SELECT id, user_id, folder_id, title, content, description, tags, created_at, updated_at FROM prompts WHERE id = ?",
		id,
	).Scan(&promptID, &userID, &folderID, &title, &content, &description, &tagsJSON, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Prompt not found")
	}
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":          promptID,
		"user_id":     userID,
		"title":       title,
		"content":     content,
		"description": description,
		"tags":        helpers.ParseTagsFromJSON(tagsJSON),
		"created_at":  createdAt,
		"updated_at":  updatedAt,
	}
	if folderID.Valid {
		result["folder_id"] = int(folderID.Int64)
	} else {
		result["folder_id"] = nil
	}

	return result, nil
}

func timeNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

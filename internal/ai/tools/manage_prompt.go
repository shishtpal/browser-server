package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/helpers"
	"browser-server/internal/prompt"
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
		Tags        *[]string       `json:"tags"`
		Pinned      *bool           `json:"pinned"`
	}
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "action": true,
		"id": true, "title": true, "content": true,
		"description": true, "tags": true, "pinned": true,
	}); err != nil {
		return nil, err
	}

	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}

	switch a.Action {
	case "add", "edit", "remove":
	default:
		return nil, fmt.Errorf("action must be one of: add, edit, remove")
	}

	if a.Action == "add" {
		if a.Title == nil {
			return nil, fmt.Errorf("title is required for add action")
		}
		if _, err := prompt.ValidateTitle(*a.Title); err != nil {
			return nil, err
		}
		if a.Content == nil {
			return nil, fmt.Errorf("content is required for add action")
		}
		if err := prompt.ValidateContent(*a.Content); err != nil {
			return nil, err
		}
		if a.Description != nil {
			if err := prompt.ValidateDescription(*a.Description); err != nil {
				return nil, err
			}
		}
	}

	if a.Action == "edit" {
		if a.Title != nil {
			title, err := prompt.ValidateTitle(*a.Title)
			if err != nil {
				return nil, err
			}
			*a.Title = title
		}
		if a.Content != nil {
			if err := prompt.ValidateContent(*a.Content); err != nil {
				return nil, err
			}
		}
		if a.Description != nil {
			if err := prompt.ValidateDescription(*a.Description); err != nil {
				return nil, err
			}
		}
	}

	if a.Tags != nil {
		if err := prompt.ValidateTags(*a.Tags); err != nil {
			return nil, err
		}
	}

	switch a.Action {
	case "add":
		return managePromptAdd(ctx, a.UserID, a.Title, a.Content, a.Description, a.Tags, a.Pinned)
	case "edit":
		return managePromptEdit(ctx, a.UserID, a.ID, a.Title, a.Content, a.Description, a.Tags, a.Pinned)
	case "remove":
		return managePromptRemove(ctx, a.UserID, a.ID)
	default:
		return nil, fmt.Errorf("unknown action: %s", a.Action)
	}
}

func managePromptAdd(ctx context.Context, userID int, title, content, description *string, tags *[]string, pinned *bool) (any, error) {
	titleStr := strings.TrimSpace(*title)
	descriptionStr := ""
	if description != nil {
		descriptionStr = strings.TrimSpace(*description)
	}

	tagList := []string{}
	if tags != nil {
		tagList = *tags
	}

	pinnedVal := false
	if pinned != nil {
		pinnedVal = *pinned
	}

	id, _, err := prompt.Create(ctx, prompt.CreateInput{
		UserID:      userID,
		Title:       titleStr,
		Content:     *content,
		Description: descriptionStr,
		Tags:        tagList,
		Pinned:      pinnedVal,
	})
	if err != nil {
		return nil, err
	}

	rec, err := prompt.GetByID(ctx, int(id))
	if err != nil {
		return nil, err
	}
	return prompt.Map(rec), nil
}

func managePromptEdit(ctx context.Context, userID int, idRaw json.RawMessage, title, content, description *string, tags *[]string, pinned *bool) (any, error) {
	id, err := promptIDFromRaw(idRaw, "edit")
	if err != nil {
		return nil, err
	}

	rec, err := prompt.GetByID(ctx, id)
	if err == prompt.ErrNotFound {
		return nil, prompt.ErrPromptNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.Prompt.UserID != userID {
		return nil, prompt.ErrPromptNotOwned
	}

	builder := prompt.NewUpdateBuilder()
	if title != nil {
		builder.Set("title", *title)
	}
	if content != nil {
		builder.Set("content", *content)
	}
	if description != nil {
		builder.Set("description", *description)
	}
	if tags != nil {
		tagsJSON := "[]"
		if len(*tags) > 0 {
			tagsJSON = helpers.TagsToJSON(*tags)
		}
		builder.Set("tags", tagsJSON)
	}
	if pinned != nil {
		builder.Set("pinned", *pinned)
	}

	if builder.Empty() {
		return nil, fmt.Errorf("no updatable fields provided")
	}
	if err := builder.Exec(ctx, id); err != nil {
		return nil, err
	}

	rec, err = prompt.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return prompt.Map(rec), nil
}

func managePromptRemove(ctx context.Context, userID int, idRaw json.RawMessage) (any, error) {
	id, err := promptIDFromRaw(idRaw, "remove")
	if err != nil {
		return nil, err
	}
	if err := prompt.VerifyOwnership(ctx, id, userID); err != nil {
		return nil, err
	}
	if _, err := prompt.Delete(ctx, id); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":      id,
		"user_id": userID,
		"removed": true,
	}, nil
}

// promptIDFromRaw parses the id argument, naming the action in the error.
func promptIDFromRaw(raw json.RawMessage, action string) (int, error) {
	var id int
	if err := json.Unmarshal(raw, &id); err != nil || id < 1 {
		return 0, fmt.Errorf("id is required for %s action", action)
	}
	return id, nil
}

package prompt

import "browser-server/internal/models"

// Response renders a record as the REST API's PromptResponse.
func Response(rec Record) models.PromptResponse {
	return models.PromptResponse{
		Prompt: rec.Prompt,
		Tags:   rec.Tags(),
	}
}

// Responses renders many records for the REST API.
func Responses(records []Record) []models.PromptResponse {
	out := make([]models.PromptResponse, 0, len(records))
	for _, rec := range records {
		out = append(out, Response(rec))
	}
	return out
}

// Map renders a record as the map the AI tools return.
func Map(rec Record) map[string]any {
	return map[string]any{
		"id":          rec.Prompt.ID,
		"user_id":     rec.Prompt.UserID,
		"title":       rec.Prompt.Title,
		"content":     rec.Prompt.Content,
		"description": rec.Prompt.Description,
		"tags":        rec.Tags(),
		"pinned":      rec.Prompt.Pinned,
		"created_at":  rec.Prompt.CreatedAt,
		"updated_at":  rec.Prompt.UpdatedAt,
	}
}

// SearchMap renders the trimmed record shape the search_prompts tool returns.
func SearchMap(rec Record) map[string]any {
	return map[string]any{
		"id":          rec.Prompt.ID,
		"title":       rec.Prompt.Title,
		"content":     rec.Prompt.Content,
		"description": rec.Prompt.Description,
		"pinned":      rec.Prompt.Pinned,
	}
}

package quiz

import (
	"fmt"

	"browser-server/internal/models"
)

// Response renders a record as the REST API's QuestionResponse.
func Response(rec Record) models.QuestionResponse {
	resp := models.QuestionResponse{Question: rec.Question}
	resp.Tags = rec.Tags()

	switch rec.Question.Type {
	case "single_choice", "multiple_choice":
		resp.Options = rec.Options()
		resp.CorrectAnswers = rec.CorrectAnswers()
	case "chronology":
		resp.ChronologyItems = rec.ChronologyItems()
	case "input":
		resp.ExpectedText = rec.ExpectedText()
	}
	if rec.Question.ImageFilename != "" {
		resp.ImageURL = fmt.Sprintf("/api/quiz/questions/%d/image", rec.Question.ID)
	}
	return resp
}

// Responses renders many records for the REST API.
func Responses(records []Record) []models.QuestionResponse {
	out := make([]models.QuestionResponse, 0, len(records))
	for _, rec := range records {
		out = append(out, Response(rec))
	}
	return out
}

// Map renders a record as the map the AI tools return.
func Map(rec Record) map[string]any {
	q := rec.Question
	out := map[string]any{
		"id":          q.ID,
		"user_id":     q.UserID,
		"type":        q.Type,
		"difficulty":  q.Difficulty,
		"question":    q.Question,
		"explanation": q.Explanation,
		"tags":        rec.Tags(),
		"subject":     q.Subject,
		"topic":       q.Topic,
		"sub_topic":   q.SubTopic,
		"source":      q.Source,
		"created_at":  q.CreatedAt,
		"updated_at":  q.UpdatedAt,
		"image_url":   "",
	}
	switch q.Type {
	case "single_choice", "multiple_choice":
		out["options"] = rec.Options()
		out["correct_answers"] = rec.CorrectAnswers()
	case "chronology":
		out["chronology_items"] = rec.ChronologyItems()
	case "input":
		out["expected_text"] = rec.ExpectedText()
	}
	if q.ImageFilename != "" {
		out["image_url"] = fmt.Sprintf("/api/quiz/questions/%d/image", q.ID)
	}
	return out
}

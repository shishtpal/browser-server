package tools

import (
	"browser-server/internal/quiz"
	quizconfig "browser-server/internal/quiz/config"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
)

//go:embed schemas/manage_question.json
var manageQuestionSchema []byte

func registerManageQuestion(r *Registry) {
	r.add(Tool{
		Name:        "manage_question",
		Category:    "General",
		Description: "Add, edit, remove, get, list, or list_tags for questions in the question bank. Requires user_id and action. For add: type and question are required; options (with correct flags) for single_choice/multiple_choice, chronology_items for chronology, expected_text for input. For edit/remove/get: id is required. List accepts the same filters as search without the query. list_tags returns the distinct tags, subjects, topics, and sub-topics already used by the user so add/edit can reuse them instead of inventing duplicates.",
		Schema:      json.RawMessage(manageQuestionSchema),
		Execute:     manageQuestion,
	})
}

// questionArgs is the decoded manage_question argument shape. The editable
// question fields come from quiz.EditFields so the tool and the REST PUT
// handler stay on one wire shape and one validation path.
type questionArgs struct {
	UserID int             `json:"user_id"`
	Action string          `json:"action"`
	ID     json.RawMessage `json:"id"`
	quiz.EditFields
}

func manageQuestion(ctx context.Context, raw json.RawMessage) (any, error) {
	cfg := quizconfig.Get()
	if !cfg.Enabled {
		return nil, fmt.Errorf("quiz feature disabled")
	}

	var a questionArgs
	if err := strict(raw, &a, map[string]bool{
		"user_id": true, "action": true, "id": true, "type": true,
		"difficulty": true, "question": true, "explanation": true,
		"options": true, "chronology_items": true, "expected_text": true,
		"tags": true, "subject": true, "topic": true, "sub_topic": true, "source": true,
	}); err != nil {
		return nil, err
	}

	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required and must be a positive integer")
	}

	rules := cfg.Rules()
	switch a.Action {
	case "add":
		return manageQuestionAdd(ctx, a, rules)
	case "edit":
		return manageQuestionEdit(ctx, a, rules)
	case "remove":
		return manageQuestionRemove(ctx, a)
	case "get":
		return manageQuestionGet(ctx, a)
	case "list":
		return manageQuestionList(ctx, a)
	case "list_tags":
		return quiz.TagVocabulary(ctx, a.UserID)
	default:
		return nil, fmt.Errorf("action must be one of: add, edit, remove, get, list, list_tags")
	}
}

func manageQuestionAdd(ctx context.Context, a questionArgs, rules quiz.Rules) (any, error) {
	if a.Type == nil {
		return nil, fmt.Errorf("type is required for add action")
	}
	if a.Question == nil {
		return nil, fmt.Errorf("question is required for add action")
	}

	createInput, err := rules.BuildCreate(quiz.CreateFields{
		UserID:      a.UserID,
		Type:        *a.Type,
		Difficulty:  derefStr(a.Difficulty),
		Question:    *a.Question,
		Explanation: derefStr(a.Explanation),
		Payload: quiz.AnswerPayload{
			Options:         derefSlice(a.Options),
			ChronologyItems: derefSlice(a.ChronologyItems),
			ExpectedText:    derefStr(a.ExpectedText),
		},
		Tags:     derefSlice(a.Tags),
		Subject:  derefStr(a.Subject),
		Topic:    derefStr(a.Topic),
		SubTopic: derefStr(a.SubTopic),
		Source:   derefStr(a.Source),
	})
	if err != nil {
		return nil, err
	}

	id, err := quiz.Create(ctx, createInput)
	if err != nil {
		return nil, err
	}

	rec, err := quiz.GetByID(ctx, int(id))
	if err != nil {
		return nil, err
	}
	return quiz.Map(rec), nil
}

func manageQuestionEdit(ctx context.Context, a questionArgs, rules quiz.Rules) (any, error) {
	id, err := questionIDFromRaw(a.ID, "edit")
	if err != nil {
		return nil, err
	}

	rec, err := loadOwnedQuestion(ctx, id, a.UserID)
	if err != nil {
		return nil, err
	}

	builder, err := rules.BuildUpdate(a.EditFields, rec.Question.Type)
	if err != nil {
		return nil, err
	}
	if builder.Empty() {
		return nil, fmt.Errorf("no updatable fields provided")
	}
	if err := builder.Exec(ctx, id); err != nil {
		return nil, err
	}

	rec, err = quiz.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return quiz.Map(rec), nil
}

func manageQuestionRemove(ctx context.Context, a questionArgs) (any, error) {
	id, err := questionIDFromRaw(a.ID, "remove")
	if err != nil {
		return nil, err
	}
	if err := quiz.VerifyOwnership(ctx, id, a.UserID); err != nil {
		return nil, err
	}
	if _, err := quiz.Delete(ctx, id); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":      id,
		"user_id": a.UserID,
		"removed": true,
	}, nil
}

func manageQuestionGet(ctx context.Context, a questionArgs) (any, error) {
	id, err := questionIDFromRaw(a.ID, "get")
	if err != nil {
		return nil, err
	}
	rec, err := loadOwnedQuestion(ctx, id, a.UserID)
	if err != nil {
		return nil, err
	}
	return quiz.Map(rec), nil
}

func manageQuestionList(ctx context.Context, a questionArgs) (any, error) {
	records, err := quiz.List(ctx, quiz.ListQuery{
		Filter: quiz.Filter{
			UserID:     a.UserID,
			Type:       derefStr(a.Type),
			Difficulty: derefStr(a.Difficulty),
			Tags:       derefSlice(a.Tags),
			Subject:    derefStr(a.Subject),
			Topic:      derefStr(a.Topic),
			SubTopic:   derefStr(a.SubTopic),
		},
		Limit: 50,
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, quiz.Map(rec))
	}
	return map[string]any{"results": out, "total": len(out)}, nil
}

// loadOwnedQuestion fetches a question and confirms it belongs to userID, so
// a tool call cannot read or edit another user's row.
func loadOwnedQuestion(ctx context.Context, id, userID int) (quiz.Record, error) {
	rec, err := quiz.GetByID(ctx, id)
	if errors.Is(err, quiz.ErrNotFound) {
		return rec, quiz.ErrQuestionNotFound
	}
	if err != nil {
		return rec, err
	}
	if rec.Question.UserID != userID {
		return rec, quiz.ErrQuestionNotOwned
	}
	return rec, nil
}

// questionIDFromRaw parses the id argument, naming the action in the error.
func questionIDFromRaw(raw json.RawMessage, action string) (int, error) {
	var id int
	if err := json.Unmarshal(raw, &id); err != nil || id < 1 {
		return 0, fmt.Errorf("id is required for %s action", action)
	}
	return id, nil
}

func derefSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

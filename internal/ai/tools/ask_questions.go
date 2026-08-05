package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed schemas/ask_questions.json
var askQuestionsSchema []byte

// Question is one clarification requested from the user.
type Question struct {
	ID       string   `json:"id,omitempty"`
	Prompt   string   `json:"prompt"`
	Kind     string   `json:"kind,omitempty"`
	Options  []string `json:"options,omitempty"`
	Default  string   `json:"default,omitempty"`
	Required bool     `json:"required,omitempty"`
}

// QuestionRequest is the validated argument passed to the question tool.
type QuestionRequest struct {
	Context   string     `json:"context,omitempty"`
	Questions []Question `json:"questions"`
}

// ValidateQuestionArguments validates and normalizes a question tool call.
// It is exported so the chat orchestrator can validate before suspending a
// generation on the existing tool-decision channel.
func ValidateQuestionArguments(raw json.RawMessage) (QuestionRequest, error) {
	var request QuestionRequest
	if err := strict(raw, &request, map[string]bool{"context": true, "questions": true}); err != nil {
		return request, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return request, fmt.Errorf("arguments must be a JSON object")
	}
	var rawQuestions []json.RawMessage
	if err := json.Unmarshal(envelope["questions"], &rawQuestions); err != nil {
		return request, fmt.Errorf("questions must be an array")
	}
	if len(request.Questions) == 0 || len(request.Questions) > 20 {
		return request, fmt.Errorf("questions must contain between 1 and 20 items")
	}
	if len(request.Context) > 2000 {
		return request, fmt.Errorf("context must be 2000 characters or fewer")
	}
	request.Context = strings.TrimSpace(request.Context)
	seenIDs := map[string]bool{}
	for i := range request.Questions {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawQuestions[i], &fields); err != nil || fields == nil {
			return request, fmt.Errorf("questions[%d] must be an object", i)
		}
		for field := range fields {
			switch field {
			case "id", "prompt", "kind", "options", "default", "required":
			default:
				return request, fmt.Errorf("questions[%d]: unknown argument %q", i, field)
			}
		}
		question := &request.Questions[i]
		question.ID = strings.TrimSpace(question.ID)
		question.Prompt = strings.TrimSpace(question.Prompt)
		if question.Prompt == "" {
			return request, fmt.Errorf("questions[%d].prompt is required", i)
		}
		if len(question.Prompt) > 2000 {
			return request, fmt.Errorf("questions[%d].prompt must be 2000 characters or fewer", i)
		}
		if question.ID == "" {
			question.ID = fmt.Sprintf("q%d", i+1)
		}
		if seenIDs[question.ID] {
			return request, fmt.Errorf("questions[%d].id %q is duplicated", i, question.ID)
		}
		seenIDs[question.ID] = true
		if question.Kind == "" {
			question.Kind = "text"
		}
		// Accept the more readable spelling as an alias while keeping the
		// established multi_choice value on the wire.
		if question.Kind == "multiple_choice" {
			question.Kind = "multi_choice"
		}
		switch question.Kind {
		case "text", "choice", "multi_choice", "confirm":
		default:
			return request, fmt.Errorf("questions[%d].kind must be text, choice, multi_choice, or confirm", i)
		}
		if len(question.Options) > 20 {
			return request, fmt.Errorf("questions[%d].options must contain 20 items or fewer", i)
		}
		if (question.Kind == "choice" || question.Kind == "multi_choice") && len(question.Options) == 0 {
			return request, fmt.Errorf("questions[%d].options is required for %s questions", i, question.Kind)
		}
		if question.Kind == "confirm" && len(question.Options) > 0 {
			return request, fmt.Errorf("questions[%d].options is not allowed for confirm questions", i)
		}
		for optionIndex := range question.Options {
			question.Options[optionIndex] = strings.TrimSpace(question.Options[optionIndex])
			if question.Options[optionIndex] == "" {
				return request, fmt.Errorf("questions[%d].options[%d] must not be empty", i, optionIndex)
			}
		}
		question.Default = strings.TrimSpace(question.Default)
	}
	return request, nil
}

func registerAskQuestions(r *Registry) {
	r.add(Tool{
		Name:           "ask_questions",
		Category:       "Interactive",
		Description:    "Ask the user up to 20 concise clarification questions when required information is genuinely missing or a consequential choice cannot be safely inferred. Do not ask for information discoverable with other tools, trivial defaults, or questions already answered. The call pauses until the user responds; include context and use choice/multi_choice/confirm when appropriate.",
		Schema:         json.RawMessage(askQuestionsSchema),
		Execute:        ask_questions,
		RawContentFunc: rawAskQuestionsResult,
	})
}

func ask_questions(_ context.Context, raw json.RawMessage) (any, error) {
	request, err := ValidateQuestionArguments(raw)
	if err != nil {
		return nil, err
	}
	// The chat service supplies the interactive transport. Direct execution
	// (including YOLO/non-interactive mode) must degrade to a model-readable
	// result instead of pretending that an answer was collected.
	return map[string]any{
		"status":    "unavailable",
		"questions": request.Questions,
		"hint":      "No interactive answer channel is available; proceed with explicit assumptions.",
	}, nil
}

// rawAskQuestionsResult keeps interactive answer results small in raw mode so
// a large question set does not unnecessarily consume model context.
func rawAskQuestionsResult(value any) ([]byte, bool) {
	result, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	status, ok := result["status"].(string)
	if !ok || status == "" {
		return nil, false
	}
	answers, _ := result["answers"].([]any)
	if answers == nil {
		if typed, ok := result["answers"].([]map[string]any); ok {
			answers = make([]any, len(typed))
			for i := range typed {
				answers[i] = typed[i]
			}
		}
	}
	// A result with no answer records means every question was effectively
	// skipped; only a concrete non-skipped answer flips this to false.
	skipped := true
	parts := make([]string, 0, len(answers))
	for _, item := range answers {
		answer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := answer["id"].(string)
		isSkipped, _ := answer["skipped"].(bool)
		if !isSkipped {
			skipped = false
		}
		if id == "" || isSkipped {
			continue
		}
		parts = append(parts, id+":"+compactQuestionAnswer(answer["answer"]))
	}
	output := fmt.Sprintf("status=%s; skipped=%t", status, skipped)
	if len(parts) > 0 {
		output += "; answers=" + strings.Join(parts, ",")
	}
	// In raw mode the direct-execution "unavailable" result would otherwise
	// drop the hint that tells the model how to proceed, so include it here.
	if status == "unavailable" {
		if hint, ok := result["hint"].(string); ok && hint != "" {
			output += "; hint=" + hint
		}
	}
	return []byte(output), true
}

func compactQuestionAnswer(value any) string {
	if values, ok := value.([]string); ok {
		return strings.Join(values, "+")
	}
	if values, ok := value.([]any); ok {
		out := make([]string, 0, len(values))
		for _, item := range values {
			out = append(out, fmt.Sprint(item))
		}
		return strings.Join(out, "+")
	}
	return fmt.Sprint(value)
}

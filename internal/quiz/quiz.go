// Package quiz holds the question-bank domain logic shared by the REST API
// handlers in internal/handlers and the AI tools in internal/ai/tools. It
// mirrors the internal/prompt layering: validation (quiz.go), persistence
// (store.go), response shapes (view.go), fuzzy search candidates (search.go),
// and sectioned paper generation (paper.go).
package quiz

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/models"
)

// Default field limits. They mirror the defaults the bs-quiz-config.json
// loader applies; the operator's configured values reach the validators
// through Rules, so these are only the fallback for a zero config.
const (
	DefaultMaxQuestionLength     = 2000
	DefaultMaxExplanationLength  = 20000
	DefaultMaxOptionLength       = 500
	DefaultMaxChronologyItems    = 20
	DefaultMaxOptionsPerQuestion = 10
	DefaultMaxPaperSize          = 200
)

// Limits bounds user-supplied question payloads. Values come from the
// `limits` section of bs-quiz-config.json.
type Limits struct {
	MaxQuestionLength     int
	MaxExplanationLength  int
	MaxOptionLength       int
	MaxChronologyItems    int
	MaxOptionsPerQuestion int
	MaxPaperSize          int
}

// DefaultLimits returns the built-in limits used when no config is loaded.
func DefaultLimits() Limits {
	return Limits{
		MaxQuestionLength:     DefaultMaxQuestionLength,
		MaxExplanationLength:  DefaultMaxExplanationLength,
		MaxOptionLength:       DefaultMaxOptionLength,
		MaxChronologyItems:    DefaultMaxChronologyItems,
		MaxOptionsPerQuestion: DefaultMaxOptionsPerQuestion,
		MaxPaperSize:          DefaultMaxPaperSize,
	}
}

// Rules is the operator-configured validation policy: the payload limits plus
// the allowed enum vocabularies. The REST handlers and the AI tools both build
// one from bs-quiz-config.json so a single code path enforces both.
type Rules struct {
	Limits              Limits
	AllowedTypes        []string
	AllowedDifficulties []string
}

// DefaultRules returns the built-in policy. Used as a fallback when the
// config carries no explicit vocabulary, and by tests.
func DefaultRules() Rules {
	return Rules{
		Limits:              DefaultLimits(),
		AllowedTypes:        []string{"single_choice", "multiple_choice", "input", "chronology"},
		AllowedDifficulties: []string{"easy", "medium", "hard"},
	}
}

// FieldError attributes a validation failure to a named request field so the
// REST layer can render a per-field validation response while AI tools can
// surface the message alone.
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string { return e.Err.Error() }
func (e *FieldError) Unwrap() error { return e.Err }

func fieldErrorf(field, format string, args ...any) error {
	return &FieldError{Field: field, Err: fmt.Errorf(format, args...)}
}

// ChoiceTypes require options with at least one correct answer.
var choiceTypes = map[string]bool{"single_choice": true, "multiple_choice": true}

// IsChoiceType reports whether a question type carries selectable options.
func IsChoiceType(t string) bool { return choiceTypes[t] }

// ValidateQuestionText trims and length-checks the required question body.
func (r Rules) ValidateQuestionText(s string) error {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return fieldErrorf("question", "question must not be empty")
	}
	if len(trimmed) > r.Limits.MaxQuestionLength {
		return fieldErrorf("question", "question must be %d characters or fewer", r.Limits.MaxQuestionLength)
	}
	return nil
}

// ValidateExplanation length-checks an optional explanation.
func (r Rules) ValidateExplanation(s string) error {
	if len(s) > r.Limits.MaxExplanationLength {
		return fieldErrorf("explanation", "explanation must be %d characters or fewer", r.Limits.MaxExplanationLength)
	}
	return nil
}

// ValidateOptionText length-checks a single option's text.
func (r Rules) ValidateOptionText(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("option text must not be empty")
	}
	if len(s) > r.Limits.MaxOptionLength {
		return fmt.Errorf("option text must be %d characters or fewer", r.Limits.MaxOptionLength)
	}
	return nil
}

// ValidateDifficulty ensures the difficulty is one of the allowed values.
func (r Rules) ValidateDifficulty(d string) error {
	if d == "" {
		return nil // empty means "use the default"
	}
	for _, a := range r.AllowedDifficulties {
		if d == a {
			return nil
		}
	}
	return fieldErrorf("difficulty", "difficulty must be one of: %s", strings.Join(r.AllowedDifficulties, ", "))
}

// ValidateType ensures the question type is one of the allowed values.
func (r Rules) ValidateType(t string) error {
	for _, a := range r.AllowedTypes {
		if t == a {
			return nil
		}
	}
	return fieldErrorf("type", "type must be one of: %s", strings.Join(r.AllowedTypes, ", "))
}

// ValidateChronologyItems checks a chronology question's item list: 2..Max
// items, every correct_order is a permutation of 1..len(items).
func (r Rules) ValidateChronologyItems(items []models.ChronologyItem) error {
	if len(items) < 2 {
		return fmt.Errorf("chronology items must contain at least 2 items")
	}
	if len(items) > r.Limits.MaxChronologyItems {
		return fmt.Errorf("chronology items must be %d or fewer", r.Limits.MaxChronologyItems)
	}
	seen := make(map[int]bool, len(items))
	for i, item := range items {
		if strings.TrimSpace(item.Text) == "" {
			return fmt.Errorf("chronology_items[%d] text must not be empty", i)
		}
		if len(item.Text) > r.Limits.MaxOptionLength {
			return fmt.Errorf("chronology_items[%d] text must be %d characters or fewer", i, r.Limits.MaxOptionLength)
		}
		if item.CorrectOrder < 1 || item.CorrectOrder > len(items) {
			return fmt.Errorf("chronology_items[%d] correct_order must be between 1 and %d", i, len(items))
		}
		if seen[item.CorrectOrder] {
			return fmt.Errorf("chronology correct_order values must be unique")
		}
		seen[item.CorrectOrder] = true
	}
	return nil
}

// ValidateOptions enforces the option rules for choice question types:
// 2..MaxOptionsPerQuestion options, and at least one correct answer (exactly
// one for single_choice). Non-choice types must not carry options.
func (r Rules) ValidateOptions(options []models.QuestionOption, questionType string) error {
	if !choiceTypes[questionType] {
		if len(options) > 0 {
			return fmt.Errorf("options are only valid for choice question types")
		}
		return nil
	}
	if len(options) < 2 {
		return fmt.Errorf("options must contain at least 2 items for %s", questionType)
	}
	if len(options) > r.Limits.MaxOptionsPerQuestion {
		return fmt.Errorf("options must be %d or fewer", r.Limits.MaxOptionsPerQuestion)
	}
	correct := 0
	seen := make(map[int]bool, len(options))
	for i, opt := range options {
		if err := r.ValidateOptionText(opt.Text); err != nil {
			return fmt.Errorf("options[%d]: %w", i, err)
		}
		if opt.Index < 0 || opt.Index >= len(options) {
			return fmt.Errorf("options[%d] index must be between 0 and %d", i, len(options)-1)
		}
		if seen[opt.Index] {
			return fmt.Errorf("option indexes must be unique")
		}
		seen[opt.Index] = true
		if opt.Correct {
			correct++
		}
	}
	if correct == 0 {
		return fmt.Errorf("at least one option must be marked correct")
	}
	if questionType == "single_choice" && correct != 1 {
		return fmt.Errorf("single_choice questions must have exactly one correct option")
	}
	return nil
}

// AnswerPayload carries the type-specific answer inputs of a create/edit
// request. Exactly one group is meaningful per question type.
type AnswerPayload struct {
	Options         []models.QuestionOption
	ChronologyItems []models.ChronologyItem
	ExpectedText    string
}

// EncodeAnswerPayload validates the payload against questionType and returns
// the (options_json, answer_json) column values to store. It is the single
// encoder shared by the REST handlers and the manage_question tool.
func (r Rules) EncodeAnswerPayload(questionType string, p AnswerPayload) (string, string, error) {
	switch questionType {
	case "single_choice", "multiple_choice":
		if err := r.ValidateOptions(p.Options, questionType); err != nil {
			return "", "", &FieldError{Field: "options", Err: err}
		}
		b, err := json.Marshal(p.Options)
		if err != nil {
			return "", "", err
		}
		return string(b), "[]", nil
	case "chronology":
		if err := r.ValidateChronologyItems(p.ChronologyItems); err != nil {
			return "", "", &FieldError{Field: "chronology_items", Err: err}
		}
		b, err := json.Marshal(p.ChronologyItems)
		if err != nil {
			return "", "", err
		}
		return string(b), "[]", nil
	case "input":
		if strings.TrimSpace(p.ExpectedText) == "" {
			return "", "", fieldErrorf("expected_text", "expected_text is required for input questions")
		}
		if len(p.ExpectedText) > r.Limits.MaxOptionLength {
			return "", "", fieldErrorf("expected_text", "expected_text must be %d characters or fewer", r.Limits.MaxOptionLength)
		}
		b, err := json.Marshal(map[string]string{"text": p.ExpectedText})
		if err != nil {
			return "", "", err
		}
		return "[]", string(b), nil
	default:
		return "", "", fieldErrorf("type", "unsupported type %q", questionType)
	}
}

// CreateFields is the validated create request shared by the REST POST
// handler and the manage_question tool's add action.
type CreateFields struct {
	UserID      int
	Type        string
	Difficulty  string
	Question    string
	Explanation string
	Payload     AnswerPayload
	Tags        []string
	Subject     string
	Topic       string
	SubTopic    string
	Source      string
}

// BuildCreate validates the request and returns the CreateInput to persist.
// A blank difficulty defaults to "medium".
func (r Rules) BuildCreate(in CreateFields) (CreateInput, error) {
	if err := r.ValidateType(in.Type); err != nil {
		return CreateInput{}, err
	}
	if err := r.ValidateQuestionText(in.Question); err != nil {
		return CreateInput{}, err
	}
	if in.Difficulty == "" {
		in.Difficulty = "medium"
	}
	if err := r.ValidateDifficulty(in.Difficulty); err != nil {
		return CreateInput{}, err
	}
	if err := r.ValidateExplanation(in.Explanation); err != nil {
		return CreateInput{}, err
	}
	optionsJSON, answerJSON, err := r.EncodeAnswerPayload(in.Type, in.Payload)
	if err != nil {
		return CreateInput{}, err
	}
	return CreateInput{
		UserID:      in.UserID,
		Type:        in.Type,
		Difficulty:  in.Difficulty,
		Question:    strings.TrimSpace(in.Question),
		Explanation: in.Explanation,
		Options:     optionsJSON,
		Answer:      answerJSON,
		Tags:        in.Tags,
		Subject:     in.Subject,
		Topic:       in.Topic,
		SubTopic:    in.SubTopic,
		Source:      in.Source,
	}, nil
}

// EditFields is a partial question update: a nil pointer means "leave the
// column alone". It is shared by the REST PUT handler and the
// manage_question tool's edit action, and carries the JSON tags of the
// (identical) wire shape both accept.
type EditFields struct {
	Type            *string                  `json:"type"`
	Difficulty      *string                  `json:"difficulty"`
	Question        *string                  `json:"question"`
	Explanation     *string                  `json:"explanation"`
	Options         *[]models.QuestionOption `json:"options"`
	ChronologyItems *[]models.ChronologyItem `json:"chronology_items"`
	ExpectedText    *string                  `json:"expected_text"`
	Tags            *[]string                `json:"tags"`
	Subject         *string                  `json:"subject"`
	Topic           *string                  `json:"topic"`
	SubTopic        *string                  `json:"sub_topic"`
	Source          *string                  `json:"source"`
}

// BuildUpdate validates the supplied fields and returns the accumulated SET
// clauses. currentType is the question's stored type, used to validate answer
// payloads when the request does not change the type.
func (r Rules) BuildUpdate(in EditFields, currentType string) (*UpdateBuilder, error) {
	effectiveType := currentType
	if in.Type != nil {
		if err := r.ValidateType(*in.Type); err != nil {
			return nil, err
		}
		effectiveType = *in.Type
	}

	b := NewUpdateBuilder()
	if in.Type != nil {
		b.Set("type", *in.Type)
	}
	// A type change without resupplying answer payloads would leave the old
	// options_json/answer_json pointing at the wrong shape (e.g. choice options
	// sticking around after flipping to "input"). Reset them so the row stays
	// consistent with the new type until a follow-up edit supplies matching
	// payloads.
	hasPayload := in.Options != nil || in.ChronologyItems != nil || in.ExpectedText != nil
	if in.Type != nil && !hasPayload {
		b.Set("options_json", "[]")
		b.Set("answer_json", "[]")
	}
	if in.Difficulty != nil {
		if err := r.ValidateDifficulty(*in.Difficulty); err != nil {
			return nil, err
		}
		b.Set("difficulty", *in.Difficulty)
	}
	if in.Question != nil {
		if err := r.ValidateQuestionText(*in.Question); err != nil {
			return nil, err
		}
		b.Set("question", strings.TrimSpace(*in.Question))
	}
	if in.Explanation != nil {
		if err := r.ValidateExplanation(*in.Explanation); err != nil {
			return nil, err
		}
		b.Set("explanation", *in.Explanation)
	}
	if hasPayload {
		optionsJSON, answerJSON, err := r.EncodeAnswerPayload(effectiveType, AnswerPayload{
			Options:         derefSlice(in.Options),
			ChronologyItems: derefSlice(in.ChronologyItems),
			ExpectedText:    derefString(in.ExpectedText),
		})
		if err != nil {
			return nil, err
		}
		b.Set("options_json", optionsJSON)
		b.Set("answer_json", answerJSON)
	}
	if in.Tags != nil {
		b.Set("tags", EncodeTags(*in.Tags))
	}
	for _, pair := range []struct {
		value  *string
		column string
	}{
		{in.Subject, "subject"},
		{in.Topic, "topic"},
		{in.SubTopic, "sub_topic"},
		{in.Source, "source"},
	} {
		if pair.value != nil {
			b.Set(pair.column, *pair.value)
		}
	}
	return b, nil
}

func derefSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

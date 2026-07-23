package provider

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ChatRequest struct {
	Provider        string
	Model           string
	Messages        []Message
	Temperature     float64
	MaxOutputTokens int
	Tools           []ToolSpec
}

type ChatResponse struct {
	Content     string
	Usage       Usage
	HTTPStatus  int
	Latency     time.Duration
	RawRequest  []byte
	RawResponse []byte
	ToolCalls   []ToolCall
}

type Usage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`
}

type Client interface {
	Complete(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest, emit func(Event) error) (ChatResponse, error)
}

type Event struct {
	Type     string
	Text     string
	ToolCall *ToolCall
	Usage    Usage
}

type Error struct {
	Code       string
	Status     int
	Retryable  bool
	Diagnostic string
}

func (e *Error) Error() string {
	if e.Diagnostic != "" {
		return e.Code + ": " + e.Diagnostic
	}
	return e.Code
}
func SafeError(err error) (string, int, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Code, e.Status, e.Retryable
	}
	return "provider_error", 502, false
}

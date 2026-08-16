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
	// ImageParts, when non-empty, serializes Content as an array of OpenAI
	// content parts ([{type:text}, {type:image_url}...]) instead of a plain
	// string. Built from validated private files at request time; never
	// persisted and never logged.
	ImageParts []ImagePart `json:"-"`
}

// ImagePart is one multimodal image payload (an RFC 2397 data URL) for
// vision-capable OpenAI-compatible models.
type ImagePart struct {
	DataURL string
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
	// GoogleSearch opts the request into the provider's native web-search
	// tool (Gemini's google_search) in addition to server-side tools. It is
	// set from per-provider config and ignored by OpenAI-compatible clients.
	GoogleSearch bool
}

type ChatResponse struct {
	Content string
	// Reasoning is the model's reasoning content (e.g. OpenRouter's
	// "reasoning" / "reasoning_content" fields). The chat service streams
	// it live and persists the accumulated value on the assistant message.
	Reasoning   string
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

// New builds a provider client for the given provider type ("openai_compatible"
// or "gemini_interactions"). Unknown types fall back to the OpenAI-compatible
// client; config validation rejects them before any client is constructed.
func New(typ, baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration, openRouterSiteURL, openRouterAppName string) Client {
	if typ == "gemini_interactions" {
		return NewGeminiInteractionsClient(baseURL, apiKey, timeout, retryAttempts, retryDelay)
	}
	return NewOpenAICompatibleClient(baseURL, apiKey, timeout, retryAttempts, retryDelay, openRouterSiteURL, openRouterAppName)
}

type Event struct {
	Type string
	// Text carries the chunk for text_delta and reasoning_delta events.
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

package store

import "time"

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Profile   string    `json:"profile"`
	Skills    []string  `json:"skills,omitempty"`
	Preview   string    `json:"preview,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Archived  bool      `json:"archived"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	ToolCallID     string    `json:"tool_call_id,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	// Attachments is populated for user messages when loading a conversation
	// (ListMessages/GetConversation) and when a turn is created. It is not
	// scanned from the messages table itself.
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment is a server-issued image attachment for a conversation. The image
// bytes live on disk under the AI data directory; only the server-controlled
// StorageKey is persisted here so the client can never address arbitrary files.
type Attachment struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id,omitempty"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	Width          int       `json:"width,omitempty"`
	Height         int       `json:"height,omitempty"`
	StorageKey     string    `json:"-"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type RequestLog struct {
	ID               string     `json:"id"`
	ConversationID   string     `json:"conversation_id,omitempty"`
	MessageID        string     `json:"message_id,omitempty"`
	Source           string     `json:"source"`
	TaskID           string     `json:"task_id,omitempty"`
	Iteration        int        `json:"iteration"`
	Provider         string     `json:"provider"`
	Model            string     `json:"model"`
	Endpoint         string     `json:"endpoint"`
	RequestPayload   string     `json:"request_payload,omitempty"`
	ResponsePayload  string     `json:"response_payload,omitempty"`
	PayloadTruncated bool       `json:"payload_truncated"`
	HTTPStatus       *int       `json:"http_status,omitempty"`
	PromptTokens     *int       `json:"prompt_tokens,omitempty"`
	CompletionTokens *int       `json:"completion_tokens,omitempty"`
	TotalTokens      *int       `json:"total_tokens,omitempty"`
	LatencyMS        int64      `json:"latency_ms"`
	Status           string     `json:"status"`
	ErrorCode        string     `json:"error_code,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID               string    `json:"id"`
	RequestID        string    `json:"request_id"`
	MessageID        string    `json:"message_id,omitempty"`
	ToolName         string    `json:"tool_name"`
	Arguments        string    `json:"arguments,omitempty"`
	Result           string    `json:"result,omitempty"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	Status           string    `json:"status"`
	Decision         string    `json:"decision"`
	DurationMS       int64     `json:"duration_ms"`
	PayloadTruncated bool      `json:"payload_truncated"`
	CreatedAt        time.Time `json:"created_at"`
}

type LogFilter struct {
	Source, Status, ConversationID, TaskID string
	Limit, Offset                          int
}
type Monitoring struct {
	WindowHours      int        `json:"window_hours"`
	Requests         int64      `json:"requests"`
	Errors           int64      `json:"errors"`
	Cancellations    int64      `json:"cancellations"`
	ToolSuccesses    int64      `json:"tool_successes"`
	ToolErrors       int64      `json:"tool_errors"`
	ToolRejections   int64      `json:"tool_rejections"`
	PromptTokens     int64      `json:"prompt_tokens"`
	CompletionTokens int64      `json:"completion_tokens"`
	TotalTokens      int64      `json:"total_tokens"`
	AverageLatencyMS float64    `json:"average_latency_ms"`
	MaxLatencyMS     int64      `json:"max_latency_ms"`
	LatestActivity   *time.Time `json:"latest_activity,omitempty"`
}

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
}

type RequestLog struct {
	ID               string
	ConversationID   string
	MessageID        string
	Provider         string
	Model            string
	Endpoint         string
	RequestPayload   string
	ResponsePayload  string
	PayloadTruncated bool
	HTTPStatus       *int
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
	LatencyMS        int64
	Status           string
	ErrorCode        string
	ErrorMessage     string
}

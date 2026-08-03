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
	ID             string       `json:"id"`
	ConversationID string       `json:"conversation_id"`
	Role           string       `json:"role"`
	Content        string       `json:"content"`
	ToolCallID     string       `json:"tool_call_id,omitempty"`
	Status         string       `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
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

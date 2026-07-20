package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Profile   string    `json:"profile"`
	Preview   string    `json:"preview,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		db.Close()
		return nil, fmt.Errorf("verify sqlite foreign_keys: value=%d err=%v", foreignKeys, err)
	}
	if _, err := db.Exec(`UPDATE messages SET status = 'cancelled' WHERE status = 'pending'`); err != nil {
		db.Close()
		return nil, fmt.Errorf("reconcile pending messages: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
		`INSERT INTO schema_version (version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
			content TEXT NOT NULL DEFAULT '',
			tool_call_id TEXT,
			status TEXT NOT NULL CHECK (status IN ('pending','completed','error','cancelled','superseded')),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id TEXT PRIMARY KEY,
			conversation_id TEXT REFERENCES conversations(id) ON DELETE SET NULL,
			message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			request_payload TEXT,
			response_payload TEXT,
			payload_truncated INTEGER NOT NULL DEFAULT 0,
			http_status INTEGER,
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			latency_ms INTEGER NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('success','error','cancelled')),
			error_code TEXT,
			error_message TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tool_calls (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL REFERENCES request_logs(id) ON DELETE CASCADE,
			message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
			tool_name TEXT NOT NULL,
			arguments TEXT NOT NULL,
			result TEXT,
			error_message TEXT,
			status TEXT NOT NULL CHECK (status IN ('success','error','cancelled','rejected')),
			duration_ms INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_conversation_created ON request_logs(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_request ON tool_calls(request_id)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	// Incremental migrations keyed by schema_version
	migrations := []struct {
		version    int
		statements []string
	}{
		{2, []string{
			`ALTER TABLE conversations ADD COLUMN profile TEXT NOT NULL DEFAULT ''`,
		}},
	}
	var currentVersion int
	s.db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&currentVersion)
	for _, m := range migrations {
		if currentVersion >= m.version {
			continue
		}
		for _, stmt := range m.statements {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, m.version); err != nil {
			return err
		}
	}

	return nil
}

func NewID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func (s *Store) CreateConversation(ctx context.Context, title, provider, model, profile string) (Conversation, error) {
	now := time.Now().UTC()
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New chat"
	}
	if len(title) > 120 {
		title = title[:120]
	}
	conversation := Conversation{
		ID:        NewID("conv"),
		Title:     title,
		Provider:  provider,
		Model:     model,
		Profile:   profile,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversations (id, title, provider, model, profile, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		conversation.ID, conversation.Title, conversation.Provider, conversation.Model, conversation.Profile, formatTime(now), formatTime(now))
	return conversation, err
}

func (s *Store) ListConversations(ctx context.Context, query string, limit int) ([]Conversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(query) != "" {
		pattern := "%" + strings.TrimSpace(query) + "%"
		rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.created_at, c.updated_at,
			COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
			FROM conversations c WHERE c.title LIKE ? ORDER BY c.updated_at DESC LIMIT ?`, pattern, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.created_at, c.updated_at,
			COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
			FROM conversations c ORDER BY c.updated_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conversations []Conversation
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, item)
	}
	return conversations, rows.Err()
}

func (s *Store) GetConversation(ctx context.Context, id string) (Conversation, []Message, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, provider, model, profile, created_at, updated_at, '' FROM conversations WHERE id = ?`, id)
	conversation, err := scanConversation(row)
	if err != nil {
		return Conversation{}, nil, err
	}
	messages, err := s.ListMessages(ctx, id, 0)
	return conversation, messages, err
}

func (s *Store) UpdateConversation(ctx context.Context, id, title, provider, model string) (Conversation, error) {
	current, _, err := s.GetConversation(ctx, id)
	if err != nil {
		return Conversation{}, err
	}
	if strings.TrimSpace(title) != "" {
		current.Title = strings.TrimSpace(title)
		if len(current.Title) > 120 {
			current.Title = current.Title[:120]
		}
	}
	if provider != "" {
		current.Provider = provider
	}
	if model != "" {
		current.Model = model
	}
	current.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `UPDATE conversations SET title = ?, provider = ?, model = ?, updated_at = ? WHERE id = ?`,
		current.Title, current.Provider, current.Model, formatTime(current.UpdatedAt), id)
	return current, err
}

func (s *Store) DeleteConversation(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// BeginTurn atomically creates the canonical user message and pending assistant.
func (s *Store) BeginTurn(ctx context.Context, conversationID, content string) (Message, Message, error) {
	now := time.Now().UTC()
	user := Message{ID: NewID("msg"), ConversationID: conversationID, Role: "user", Content: content, Status: "completed", CreatedAt: now}
	assistant := Message{ID: NewID("msg"), ConversationID: conversationID, Role: "assistant", Status: "pending", CreatedAt: now.Add(time.Millisecond)}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, Message{}, err
	}
	defer tx.Rollback()
	for _, m := range []Message{user, assistant} {
		if _, err = tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, role, content, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`, m.ID, m.ConversationID, m.Role, m.Content, m.Status, formatTime(m.CreatedAt)); err != nil {
			return Message{}, Message{}, err
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`, formatTime(now), conversationID)
	if err != nil {
		return Message{}, Message{}, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return Message{}, Message{}, fmt.Errorf("conversation update affected %d rows: %w", n, err)
	}
	if err = tx.Commit(); err != nil {
		return Message{}, Message{}, err
	}
	return user, assistant, nil
}

func (s *Store) AddMessage(ctx context.Context, conversationID, role, content, status, toolCallID string) (Message, error) {
	now := time.Now().UTC()
	message := Message{
		ID:             NewID("msg"),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		ToolCallID:     toolCallID,
		Status:         status,
		CreatedAt:      now,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, role, content, tool_call_id, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		message.ID, conversationID, role, content, nullString(toolCallID), status, formatTime(now)); err != nil {
		tx.Rollback()
		return Message{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`, formatTime(now), conversationID); err != nil {
		tx.Rollback()
		return Message{}, err
	}
	if err = tx.Commit(); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (s *Store) UpdateMessage(ctx context.Context, id, content, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE messages SET content = ?, status = ? WHERE id = ?`, content, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateMessageContent updates only the content of a message (for user editing).
func (s *Store) UpdateMessageContent(ctx context.Context, id, content string) (Message, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE messages SET content = ? WHERE id = ?`, content, id)
	if err != nil {
		return Message{}, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return Message{}, sql.ErrNoRows
	}
	var m Message
	var created string
	err = s.db.QueryRowContext(ctx, `SELECT id, conversation_id, role, content, COALESCE(tool_call_id,''), status, created_at FROM messages WHERE id = ?`, id).
		Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolCallID, &m.Status, &created)
	if err != nil {
		return Message{}, err
	}
	m.CreatedAt = parseTime(created)
	return m, nil
}

// DeleteMessage removes a message by ID, returning the conversation_id it belonged to.
func (s *Store) DeleteMessage(ctx context.Context, id string) (string, error) {
	var conversationID string
	err := s.db.QueryRowContext(ctx, `SELECT conversation_id FROM messages WHERE id = ?`, id).Scan(&conversationID)
	if err != nil {
		return "", err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, id)
	if err != nil {
		return "", err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return "", sql.ErrNoRows
	}
	return conversationID, nil
}

// FinishTurn commits terminal message state and its mandatory audit row together.
func (s *Store) FinishTurn(ctx context.Context, messageID, content, status string, log RequestLog) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE messages SET content = ?, status = ? WHERE id = ?`, content, status, messageID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return fmt.Errorf("terminal message update affected %d rows: %w", n, err)
	}
	if log.ID == "" {
		log.ID = NewID("req")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO request_logs (id, conversation_id, message_id, provider, model, endpoint, request_payload, response_payload, payload_truncated, http_status, prompt_tokens, completion_tokens, total_tokens, latency_ms, status, error_code, error_message, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, log.ID, nullString(log.ConversationID), nullString(log.MessageID), log.Provider, log.Model, log.Endpoint, nullString(log.RequestPayload), nullString(log.ResponsePayload), boolInt(log.PayloadTruncated), log.HTTPStatus, log.PromptTokens, log.CompletionTokens, log.TotalTokens, log.LatencyMS, log.Status, nullString(log.ErrorCode), nullString(log.ErrorMessage), formatTime(time.Now().UTC()))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SupersedeLatestAssistant(ctx context.Context, conversationID string) (Message, error) {
	var m Message
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id, conversation_id, role, content, COALESCE(tool_call_id,''), status, created_at FROM messages WHERE conversation_id=? AND role='assistant' AND status!='superseded' ORDER BY created_at DESC LIMIT 1`, conversationID).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolCallID, &m.Status, &created)
	if err != nil {
		return m, err
	}
	m.CreatedAt = parseTime(created)
	res, err := s.db.ExecContext(ctx, `UPDATE messages SET status='superseded' WHERE id=?`, m.ID)
	if err != nil {
		return m, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return m, fmt.Errorf("supersede affected %d rows: %w", n, err)
	}
	return m, nil
}

func (s *Store) ListMessages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	query := `SELECT id, conversation_id, role, content, COALESCE(tool_call_id, ''), status, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC, rowid ASC`
	args := []any{conversationID}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		var message Message
		var created string
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.Role, &message.Content, &message.ToolCallID, &message.Status, &created); err != nil {
			return nil, err
		}
		message.CreatedAt = parseTime(created)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *Store) InsertRequestLog(ctx context.Context, log RequestLog) error {
	if log.ID == "" {
		log.ID = NewID("req")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO request_logs (
		id, conversation_id, message_id, provider, model, endpoint, request_payload, response_payload,
		payload_truncated, http_status, prompt_tokens, completion_tokens, total_tokens, latency_ms,
		status, error_code, error_message, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, nullString(log.ConversationID), nullString(log.MessageID), log.Provider, log.Model, log.Endpoint,
		nullString(log.RequestPayload), nullString(log.ResponsePayload), boolInt(log.PayloadTruncated), log.HTTPStatus,
		log.PromptTokens, log.CompletionTokens, log.TotalTokens, log.LatencyMS, log.Status, nullString(log.ErrorCode),
		nullString(log.ErrorMessage), formatTime(time.Now().UTC()))
	return err
}

func (s *Store) CleanupRetention(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	_, err := s.db.ExecContext(ctx, `DELETE FROM request_logs WHERE created_at < ?`, formatTime(cutoff))
	return err
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

type conversationScanner interface {
	Scan(dest ...any) error
}

func scanConversation(row conversationScanner) (Conversation, error) {
	var item Conversation
	var created, updated string
	if err := row.Scan(&item.ID, &item.Title, &item.Provider, &item.Model, &item.Profile, &created, &updated, &item.Preview); err != nil {
		return Conversation{}, err
	}
	item.CreatedAt = parseTime(created)
	item.UpdatedAt = parseTime(updated)
	if len(item.Preview) > 160 {
		item.Preview = item.Preview[:160]
	}
	return item, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

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

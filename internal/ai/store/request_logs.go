package store

import (
	"context"
	"time"
)

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

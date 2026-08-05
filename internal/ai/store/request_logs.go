package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) InsertRequestLog(ctx context.Context, log RequestLog) error {
	if log.ID == "" {
		log.ID = NewID("req")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO request_logs (
		id, conversation_id, message_id, provider, model, endpoint, request_payload, response_payload,
		payload_truncated, http_status, prompt_tokens, completion_tokens, total_tokens, latency_ms,
		status, error_code, error_message, created_at, source, task_id, iteration
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, nullString(log.ConversationID), nullString(log.MessageID), log.Provider, log.Model, log.Endpoint,
		nullString(log.RequestPayload), nullString(log.ResponsePayload), boolInt(log.PayloadTruncated), log.HTTPStatus,
		log.PromptTokens, log.CompletionTokens, log.TotalTokens, log.LatencyMS, log.Status, nullString(log.ErrorCode),
		nullString(log.ErrorMessage), formatTime(time.Now().UTC()), defaultSource(log.Source), nullString(log.TaskID), log.Iteration)
	return err
}

func defaultSource(source string) string {
	if source == "" {
		return "chat"
	}
	return source
}

func (s *Store) InsertToolCall(ctx context.Context, call ToolCall) error {
	if call.ID == "" {
		call.ID = NewID("tool")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO tool_calls (id,request_id,message_id,tool_name,arguments,result,error_message,status,duration_ms,created_at,decision,payload_truncated) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, call.ID, call.RequestID, nullString(call.MessageID), call.ToolName, call.Arguments, nullString(call.Result), nullString(call.ErrorMessage), call.Status, call.DurationMS, formatTime(time.Now().UTC()), call.Decision, boolInt(call.PayloadTruncated))
	return err
}

func (s *Store) ListRequestLogs(ctx context.Context, f LogFilter) ([]RequestLog, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	q := `SELECT id,COALESCE(conversation_id,''),COALESCE(message_id,''),provider,model,endpoint,COALESCE(request_payload,''),COALESCE(response_payload,''),payload_truncated,http_status,prompt_tokens,completion_tokens,total_tokens,latency_ms,status,COALESCE(error_code,''),COALESCE(error_message,''),created_at,source,COALESCE(task_id,''),iteration FROM request_logs WHERE (?='' OR source=?) AND (?='' OR status=?) AND (?='' OR conversation_id=?) AND (?='' OR task_id=?) ORDER BY created_at DESC,rowid DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, f.Source, f.Source, f.Status, f.Status, f.ConversationID, f.ConversationID, f.TaskID, f.TaskID, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RequestLog, 0)
	for rows.Next() {
		var l RequestLog
		var created string
		if err := rows.Scan(&l.ID, &l.ConversationID, &l.MessageID, &l.Provider, &l.Model, &l.Endpoint, &l.RequestPayload, &l.ResponsePayload, &l.PayloadTruncated, &l.HTTPStatus, &l.PromptTokens, &l.CompletionTokens, &l.TotalTokens, &l.LatencyMS, &l.Status, &l.ErrorCode, &l.ErrorMessage, &created, &l.Source, &l.TaskID, &l.Iteration); err != nil {
			return nil, err
		}
		l.CreatedAt = parseTime(created)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		calls, err := s.listToolCalls(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ToolCalls = calls
	}
	return out, nil
}

func (s *Store) listToolCalls(ctx context.Context, requestID string) ([]ToolCall, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,request_id,COALESCE(message_id,''),tool_name,arguments,COALESCE(result,''),COALESCE(error_message,''),status,duration_ms,created_at,decision,payload_truncated FROM tool_calls WHERE request_id=? ORDER BY created_at,rowid`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ToolCall
	for rows.Next() {
		var c ToolCall
		var created string
		if err := rows.Scan(&c.ID, &c.RequestID, &c.MessageID, &c.ToolName, &c.Arguments, &c.Result, &c.ErrorMessage, &c.Status, &c.DurationMS, &created, &c.Decision, &c.PayloadTruncated); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTime(created)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Monitoring(ctx context.Context, hours int) (Monitoring, error) {
	if hours < 1 {
		hours = 24
	}
	if hours > 24*90 {
		hours = 24 * 90
	}
	m := Monitoring{WindowHours: hours}
	cutoff := formatTime(time.Now().UTC().Add(-time.Duration(hours) * time.Hour))
	var latest sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(status='error'),0),COALESCE(SUM(status='cancelled'),0),COALESCE(SUM(prompt_tokens),0),COALESCE(SUM(completion_tokens),0),COALESCE(SUM(total_tokens),0),COALESCE(AVG(latency_ms),0),COALESCE(MAX(latency_ms),0),MAX(created_at) FROM request_logs WHERE created_at>=?`, cutoff).Scan(&m.Requests, &m.Errors, &m.Cancellations, &m.PromptTokens, &m.CompletionTokens, &m.TotalTokens, &m.AverageLatencyMS, &m.MaxLatencyMS, &latest)
	if err != nil {
		return m, err
	}
	if latest.Valid {
		t := parseTime(latest.String)
		m.LatestActivity = &t
	}
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(tc.status='success'),0),COALESCE(SUM(tc.status IN ('error','cancelled')),0),COALESCE(SUM(tc.status='rejected' OR tc.decision IN ('rejected','unauthorized')),0) FROM tool_calls tc JOIN request_logs rl ON rl.id=tc.request_id WHERE rl.created_at>=?`, cutoff).Scan(&m.ToolSuccesses, &m.ToolErrors, &m.ToolRejections)
	return m, err
}

func (s *Store) CleanupRetention(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	_, err := s.db.ExecContext(ctx, `DELETE FROM request_logs WHERE created_at < ?`, formatTime(cutoff))
	return err

}

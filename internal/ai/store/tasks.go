package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Task statuses. Staleness is deliberately not a status: a task whose lease has
// expired is still 'running' in the table, and RecoverExpired transitions it
// directly to 'queued' or 'failed'. Keeping staleness implicit means there is no
// state that can be observed but never acted upon.
const (
	TaskQueued    = "queued"
	TaskRunning   = "running"
	TaskCompleted = "completed"
	TaskFailed    = "failed"
)

// ErrLeaseLost is returned when a mutating task statement matches no row because
// the task is no longer 'running' under this worker. A worker that sees it must
// abandon the task immediately without writing anything further — another worker
// already owns the task and any further write would corrupt its state.
var ErrLeaseLost = errors.New("task lease lost or reassigned")

// Task is one durable unit of agent work.
type Task struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Prompt         string `json:"prompt"`
	Status         string `json:"status"`

	Checkpoint []byte `json:"-"`
	Result     []byte `json:"-"`

	LeaseOwner    string    `json:"lease_owner,omitempty"`
	LeaseUntil    time.Time `json:"lease_until,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
	LastProgress  time.Time `json:"last_progress,omitempty"`
	AvailableAt   time.Time `json:"available_at"`

	Attempts    int    `json:"attempts"`
	MaxAttempts int    `json:"max_attempts"`
	LastError   string `json:"last_error,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewTask describes work to enqueue. It is also the shape a finished task uses
// to atomically queue its follow-up.
type NewTask struct {
	ConversationID string    `json:"conversation_id,omitempty"`
	Prompt         string    `json:"prompt"`
	MaxAttempts    int       `json:"max_attempts,omitempty"`
	AvailableAt    time.Time `json:"available_at,omitempty"`
}

const taskColumns = `id, conversation_id, prompt, status, checkpoint, result,
	lease_owner, lease_until, last_heartbeat, last_progress, available_at,
	attempts, max_attempts, last_error, created_at, completed_at`

const maxTaskPromptBytes = 512 * 1024

// MaxCheckpointBytes bounds a single serialized checkpoint. A checkpoint that
// outgrows this indicates the agent is accumulating context it should be
// summarizing instead, and an unbounded BLOB would eventually stall every write.
const MaxCheckpointBytes = 4 << 20

// EnqueueTask inserts a new queued task and returns its ID.
func (s *Store) EnqueueTask(ctx context.Context, t NewTask) (string, error) {
	prompt := strings.TrimSpace(t.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("task prompt is required")
	}
	if len(prompt) > maxTaskPromptBytes {
		return "", fmt.Errorf("task prompt exceeds %d bytes", maxTaskPromptBytes)
	}
	if t.MaxAttempts <= 0 {
		t.MaxAttempts = 3
	}
	now := time.Now().UTC()
	availableAt := t.AvailableAt
	if availableAt.IsZero() {
		availableAt = now
	}
	id := NewID("task")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_tasks (id, conversation_id, prompt, status, available_at, attempts, max_attempts, created_at)
		 VALUES (?, ?, ?, 'queued', ?, 0, ?, ?)`,
		id, nullString(t.ConversationID), prompt, formatTime(availableAt.UTC()), t.MaxAttempts, formatTime(now))
	if err != nil {
		return "", err
	}
	return id, nil
}

// ClaimTask atomically claims one queued, available task and grants workerID a
// lease. It returns (nil, nil) when nothing is claimable.
//
// The SELECT and UPDATE run in one transaction so two workers polling
// concurrently can never observe the same row as claimable.
func (s *Store) ClaimTask(ctx context.Context, workerID string, lease time.Duration) (*Task, error) {
	if strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("worker id is required")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("lease duration must be positive")
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM ai_tasks
		 WHERE status = 'queued' AND available_at <= ?
		 ORDER BY created_at ASC, rowid ASC LIMIT 1`, formatTime(now)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	leaseUntil := now.Add(lease)
	res, err := tx.ExecContext(ctx,
		`UPDATE ai_tasks
		 SET status = 'running', lease_owner = ?, lease_until = ?, last_heartbeat = ?, last_progress = ?
		 WHERE id = ? AND status = 'queued'`,
		workerID, formatTime(leaseUntil), formatTime(now), formatTime(now), id)
	if err != nil {
		return nil, err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return nil, nil
	}

	task, err := scanTask(tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM ai_tasks WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &task, nil
}

// HeartbeatTask extends the lease, but only while the task is still 'running'
// under workerID. Returns ErrLeaseLost once that stops being true.
func (s *Store) HeartbeatTask(ctx context.Context, taskID, workerID string, leaseUntil time.Time) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks SET lease_until = ?, last_heartbeat = ?
		 WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		formatTime(leaseUntil.UTC()), formatTime(now), taskID, workerID)
	if err != nil {
		return err
	}
	return requireOwnedRow(res)
}

// SaveTaskCheckpoint persists resumable agent state and marks forward progress.
// last_progress is bumped here and nowhere else: it is the only signal that
// distinguishes a working agent from a live worker driving a wedged agent.
func (s *Store) SaveTaskCheckpoint(ctx context.Context, taskID, workerID string, checkpoint []byte) error {
	if len(checkpoint) > MaxCheckpointBytes {
		return fmt.Errorf("checkpoint exceeds %d bytes", MaxCheckpointBytes)
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks SET checkpoint = ?, last_progress = ?
		 WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		checkpoint, formatTime(now), taskID, workerID)
	if err != nil {
		return err
	}
	return requireOwnedRow(res)
}

// RetryTask requeues a transiently failed task at retryAt, preserving the
// checkpoint so the next attempt resumes instead of restarting.
func (s *Store) RetryTask(ctx context.Context, taskID, workerID, reason string, retryAt time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks
		 SET status = 'queued', lease_owner = NULL, lease_until = NULL,
		     attempts = attempts + 1, available_at = ?, last_error = ?
		 WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		formatTime(retryAt.UTC()), nullString(truncateError(reason)), taskID, workerID)
	if err != nil {
		return err
	}
	return requireOwnedRow(res)
}

// FailTask marks a task terminally failed.
func (s *Store) FailTask(ctx context.Context, taskID, workerID, reason string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks
		 SET status = 'failed', lease_owner = NULL, lease_until = NULL,
		     attempts = attempts + 1, last_error = ?, completed_at = ?
		 WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		nullString(truncateError(reason)), formatTime(now), taskID, workerID)
	if err != nil {
		return err
	}
	return requireOwnedRow(res)
}

// CompleteTaskWithNext marks the task complete and enqueues its follow-up in a
// single transaction. Splitting these would let a crash in between lose the
// follow-up permanently, with the completed task giving no indication anything
// was dropped.
func (s *Store) CompleteTaskWithNext(ctx context.Context, taskID, workerID string, result []byte, next *NewTask) (string, error) {
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE ai_tasks
		 SET status = 'completed', result = ?, lease_owner = NULL, lease_until = NULL,
		     last_progress = ?, completed_at = ?, last_error = NULL
		 WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		result, formatTime(now), formatTime(now), taskID, workerID)
	if err != nil {
		return "", err
	}
	if err := requireOwnedRow(res); err != nil {
		return "", err
	}

	var nextID string
	if next != nil {
		prompt := strings.TrimSpace(next.Prompt)
		if prompt == "" {
			return "", fmt.Errorf("next task prompt is required")
		}
		if len(prompt) > maxTaskPromptBytes {
			return "", fmt.Errorf("next task prompt exceeds %d bytes", maxTaskPromptBytes)
		}
		maxAttempts := next.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
		availableAt := next.AvailableAt
		if availableAt.IsZero() {
			availableAt = now
		}
		nextID = NewID("task")
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ai_tasks (id, conversation_id, prompt, status, available_at, attempts, max_attempts, created_at)
			 VALUES (?, ?, ?, 'queued', ?, 0, ?, ?)`,
			nextID, nullString(next.ConversationID), prompt, formatTime(availableAt.UTC()), maxAttempts, formatTime(now)); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return nextID, nil
}

// RecoverExpiredTasks requeues running tasks whose lease has expired, failing
// those that have exhausted their attempts. The checkpoint is deliberately left
// intact so recovery is a resume, not a restart.
func (s *Store) RecoverExpiredTasks(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks
		 SET status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'queued' END,
		     lease_owner = NULL,
		     lease_until = NULL,
		     attempts = attempts + 1,
		     available_at = ?,
		     completed_at = CASE WHEN attempts + 1 >= max_attempts THEN ? ELSE NULL END,
		     last_error = 'worker lease expired'
		 WHERE status = 'running' AND lease_until IS NOT NULL AND lease_until < ?`,
		formatTime(now), formatTime(now), formatTime(now))
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// GetTask loads a single task by ID.
func (s *Store) GetTask(ctx context.Context, id string) (Task, error) {
	return scanTask(s.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM ai_tasks WHERE id = ?`, id))
}

// ListTasks returns tasks newest-first, optionally filtered by status.
func (s *Store) ListTasks(ctx context.Context, status string, limit int) ([]Task, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT ` + taskColumns + ` FROM ai_tasks`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC, rowid DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// CancelTask fails a task that has not started yet. Running tasks are not
// cancellable here: the worker owns the lease, so it must be asked to stop
// through the in-process registry instead of having the row yanked out from
// under it.
func (s *Store) CancelTask(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_tasks SET status = 'failed', last_error = 'cancelled by operator', completed_at = ?
		 WHERE id = ? AND status = 'queued'`, formatTime(now), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteTask removes a terminal task and its idempotency ledger.
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_tasks WHERE id = ? AND status IN ('completed','failed')`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// TaskCounts returns the number of tasks in each status, for status reporting.
func (s *Store) TaskCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM ai_tasks GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{TaskQueued: 0, TaskRunning: 0, TaskCompleted: 0, TaskFailed: 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// RecordToolCall stores a tool result under its idempotency key. It returns the
// previously stored result and true when the key was already recorded, which is
// how a resumed task avoids repeating a side effect that landed just before the
// crash that lost its checkpoint.
func (s *Store) RecordToolCall(ctx context.Context, taskID, idempotencyKey, result string) (string, bool, error) {
	var existing sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT result FROM ai_task_tool_calls WHERE task_id = ? AND idempotency_key = ?`,
		taskID, idempotencyKey).Scan(&existing)
	if err == nil {
		return existing.String, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_task_tool_calls (task_id, idempotency_key, result, created_at) VALUES (?, ?, ?, ?)`,
		taskID, idempotencyKey, nullString(result), formatTime(time.Now().UTC())); err != nil {
		return "", false, err
	}
	return result, false, nil
}

// LookupToolCall reports whether a tool call with this key already executed.
func (s *Store) LookupToolCall(ctx context.Context, taskID, idempotencyKey string) (string, bool, error) {
	var result sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT result FROM ai_task_tool_calls WHERE task_id = ? AND idempotency_key = ?`,
		taskID, idempotencyKey).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return result.String, true, nil
}

// CleanupTasks deletes terminal tasks older than retentionDays.
func (s *Store) CleanupTasks(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM ai_tasks WHERE status IN ('completed','failed') AND created_at < ?`, formatTime(cutoff))
	return err
}

func requireOwnedRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrLeaseLost
	}
	return nil
}

func truncateError(reason string) string {
	const maxErrorBytes = 2000
	if len(reason) > maxErrorBytes {
		return reason[:maxErrorBytes] + "..."
	}
	return reason
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (Task, error) {
	var t Task
	var conversationID, leaseOwner, lastError sql.NullString
	var leaseUntil, lastHeartbeat, lastProgress, completedAt sql.NullString
	var availableAt, createdAt string
	if err := row.Scan(
		&t.ID, &conversationID, &t.Prompt, &t.Status, &t.Checkpoint, &t.Result,
		&leaseOwner, &leaseUntil, &lastHeartbeat, &lastProgress, &availableAt,
		&t.Attempts, &t.MaxAttempts, &lastError, &createdAt, &completedAt,
	); err != nil {
		return Task{}, err
	}
	t.ConversationID = conversationID.String
	t.LeaseOwner = leaseOwner.String
	t.LastError = lastError.String
	t.AvailableAt = parseTime(availableAt)
	t.CreatedAt = parseTime(createdAt)
	if leaseUntil.Valid {
		t.LeaseUntil = parseTime(leaseUntil.String)
	}
	if lastHeartbeat.Valid {
		t.LastHeartbeat = parseTime(lastHeartbeat.String)
	}
	if lastProgress.Valid {
		t.LastProgress = parseTime(lastProgress.String)
	}
	if completedAt.Valid {
		parsed := parseTime(completedAt.String)
		t.CompletedAt = &parsed
	}
	return t, nil
}

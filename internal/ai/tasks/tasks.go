// Package tasks implements the durable task state machine that lets local agent
// work survive crashes, hangs, and restarts.
//
// Three signals are tracked independently, because each detects a different
// failure:
//
//	heartbeat/lease  is the worker process alive?      crash, kill, disconnect
//	checkpoint       where should the work resume?     lost work on restart
//	last progress    is the agent actually doing work? live worker, stuck agent
//
// A heartbeat goroutine stays perfectly healthy while the agent it supervises is
// wedged forever, which is why last_progress exists separately from
// last_heartbeat and why NoProgressSeconds is enforced by the worker loop.
//
// Files:
//   - tasks.go       shared types, retry classification, timing helpers
//   - checkpoint.go  conversation checkpoint serialization and resume
//   - agent.go       the chat-backed Agent implementation
//   - worker.go      claim → step → checkpoint → complete loop with heartbeat
//   - watchdog.go    requeues tasks whose lease expired
//   - scheduler.go   worker pool lifecycle and graceful shutdown
package tasks

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/store"
)

// Config is the runner's configuration. It is the operator-facing config type
// so there is exactly one place where the timing invariants are defined.
type Config = aiconfig.TasksConfig

// maxRetryBackoffShift caps the exponential backoff at 2^6 seconds plus jitter.
const maxRetryBackoffShift = 6

// finalizeTimeout bounds the fresh context used for terminal DB writes after the
// task context has been cancelled. An unbounded context.Background() here would
// let a shutdown hang on a wedged database.
const finalizeTimeout = 10 * time.Second

// StepResult is what one bounded agent step produces.
type StepResult struct {
	// Checkpoint is the serialized resumable state after this step. Empty means
	// the step made no durable progress.
	Checkpoint []byte
	// Output is the final result payload, only meaningful when Done.
	Output []byte
	// Done reports that the task reached a terminal success state.
	Done bool
	// NextTask, when set on a Done step, is queued atomically with completion.
	NextTask *store.NewTask
}

// Agent executes exactly one bounded step per call: one model call and the tool
// calls it requests. Keeping steps bounded is what makes checkpointing possible
// — an agent that runs to completion in one call has no intermediate state to
// persist and therefore nothing to resume from.
//
// The agent marks progress as the step proceeds so a stall inside a single step
// is visible to the worker before StepTimeout fires.
type Agent interface {
	RunStep(ctx context.Context, task *store.Task, checkpoint *ConversationCheckpoint, progress *Progress) (StepResult, error)
}

// Progress is the fine-grained liveness signal. The lease heartbeat only proves
// the worker process is alive; it keeps renewing perfectly while the generation
// it supervises has stalled mid-stream with the connection still open. Marking
// on every stream chunk is what turns coarse per-request liveness into something
// NoProgressWindow can actually act on.
type Progress struct {
	mu   sync.Mutex
	last time.Time
}

// NewProgress starts a tracker at the current time.
func NewProgress() *Progress {
	return &Progress{last: time.Now()}
}

// Mark records forward progress.
func (p *Progress) Mark() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.last = time.Now()
	p.mu.Unlock()
}

// Since reports how long the tracker has gone without a mark.
func (p *Progress) Since() time.Duration {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Since(p.last)
}

// sleepContext waits for d or until ctx is done, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// isTransient classifies errors worth retrying. Timeouts and network faults are
// transient; a blocked agent or an exhausted step budget is not, because the
// retry would hit the same wall.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrBlocked) || errors.Is(err, ErrMaxSteps) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, ErrNoProgress) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"rate_limited", "rate limit", "timeout", "timed out",
		"connection reset", "connection refused", "broken pipe",
		"temporarily unavailable", "provider_timeout", "502", "503", "504",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

// retryBackoff is exponential with jitter. Without jitter, a batch of tasks that
// fail together (a provider outage) retries in lockstep and re-creates the
// thundering herd that caused the outage.
func retryBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > maxRetryBackoffShift {
		attempt = maxRetryBackoffShift
	}
	base := time.Second * time.Duration(1<<attempt)
	return base + time.Duration(rand.Int63n(int64(base/2)+1))
}

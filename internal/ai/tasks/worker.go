package tasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"browser-server/internal/ai/store"
)

// ErrNoProgress is raised by the worker when a task holds a healthy lease but
// has not produced a checkpoint within MaxNoProgress. The heartbeat goroutine
// cannot detect this: it keeps renewing the lease perfectly while the agent it
// supervises is wedged.
var ErrNoProgress = errors.New("task made no progress within the allowed window")

// ErrMaxSteps is raised when a task exhausts its step budget. It is deliberately
// not transient — the same budget would be exhausted again on retry.
var ErrMaxSteps = errors.New("task exceeded its maximum step count")

// Worker claims one task at a time and drives it to a terminal state.
type Worker struct {
	ID     string
	Store  *store.Store
	Agent  Agent
	Config Config
}

// Run polls for claimable work until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		task, err := w.Store.ClaimTask(ctx, w.ID, w.Config.LeaseDuration())
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// A claim failure is almost always a transient database contention
			// error. Backing off and retrying is preferable to killing the
			// worker, which would silently shrink the pool for the process's
			// remaining lifetime.
			log.Printf("AI tasks: worker %s claim failed: %v", w.ID, err)
			if sleepErr := sleepContext(ctx, w.Config.IdleDelay()); sleepErr != nil {
				return sleepErr
			}
			continue
		}
		if task == nil {
			if sleepErr := sleepContext(ctx, w.Config.IdleDelay()); sleepErr != nil {
				return sleepErr
			}
			continue
		}
		w.processTask(ctx, task)
	}
}

// processTask runs the claimed task's step loop under an independent heartbeat.
func (w *Worker) processTask(parent context.Context, task *store.Task) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Buffered so the heartbeat goroutine never blocks on a step loop that has
	// already returned.
	leaseLost := make(chan struct{}, 1)
	go w.heartbeat(ctx, cancel, task.ID, leaseLost)

	checkpoint, err := LoadCheckpoint(task)
	if err != nil {
		// An unreadable checkpoint cannot be retried into readability.
		w.finalize(task, fmt.Errorf("load checkpoint: %w", err), false)
		return
	}

	lastProgress := time.Now()
	progress := NewProgress()
	for step := checkpoint.Step; ; step++ {
		if step >= w.Config.MaxSteps {
			cancel()
			w.finalize(task, fmt.Errorf("%w (%d steps)", ErrMaxSteps, w.Config.MaxSteps), false)
			return
		}
		if time.Since(lastProgress) > w.Config.NoProgressWindow() {
			cancel()
			w.finalize(task, ErrNoProgress, true)
			return
		}
		if ctx.Err() != nil {
			// Either shutdown or a lost lease. Both leave the task resumable:
			// the checkpoint is already durable and the watchdog will requeue.
			return
		}

		checkpoint.Step = step
		stepCtx, stepCancel := context.WithTimeout(ctx, w.Config.StepTimeout())
		progress.Mark()
		stallDone := make(chan struct{})
		go w.watchStall(stepCtx, stepCancel, progress, stallDone)
		result, stepErr := w.Agent.RunStep(stepCtx, task, checkpoint, progress)
		stepCancel()
		<-stallDone

		if stepErr != nil {
			cancel()
			if errors.Is(stepErr, store.ErrLeaseLost) {
				// Another worker owns the task now; writing anything further
				// would corrupt its state.
				return
			}
			if parent.Err() != nil {
				// Shutdown cancelled the step. Leave the lease to expire so the
				// watchdog resumes from the checkpoint instead of burning an
				// attempt on an interruption the task did not cause.
				return
			}
			if progress.Since() > w.Config.NoProgressWindow() {
				// The step was killed by the stall watcher, so the deadline
				// error it reports is a symptom rather than the cause.
				stepErr = ErrNoProgress
			}
			w.finalize(task, stepErr, isTransient(stepErr))
			return
		}

		if len(result.Checkpoint) > 0 {
			if err := w.Store.SaveTaskCheckpoint(ctx, task.ID, w.ID, result.Checkpoint); err != nil {
				cancel()
				if !errors.Is(err, store.ErrLeaseLost) {
					log.Printf("AI tasks: worker %s failed to checkpoint %s: %v", w.ID, task.ID, err)
				}
				return
			}
			// The next step must read the state that was just persisted, not
			// the state the task was claimed with.
			task.Checkpoint = result.Checkpoint
			lastProgress = time.Now()
		}

		if result.Done {
			cancel()
			finishCtx, finishCancel := context.WithTimeout(context.Background(), finalizeTimeout)
			nextID, err := w.Store.CompleteTaskWithNext(finishCtx, task.ID, w.ID, result.Output, result.NextTask)
			finishCancel()
			if err != nil {
				if !errors.Is(err, store.ErrLeaseLost) {
					log.Printf("AI tasks: worker %s failed to complete %s: %v", w.ID, task.ID, err)
				}
				return
			}
			if nextID != "" {
				log.Printf("AI tasks: %s completed and queued follow-up %s", task.ID, nextID)
			}
			return
		}

		select {
		case <-leaseLost:
			// Another worker will resume from the checkpoint.
			return
		default:
		}
	}
}

// watchStall kills a step that has stopped marking progress. StepTimeout alone
// is not enough: a step legitimately allowed 120s of model time gives a stream
// that stalls at second 1 another 119 seconds of nothing. Sampling the progress
// tracker catches the stall as soon as the window elapses.
func (w *Worker) watchStall(ctx context.Context, cancel context.CancelFunc, progress *Progress, done chan<- struct{}) {
	defer close(done)
	window := w.Config.NoProgressWindow()
	ticker := time.NewTicker(w.Config.HeartbeatInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if progress.Since() > window {
				log.Printf("AI tasks: worker %s aborting stalled step after %s without progress", w.ID, window)
				cancel()
				return
			}
		}
	}
}

// heartbeat renews the lease on its own schedule so a slow model call never
// looks like a dead worker. Losing the lease cancels the task context, which
// aborts the in-flight step.
func (w *Worker) heartbeat(ctx context.Context, cancel context.CancelFunc, taskID string, lost chan<- struct{}) {
	ticker := time.NewTicker(w.Config.HeartbeatInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := w.Store.HeartbeatTask(ctx, taskID, w.ID, time.Now().Add(w.Config.LeaseDuration()))
			if err == nil {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, store.ErrLeaseLost) {
				log.Printf("AI tasks: worker %s lost the lease on %s", w.ID, taskID)
			} else {
				log.Printf("AI tasks: worker %s heartbeat failed for %s: %v", w.ID, taskID, err)
			}
			select {
			case lost <- struct{}{}:
			default:
			}
			cancel()
			return
		}
	}
}

// finalize writes the terminal outcome under a fresh bounded context. The task
// context is already cancelled by this point, and an unbounded background
// context would let shutdown hang on a wedged database.
func (w *Worker) finalize(task *store.Task, cause error, transient bool) {
	ctx, cancel := context.WithTimeout(context.Background(), finalizeTimeout)
	defer cancel()

	reason := cause.Error()
	if transient && task.Attempts+1 < task.MaxAttempts {
		retryAt := time.Now().Add(retryBackoff(task.Attempts))
		if err := w.Store.RetryTask(ctx, task.ID, w.ID, reason, retryAt); err != nil && !errors.Is(err, store.ErrLeaseLost) {
			log.Printf("AI tasks: worker %s failed to requeue %s: %v", w.ID, task.ID, err)
		}
		return
	}
	if err := w.Store.FailTask(ctx, task.ID, w.ID, reason); err != nil && !errors.Is(err, store.ErrLeaseLost) {
		log.Printf("AI tasks: worker %s failed to mark %s failed: %v", w.ID, task.ID, err)
	}
}

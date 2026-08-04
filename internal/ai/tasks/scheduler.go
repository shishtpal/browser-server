package tasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"browser-server/internal/ai/store"
)

// Runner owns the worker pool, the watchdog, and the retention sweep. It is the
// only entry point the rest of the server uses to submit and inspect tasks.
type Runner struct {
	cfg    Config
	store  *store.Store
	agent  Agent
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	started bool
}

// NewRunner builds a runner from an already-normalized config.
func NewRunner(cfg Config, st *store.Store, agent Agent) *Runner {
	cfg.Normalize()
	return &Runner{cfg: cfg, store: st, agent: agent}
}

// Enabled reports whether task execution is configured to run.
func (r *Runner) Enabled() bool {
	return r != nil && r.cfg.Enabled && r.store != nil && r.agent != nil
}

// Config exposes the effective (normalized) configuration.
func (r *Runner) Config() Config {
	if r == nil {
		return Config{}
	}
	return r.cfg
}

// Start launches the worker pool, the watchdog, and the retention sweep. It
// returns immediately; Stop drains everything.
func (r *Runner) Start() {
	if !r.Enabled() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.started = true

	// Recover leases orphaned by the previous process before any worker starts
	// polling, so a restart resumes in-flight work instead of waiting a full
	// watchdog interval for it.
	recoverCtx, recoverCancel := context.WithTimeout(ctx, finalizeTimeout)
	if count, err := r.store.RecoverExpiredTasks(recoverCtx); err != nil {
		log.Printf("AI tasks: startup recovery failed: %v", err)
	} else if count > 0 {
		log.Printf("AI tasks: startup recovery requeued %d task(s)", count)
	}
	recoverCancel()

	prefix := workerPrefix()
	for i := 0; i < r.cfg.MaxConcurrent; i++ {
		worker := &Worker{
			ID:     fmt.Sprintf("%s-%d", prefix, i),
			Store:  r.store,
			Agent:  r.agent,
			Config: r.cfg,
		}
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("AI tasks: worker %s exited: %v", worker.ID, err)
			}
		}()
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if err := RunWatchdog(ctx, r.store, r.cfg.WatchdogInterval()); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("AI tasks: watchdog exited: %v", err)
		}
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.runRetention(ctx)
	}()

	log.Printf("AI tasks: runner started with %d worker(s), lease %ds, heartbeat %ds",
		r.cfg.MaxConcurrent, r.cfg.LeaseSeconds, r.cfg.HeartbeatSeconds)
}

// Stop cancels every worker and waits for in-flight steps to unwind. Tasks
// interrupted this way keep their checkpoints and are resumed after restart.
func (r *Runner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.started = false
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()
}

func (r *Runner) runRetention(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		if err := r.store.CleanupTasks(ctx, r.cfg.RetentionDays); err != nil && ctx.Err() == nil {
			log.Printf("AI tasks: retention cleanup failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Enqueue submits a new task.
func (r *Runner) Enqueue(ctx context.Context, task store.NewTask) (string, error) {
	if r == nil || r.store == nil {
		return "", errors.New("task runner is unavailable")
	}
	if task.MaxAttempts <= 0 {
		task.MaxAttempts = r.cfg.MaxAttempts
	}
	return r.store.EnqueueTask(ctx, task)
}

// Get returns a single task.
func (r *Runner) Get(ctx context.Context, id string) (store.Task, error) {
	return r.store.GetTask(ctx, id)
}

// List returns tasks newest-first, optionally filtered by status.
func (r *Runner) List(ctx context.Context, status string, limit int) ([]store.Task, error) {
	return r.store.ListTasks(ctx, status, limit)
}

// Cancel fails a task that has not started yet.
func (r *Runner) Cancel(ctx context.Context, id string) error {
	return r.store.CancelTask(ctx, id)
}

// Delete removes a terminal task.
func (r *Runner) Delete(ctx context.Context, id string) error {
	return r.store.DeleteTask(ctx, id)
}

// Counts returns queue depth by status.
func (r *Runner) Counts(ctx context.Context) (map[string]int, error) {
	return r.store.TaskCounts(ctx)
}

// workerPrefix identifies this process in lease_owner. Including the host and
// PID means a lease held by a previous process is visibly distinct from one held
// by this process, which makes an unexpected recovery traceable to its owner.
func workerPrefix() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "host"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

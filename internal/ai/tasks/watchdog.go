package tasks

import (
	"context"
	"log"
	"time"

	"browser-server/internal/ai/store"
)

// RunWatchdog sweeps tasks whose lease expired back into the queue. It is the
// only component permitted to touch a task it does not own, and it does so
// solely through the lease_until < now predicate.
func RunWatchdog(ctx context.Context, st *store.Store, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			count, err := st.RecoverExpiredTasks(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("AI tasks: watchdog sweep failed: %v", err)
				continue
			}
			if count > 0 {
				log.Printf("AI tasks: watchdog requeued %d task(s) with expired leases", count)
			}
		}
	}
}

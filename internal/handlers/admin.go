package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"browser-server/internal/auth"
)

// AdminShutdown is installed by main so managed restarts can close long-lived
// AI/MCP resources before exiting. The handler enforces a short deadline.
var AdminShutdown func()

type adminStatusResponse struct {
	Managed         bool    `json:"managed"`
	AdminConfigured bool    `json:"admin_configured"`
	UptimeSeconds   float64 `json:"uptime_seconds"`
}

// AdminStatus reports whether a supervisor explicitly opted the process into
// self-restart. The environment flag is intentionally conservative: without
// it, exiting would strand a server started from a terminal.
func AdminStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminStatusResponse{
		Managed:         os.Getenv("BS_MANAGED") == "1",
		AdminConfigured: auth.AdminConfigured(),
		UptimeSeconds:   time.Since(StartedAt).Seconds(),
	})
}

// AdminShutdownOnce guards the deferred AdminShutdown so a slow flush cannot
// overlap a second /api/admin/restart request before the process exits.
var AdminShutdownOnce sync.Once

// AdminRestart acknowledges the request before terminating. NSSM is configured
// to restart a cleanly exited process. A bounded shutdown hook closes long-lived
// AI/MCP resources because os.Exit itself does not run deferred cleanup.
func AdminRestart(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("BS_MANAGED") != "1" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "restart_unsupported",
			"message": "Server restart requires a managed service with BS_MANAGED=1.",
		})
		return
	}

	shutdownStarted := false
	AdminShutdownOnce.Do(func() { shutdownStarted = true })
	if !shutdownStarted {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "restart_in_progress",
			"message": "A server restart is already in progress.",
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]bool{"restarting": true})
	time.AfterFunc(500*time.Millisecond, func() {
		if AdminShutdown != nil {
			done := make(chan struct{})
			go func() {
				AdminShutdown()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}
		os.Exit(0)
	})
}

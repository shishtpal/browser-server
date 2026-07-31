package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (m *Module) requireAI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.cfg == nil || !m.cfg.Enabled || m.store == nil || m.service == nil {
			writeError(w, http.StatusServiceUnavailable, "ai_disabled", "AI is disabled. Create bs-ai-config.json and bs-ai-models.json and restart the server.")
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func writeSSE(w http.ResponseWriter, event string, value any) {
	payload, _ := json.Marshal(value)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

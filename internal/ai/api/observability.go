package api

import (
	"net/http"
	"strconv"

	"browser-server/internal/ai/store"
)

func (m *Module) Logs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	logs, err := m.store.ListRequestLogs(r.Context(), store.LogFilter{Source: q.Get("source"), Status: q.Get("status"), ConversationID: q.Get("conversation_id"), TaskID: q.Get("task_id"), Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "logs_failed", "Failed to load AI logs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs, "limit": boundedLimit(limit), "offset": max(offset, 0)})
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (m *Module) Monitoring(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("window_hours"))
	metrics, err := m.store.Monitoring(r.Context(), hours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "monitoring_failed", "Failed to load AI monitoring data.")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

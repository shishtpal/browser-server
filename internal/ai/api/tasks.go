package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"browser-server/internal/ai/store"

	"github.com/gorilla/mux"
)

type createTaskRequest struct {
	Prompt         string `json:"prompt"`
	ConversationID string `json:"conversation_id,omitempty"`
	MaxAttempts    int    `json:"max_attempts,omitempty"`
}

type createTaskResponse struct {
	TaskID string `json:"task_id"`
}

// taskView is the wire shape of a task. The stored checkpoint is never exposed:
// it is an internal resume artifact that can hold the full conversation, and the
// result is surfaced separately in decoded form.
type taskView struct {
	store.Task
	Result        json.RawMessage `json:"result,omitempty"`
	Stale         bool            `json:"stale"`
	HasCheckpoint bool            `json:"has_checkpoint"`
}

type taskStatusResponse struct {
	Enabled bool           `json:"enabled"`
	Workers int            `json:"workers"`
	Counts  map[string]int `json:"counts"`
}

func (m *Module) requireTasks(next http.HandlerFunc) http.HandlerFunc {
	return m.requireAI(func(w http.ResponseWriter, r *http.Request) {
		if m.tasks == nil || !m.tasks.Enabled() {
			writeError(w, http.StatusServiceUnavailable, "tasks_disabled",
				"Background tasks are disabled. Set tasks.enabled in bs-ai-config.json and restart the server.")
			return
		}
		next(w, r)
	})
}

func (m *Module) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "prompt is required")
		return
	}
	if req.ConversationID != "" {
		if _, _, err := m.store.GetConversation(r.Context(), req.ConversationID); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Unknown conversation")
			return
		}
	}
	id, err := m.tasks.Enqueue(r.Context(), store.NewTask{
		ConversationID: req.ConversationID,
		Prompt:         req.Prompt,
		MaxAttempts:    req.MaxAttempts,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, createTaskResponse{TaskID: id})
}

func (m *Module) ListTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", store.TaskQueued, store.TaskRunning, store.TaskCompleted, store.TaskFailed:
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "Unknown task status filter")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	tasks, err := m.tasks.List(r.Context(), status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to list tasks")
		return
	}
	out := make([]taskView, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, newTaskView(task))
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) GetTask(w http.ResponseWriter, r *http.Request) {
	task, err := m.tasks.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to load task")
		return
	}
	writeJSON(w, http.StatusOK, newTaskView(task))
}

// CancelTask cancels a queued task. A running task is not cancelled here: its
// worker holds the lease, and yanking the row out from under it would leave the
// in-flight step writing to a task it no longer owns. Stopping the server (or
// letting the lease expire) is the supported way to interrupt running work.
func (m *Module) CancelTask(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	err := m.tasks.Cancel(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		task, getErr := m.tasks.Get(r.Context(), id)
		if errors.Is(getErr, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		if getErr != nil {
			writeError(w, http.StatusInternalServerError, "store_error", "Failed to load task")
			return
		}
		writeError(w, http.StatusConflict, "task_not_cancellable",
			"Only queued tasks can be cancelled; this task is "+task.Status)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to cancel task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	err := m.tasks.Delete(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		task, getErr := m.tasks.Get(r.Context(), id)
		if errors.Is(getErr, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		if getErr != nil {
			writeError(w, http.StatusInternalServerError, "store_error", "Failed to load task")
			return
		}
		writeError(w, http.StatusConflict, "task_not_terminal",
			"Only completed or failed tasks can be deleted; this task is "+task.Status)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) TaskStatus(w http.ResponseWriter, r *http.Request) {
	resp := taskStatusResponse{Counts: map[string]int{}}
	if m.tasks != nil && m.tasks.Enabled() {
		resp.Enabled = true
		resp.Workers = m.tasks.Config().MaxConcurrent
		counts, err := m.tasks.Counts(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store_error", "Failed to read task counts")
			return
		}
		resp.Counts = counts
	}
	writeJSON(w, http.StatusOK, resp)
}

// newTaskView derives the client-facing view. Staleness is computed here rather
// than stored: a task whose lease has expired is still 'running' in the table
// until the watchdog sweeps it, and persisting a 'stale' status would create a
// state that can be observed but never acted upon.
func newTaskView(task store.Task) taskView {
	view := taskView{Task: task, HasCheckpoint: len(task.Checkpoint) > 0}
	if task.Status == store.TaskRunning && !task.LeaseUntil.IsZero() {
		view.Stale = task.LeaseUntil.Before(time.Now().UTC())
	}
	if len(task.Result) > 0 && json.Valid(task.Result) {
		view.Result = json.RawMessage(task.Result)
	}
	view.Task.Checkpoint = nil
	view.Task.Result = nil
	return view
}

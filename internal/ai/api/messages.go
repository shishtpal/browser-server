package api

import (
	"browser-server/internal/ai/chat"
	"browser-server/internal/ai/store"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type toolDecisionRequest struct {
	Approved *bool  `json:"approved"`
	Comment  string `json:"comment,omitempty"`
}

func (m *Module) AppendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" || len(content) > 512*1024 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Message content is required and must not exceed 524288 bytes")
		return
	}
	message, err := m.service.AppendMessage(r.Context(), mux.Vars(r)["id"], content)
	if err != nil {
		switch {
		case errors.Is(err, chat.ErrAppendWindowClosed):
			writeError(w, http.StatusConflict, "append_window_closed", "No tool call is accepting appended context")
		case errors.Is(err, chat.ErrAppendByteLimit):
			writeError(w, http.StatusRequestEntityTooLarge, "append_window_limit", "Append window byte limit reached")
		case errors.Is(err, chat.ErrAppendMessageLimit):
			writeError(w, http.StatusTooManyRequests, "append_window_limit", "Append window message limit reached")
		case store.IsNotFound(err):
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
		default:
			writeError(w, http.StatusInternalServerError, "store_error", "Failed to append message")
		}
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (m *Module) SubmitMessage(w http.ResponseWriter, r *http.Request) {
	var req chat.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	stream := m.cfg.Chat.Stream
	if req.Stream != nil {
		stream = *req.Stream
	}
	if stream {
		m.submitMessageSSE(w, r, req)
		return
	}
	result, err := m.service.Submit(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		m.writeSubmitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (m *Module) submitMessageSSE(w http.ResponseWriter, r *http.Request, req chat.SubmitRequest) {
	if m.service.IsActive(mux.Vars(r)["id"]) {
		m.writeSubmitError(w, chat.ErrConflict)
		return
	}
	if strings.TrimSpace(req.Content) == "" && len(req.AttachmentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Message content is required")
		return
	}
	if len(strings.TrimSpace(req.Content)) > 512*1024 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Message content must not exceed 524288 bytes")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")
	flusher, _ := w.(http.Flusher)
	// Flush immediately to send the 200 + headers to the client so the
	// browser's fetch() resolves and begins reading the stream.  Without
	// this, the request appears "stuck" until the first SSE event arrives
	// from the LLM provider (which can take several seconds).
	if flusher != nil {
		flusher.Flush()
	}
	result, err := m.service.SubmitStream(r.Context(), mux.Vars(r)["id"], req, func(event chat.Event) error {
		writeSSE(w, event.Type, event)
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	if err != nil {
		writeSSE(w, "error", apiError{Code: submitErrorCode(err), Message: safeSubmitMessage(err)})
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
	writeSSE(w, "done", map[string]any{
		"conversation_id": result.ConversationID,
		"message_id":      result.AssistantMessage.ID,
		"status":          result.AssistantMessage.Status,
		"usage":           result.Usage,
	})
	if flusher != nil {
		flusher.Flush()
	}
}

func (m *Module) Regenerate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if m.service.IsActive(id) {
		writeError(w, 409, "generation_conflict", "Generation is active")
		return
	}
	result, err := m.service.Regenerate(r.Context(), id)
	if err != nil {
		m.writeSubmitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (m *Module) StopGeneration(w http.ResponseWriter, r *http.Request) {
	stopped := m.service.Stop(mux.Vars(r)["id"])
	writeJSON(w, http.StatusOK, map[string]bool{"stopped": stopped})
}

func (m *Module) DecideToolCall(w http.ResponseWriter, r *http.Request) {
	var req toolDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON")
		return
	}
	if req.Approved == nil && strings.TrimSpace(req.Comment) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Either approved or comment must be provided")
		return
	}
	vars := mux.Vars(r)
	approved := req.Approved != nil && *req.Approved
	if err := m.service.DecideToolCall(vars["id"], vars["callID"], approved, strings.TrimSpace(req.Comment)); err != nil {
		writeError(w, http.StatusConflict, "tool_call_not_pending", "Tool call is no longer pending approval")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (m *Module) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convID := vars["id"]
	msgID := vars["msgId"]
	if m.service.IsActive(convID) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	var req struct {
		Content *string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if req.Content == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Content field is required")
		return
	}
	msg, err := m.store.UpdateMessageContent(r.Context(), msgID, *req.Content)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to update message")
		return
	}
	if msg.ConversationID != convID {
		writeError(w, http.StatusNotFound, "not_found", "Message not found in this conversation")
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (m *Module) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convID := vars["id"]
	msgID := vars["msgId"]
	if m.service.IsActive(convID) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	ownerConvID, err := m.store.DeleteMessage(r.Context(), msgID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to delete message")
		return
	}
	if ownerConvID != convID {
		writeError(w, http.StatusNotFound, "not_found", "Message not found in this conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

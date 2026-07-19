package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/store"
)

type Module struct {
	cfg     *aiconfig.Config
	store   *store.Store
	service *chat.Service
	stop    chan struct{}
	wg      sync.WaitGroup
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createConversationRequest struct {
	Title    string `json:"title"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type updateConversationRequest struct {
	Title    string `json:"title"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type conversationDetail struct {
	Conversation store.Conversation `json:"conversation"`
	Messages     []store.Message    `json:"messages"`
}

func Init() (*Module, error) {
	cfg, err := aiconfig.Load()
	if err != nil {
		return nil, err
	}
	module := &Module{cfg: cfg}
	if !cfg.Enabled {
		log.Printf("AI disabled: no config found at %s", cfg.Path)
		return module, nil
	}
	dbPath := cfg.ResolvePath(cfg.Logging.DBPath)
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init AI store: %w", err)
	}
	if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
		st.Close()
		return nil, fmt.Errorf("AI retention cleanup: %w", err)
	}
	module.store = st
	module.service = chat.NewService(cfg, st)
	module.stop = make(chan struct{})
	module.wg.Add(1)
	go func() {
		defer module.wg.Done()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
					log.Printf("AI retention cleanup failed: %v", err)
				}
			case <-module.stop:
				return
			}
		}
	}()
	log.Printf("AI enabled with %d provider(s); store: %s", len(cfg.Providers), dbPath)
	return module, nil
}

func (m *Module) Close() error {
	if m == nil || m.store == nil {
		return nil
	}
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.service.Close()
	m.wg.Wait()
	return m.store.Close()
}

func (m *Module) Register(r *mux.Router) {
	r.HandleFunc("/ai/config", m.Config).Methods("GET")
	r.HandleFunc("/ai/conversations", m.requireAI(m.ListConversations)).Methods("GET")
	r.HandleFunc("/ai/conversations", m.requireAI(m.CreateConversation)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.GetConversation)).Methods("GET")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.UpdateConversation)).Methods("PATCH")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.DeleteConversation)).Methods("DELETE")
	r.HandleFunc("/ai/conversations/{id}/messages", m.requireAI(m.SubmitMessage)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/stop", m.requireAI(m.StopGeneration)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/regenerate", m.requireAI(m.Regenerate)).Methods("POST")
}

func (m *Module) Config(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, m.cfg.Sanitized())
}

func (m *Module) ListConversations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	conversations, err := m.store.ListConversations(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (m *Module) CreateConversation(w http.ResponseWriter, r *http.Request) {
	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	providerName := req.Provider
	modelID := req.Model
	if providerName == "" && modelID == "" {
		providerName, modelID = m.service.DefaultSelection()
	} else if providerName == "" {
		providerName = m.cfg.DefaultProvider
	} else if modelID == "" {
		model, ok := m.cfg.DefaultModel(providerName)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_model", "Unknown provider")
			return
		}
		modelID = model.ID
	}
	if err := m.service.ValidateSelection(providerName, modelID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
		return
	}
	conversation, err := m.store.CreateConversation(r.Context(), req.Title, providerName, modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to create conversation")
		return
	}
	writeJSON(w, http.StatusCreated, conversation)
}

func (m *Module) GetConversation(w http.ResponseWriter, r *http.Request) {
	conversation, messages, err := m.store.GetConversation(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to get conversation")
		return
	}
	writeJSON(w, http.StatusOK, conversationDetail{Conversation: conversation, Messages: messages})
}

func (m *Module) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	if m.service.IsActive(mux.Vars(r)["id"]) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	var req updateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" && req.Provider == "" && req.Model == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "No conversation updates were provided")
		return
	}
	if req.Title != "" && (strings.TrimSpace(req.Title) == "" || len(strings.TrimSpace(req.Title)) > 120) {
		writeError(w, http.StatusBadRequest, "invalid_title", "Title must be 1 to 120 bytes")
		return
	}
	if req.Provider != "" || req.Model != "" {
		currentProvider := req.Provider
		currentModel := req.Model
		if currentProvider == "" || currentModel == "" {
			current, _, err := m.store.GetConversation(r.Context(), mux.Vars(r)["id"])
			if err != nil {
				writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
				return
			}
			if currentProvider == "" {
				currentProvider = current.Provider
			}
			if currentModel == "" {
				if req.Provider != "" && req.Provider != current.Provider {
					model, ok := m.cfg.DefaultModel(currentProvider)
					if !ok {
						writeError(w, 400, "invalid_model", "Unknown provider")
						return
					}
					currentModel = model.ID
				} else {
					currentModel = current.Model
				}
			}
		}
		if err := m.service.ValidateSelection(currentProvider, currentModel); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_model", err.Error())
			return
		}
	}
	if req.Provider != "" && req.Model == "" {
		model, _ := m.cfg.DefaultModel(req.Provider)
		req.Model = model.ID
	}
	conversation, err := m.store.UpdateConversation(r.Context(), mux.Vars(r)["id"], req.Title, req.Provider, req.Model)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to update conversation")
		return
	}
	writeJSON(w, http.StatusOK, conversation)
}

func (m *Module) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	if m.service.IsActive(mux.Vars(r)["id"]) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	if err := m.store.DeleteConversation(r.Context(), mux.Vars(r)["id"]); err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to delete conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if strings.TrimSpace(req.Content) == "" || len(strings.TrimSpace(req.Content)) > 512*1024 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Message content is required and must not exceed 524288 bytes")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")
	flusher, _ := w.(http.Flusher)
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
	old, err := m.store.SupersedeLatestAssistant(r.Context(), id)
	if err != nil {
		m.writeSubmitError(w, err)
		return
	}
	_, messages, err := m.store.GetConversation(r.Context(), id)
	if err != nil {
		m.writeSubmitError(w, err)
		return
	}
	content := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && messages[i].CreatedAt.Before(old.CreatedAt) {
			content = messages[i].Content
			break
		}
	}
	if content == "" {
		writeError(w, 400, "invalid_request", "No user message to regenerate")
		return
	}
	m.SubmitMessage(w, rWithJSON(r, chat.SubmitRequest{Content: content, Stream: boolPtr(false)}))
}
func boolPtr(v bool) *bool { return &v }
func rWithJSON(r *http.Request, v any) *http.Request {
	b, _ := json.Marshal(v)
	r.Body = io.NopCloser(bytes.NewReader(b))
	return r
}

func (m *Module) StopGeneration(w http.ResponseWriter, r *http.Request) {
	stopped := m.service.Stop(mux.Vars(r)["id"])
	writeJSON(w, http.StatusOK, map[string]bool{"stopped": stopped})
}

func (m *Module) requireAI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.cfg == nil || !m.cfg.Enabled || m.store == nil || m.service == nil {
			writeError(w, http.StatusServiceUnavailable, "ai_disabled", "AI is disabled. Create bs-ai-config.json and restart the server.")
			return
		}
		next(w, r)
	}
}

func (m *Module) writeSubmitError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, chat.ErrConflict) {
		status = http.StatusConflict
	} else if store.IsNotFound(err) || errors.Is(err, sql.ErrNoRows) {
		status = http.StatusNotFound
	} else if strings.Contains(err.Error(), "provider") {
		status = http.StatusBadGateway
	}
	writeError(w, status, submitErrorCode(err), safeSubmitMessage(err))
}

func submitErrorCode(err error) string {
	if errors.Is(err, chat.ErrConflict) {
		return "generation_conflict"
	}
	if store.IsNotFound(err) || errors.Is(err, sql.ErrNoRows) {
		return "not_found"
	}
	if strings.Contains(err.Error(), "provider") {
		return "provider_error"
	}
	return "invalid_request"
}

func safeSubmitMessage(err error) string {
	if strings.Contains(err.Error(), "api_key") {
		return "AI request failed"
	}
	return err.Error()
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

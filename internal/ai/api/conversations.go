package api

import (
	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type createConversationRequest struct {
	Title    string `json:"title"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Profile  string `json:"profile"`
}

type updateConversationRequest struct {
	Title    string `json:"title"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type forkConversationRequest struct {
	MessageID string `json:"message_id"`
}

type conversationDetail struct {
	Conversation store.Conversation `json:"conversation"`
	Messages     []store.Message    `json:"messages"`
}

func (m *Module) ListConversations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	includeArchived := strings.EqualFold(r.URL.Query().Get("include_archived"), "true")
	ctx := context.WithValue(r.Context(), "include_archived", includeArchived)
	conversations, err := m.store.ListConversations(ctx, r.URL.Query().Get("q"), limit)
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
	// Validate profile if provided
	if req.Profile != "" {
		if _, ok := m.profiles.Get(req.Profile); !ok {
			writeError(w, http.StatusBadRequest, "invalid_profile", fmt.Sprintf("Unknown profile %q", req.Profile))
			return
		}
	}
	conversation, err := m.store.CreateConversation(r.Context(), req.Title, providerName, modelID, req.Profile)
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
	conversationID := mux.Vars(r)["id"]
	if m.service.IsActive(conversationID) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	// Collect attachment storage keys first: deleting the conversation cascades
	// the metadata rows, so the files must be reclaimed after the DB commit.
	attachmentsToRemove, _ := m.store.ListAttachmentsForConversation(r.Context(), conversationID)
	if err := m.store.DeleteConversation(r.Context(), conversationID); err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to delete conversation")
		return
	}
	for _, att := range attachmentsToRemove {
		_ = attachments.Remove(m.attachmentsDir, conversationID, att.StorageKey)
	}
	_ = attachments.RemoveConversationDir(m.attachmentsDir, conversationID)
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) ForkConversation(w http.ResponseWriter, r *http.Request) {
	sourceID := mux.Vars(r)["id"]
	if m.service.IsActive(sourceID) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	var req forkConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.MessageID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "message_id is required")
		return
	}
	conversation, err := m.store.ForkConversation(r.Context(), sourceID, req.MessageID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation or message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to branch conversation")
		return
	}
	writeJSON(w, http.StatusCreated, conversation)
}

func (m *Module) ArchiveConversation(w http.ResponseWriter, r *http.Request) {
	if m.service.IsActive(mux.Vars(r)["id"]) {
		writeError(w, http.StatusConflict, "generation_conflict", "Generation is active")
		return
	}
	if err := m.store.ArchiveConversation(r.Context(), mux.Vars(r)["id"]); err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to archive conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) RestoreConversation(w http.ResponseWriter, r *http.Request) {
	if err := m.store.RestoreConversation(r.Context(), mux.Vars(r)["id"]); err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to restore conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) ListArchivedConversations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	conversations, err := m.store.ListArchivedConversations(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to list archived conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

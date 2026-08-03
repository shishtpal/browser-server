package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/store"

	"github.com/gorilla/mux"
)

const maxUploadMemory = 8 << 20 // ParseMultipartForm in-memory threshold

func (m *Module) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	cfg := m.cfg.Chat.Attachments
	if !cfg.Enabled {
		writeError(w, http.StatusForbidden, "attachments_disabled", "Image attachments are disabled on this server")
		return
	}
	conversationID := mux.Vars(r)["id"]
	if err := r.ParseMultipartForm(maxUploadMemory); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_upload", "Failed to parse multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_upload", "Missing file field")
		return
	}
	defer file.Close()

	filename := header.Filename
	if filename == "" {
		filename = "image"
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if len(filename) > 200 {
		filename = filename[:200]
	}

	// Read with a hard per-image cap so a huge upload cannot exhaust memory.
	data, err := io.ReadAll(io.LimitReader(file, int64(cfg.MaxImageBytes)+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_upload", "Failed to read upload")
		return
	}
	if len(data) > cfg.MaxImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "image_too_large", fmt.Sprintf("Image exceeds the %d byte limit", cfg.MaxImageBytes))
		return
	}

	contentType, width, height, err := attachments.ValidateImage(data, cfg.AllowedMIMETypes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_image", "Image type is not supported or the file is corrupt")
		return
	}

	attachmentID := store.NewID("att")
	storageKey, err := attachments.Write(m.attachmentsDir, conversationID, attachmentID, contentType, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to store image")
		return
	}

	att, err := m.store.CreateStagedAttachment(r.Context(), attachmentID, conversationID, filename, contentType, int64(len(data)), width, height, storageKey)
	if err != nil {
		_ = attachments.Remove(m.attachmentsDir, conversationID, storageKey)
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to record image")
		return
	}
	writeJSON(w, http.StatusCreated, att)
}

func (m *Module) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	attachmentID := vars["attachmentId"]
	att, err := m.store.DeleteStagedAttachment(r.Context(), conversationID, attachmentID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrAttachmentNotStaged):
			writeError(w, http.StatusConflict, "attachment_not_staged", "Attachment is already attached to a message and cannot be cancelled")
		case store.IsNotFound(err):
			writeError(w, http.StatusNotFound, "attachment_not_found", "Attachment not found")
		default:
			writeError(w, http.StatusInternalServerError, "store_error", "Failed to delete attachment")
		}
		return
	}
	_ = attachments.Remove(m.attachmentsDir, conversationID, att.StorageKey)
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) GetAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	attachmentID := vars["attachmentId"]
	att, err := m.store.GetAttachment(r.Context(), conversationID, attachmentID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "attachment_not_found", "Attachment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "store_error", "Failed to load attachment")
		return
	}
	data, err := attachments.Read(m.attachmentsDir, conversationID, att.StorageKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "Attachment file is missing")
		return
	}
	// Attachment IDs are immutable once created, so the image can be cached
	// aggressively in the browser's private cache. The token query param keeps
	// it authenticated; Cache-Control private prevents shared-cache leakage.
	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Disposition", "inline; filename=\""+sanitizeHeader(att.Filename)+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\"", "'")
	return value
}

// cleanupExpiredAttachments removes staged uploads past their retention window
// and reclaims their files. Bound per pass so a backlog drains over several
// runs without stalling the database.
func (m *Module) cleanupExpiredAttachments() {
	if m == nil || m.store == nil || m.cfg == nil || !m.cfg.Chat.Attachments.Enabled {
		return
	}
	olderThan := time.Now().UTC().Add(-time.Duration(m.cfg.Chat.Attachments.RetentionHours) * time.Hour)
	expired, err := m.store.CleanupExpiredAttachments(context.Background(), olderThan)
	if err != nil {
		return
	}
	for _, att := range expired {
		_ = attachments.Remove(m.attachmentsDir, att.ConversationID, att.StorageKey)
	}
}

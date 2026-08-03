package api

import (
	"browser-server/internal/ai/chat"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (m *Module) writeSubmitError(w http.ResponseWriter, err error) {
	if isProviderError(err) {
		code, status, _ := provider.SafeError(err)
		writeError(w, status, code, safeSubmitMessage(err))
		return
	}
	var attErr *chat.AttachmentError
	if errors.As(err, &attErr) {
		status := http.StatusBadRequest
		if attErr.Code == "attachment_not_staged" {
			status = http.StatusConflict
		}
		writeError(w, status, attErr.Code, attErr.Message)
		return
	}
	// Not a provider error — determine from other signals
	status := http.StatusBadRequest
	code := "invalid_request"
	if errors.Is(err, chat.ErrConflict) {
		status = http.StatusConflict
		code = "generation_conflict"
	} else if store.IsNotFound(err) || errors.Is(err, sql.ErrNoRows) {
		status = http.StatusNotFound
		code = "not_found"
	}
	writeError(w, status, code, safeSubmitMessage(err))
}

func submitErrorCode(err error) string {
	if isProviderError(err) {
		code, _, _ := provider.SafeError(err)
		return code
	}
	var attErr *chat.AttachmentError
	if errors.As(err, &attErr) {
		return attErr.Code
	}
	if errors.Is(err, chat.ErrConflict) {
		return "generation_conflict"
	}
	if store.IsNotFound(err) || errors.Is(err, sql.ErrNoRows) {
		return "not_found"
	}
	return "invalid_request"
}

func isProviderError(err error) bool {
	var pe *provider.Error
	return errors.As(err, &pe)
}

func safeSubmitMessage(err error) string {
	var pe *provider.Error
	if errors.As(err, &pe) {
		// Never leak API keys or secrets to the frontend
		if strings.Contains(pe.Diagnostic, "api_key") || strings.Contains(pe.Diagnostic, "API key") {
			return "AI provider request failed (authentication issue)"
		}
		// Return useful diagnostic message
		switch pe.Code {
		case "rate_limited":
			return "AI provider rate limited — the server retried but the limit persists"
		case "provider_timeout":
			return "AI provider request timed out after retries"
		case "malformed_provider_response", "malformed_provider_stream":
			return "AI provider returned an invalid response after retries"
		default:
			if pe.Diagnostic != "" {
				// Truncate long diagnostics for the frontend
				msg := pe.Diagnostic
				if len(msg) > 200 {
					msg = msg[:200] + "..."
				}
				return "AI provider request failed: " + msg
			}
			return "AI provider request failed"
		}
	}
	if strings.Contains(err.Error(), "api_key") {
		return "AI request failed"
	}
	return err.Error()
}

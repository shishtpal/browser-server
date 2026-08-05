package chat

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

// finishTurnRequest bundles everything the terminal helpers need to compute a
// persisted terminal state without exposing the whole Service.
type finishTurnRequest struct {
	conversationID        string
	assistantMessage      *store.Message
	providerName          string
	modelID               string
	providerCfg           aiconfig.ProviderConfig
	resp                  provider.ChatResponse
	providerErr           error
	iterationLimitReached bool
}

func (s *Service) buildTerminalResult(generationCtx context.Context, req finishTurnRequest) (status, logStatus, errCode, errMessage, content string, err error) {
	status = "completed"
	logStatus = "success"
	if req.providerErr != nil {
		status = "error"
		logStatus = "error"
		errCode = "provider_error"
		errCode, _, _ = provider.SafeError(req.providerErr)
		errMessage = "AI provider request failed"
		if errors.Is(generationCtx.Err(), context.Canceled) {
			status = "cancelled"
			logStatus = "cancelled"
			errCode = "cancelled"
			errMessage = "generation cancelled"
		}
	}
	content = req.resp.Content
	// Graceful stop when iteration limit is reached: save whatever content
	// we have and notify the user they can continue the conversation.
	if req.iterationLimitReached && req.providerErr == nil {
		notice := fmt.Sprintf("\n\n---\n*Tool use limit reached (%d iterations). Send a message to continue where I left off.*", s.cfg.Tools.MaxIterations)
		if content == "" {
			content = notice
		} else {
			content += notice
		}
	}
	terminalCtx, terminalCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer terminalCancel()
	completedAt, persistErr := s.store.CompleteTurn(terminalCtx, req.assistantMessage.ID, req.conversationID, content, status)
	if persistErr != nil {
		return "", "", "", "", "", fmt.Errorf("persist terminal AI result: %w", persistErr)
	}
	req.assistantMessage.CreatedAt = completedAt
	return status, logStatus, errCode, errMessage, content, nil
}

func boundedPayloads(request, response []byte, enabled bool, max int) (string, string, bool) {
	if !enabled {
		return "", "", false
	}
	req, reqTruncated := bound(redact(request), max)
	res, resTruncated := bound(redact(response), max)
	return req, res, reqTruncated || resTruncated
}

var secretPattern = regexp.MustCompile(`(?i)(authorization|api[_-]?key)\s*[\\":=]+\s*(bearer\s+)?[^\s\\",}]+|bearer\s+[A-Za-z0-9._~+/-]+`)

// dataURLPattern matches RFC 2397 image data URLs. Image bytes must never be
// retained in request logs, even when full-payload logging is enabled.
var dataURLPattern = regexp.MustCompile(`data:image/[a-zA-Z0-9+.-]+;base64,[A-Za-z0-9+/=]+`)

func redact(value []byte) []byte {
	value = secretPattern.ReplaceAll(value, []byte("$1:[REDACTED]"))
	return dataURLPattern.ReplaceAll(value, []byte("data:image/[REDACTED]"))
}

func bound(value []byte, max int) (string, bool) {
	if len(value) <= max {
		return string(value), false
	}
	return string(value[:max]), true
}

func nullableStatus(status int) *int {
	if status == 0 {
		return nil
	}
	return &status
}

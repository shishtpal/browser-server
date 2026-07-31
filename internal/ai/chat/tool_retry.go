package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

const (
	toolRetryName               = "retry_tool_call"
	defaultToolRetryWait        = 5 * time.Second
	defaultMaxToolRetryAttempts = 5
)

type toolCompletion func() (provider.ChatResponse, error)

type ignoredToolCall struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type toolRetryArguments struct {
	Message          string            `json:"message"`
	ErrorCode        string            `json:"error_code"`
	HTTPStatus       int               `json:"http_status,omitempty"`
	Attempt          int               `json:"attempt"`
	IgnoredToolCalls []ignoredToolCall `json:"ignored_tool_calls"`
}

// retryToolContinuation recovers a provider failure that happened after tools
// were executed. Provider-level retries intentionally distinguish transient
// and permanent HTTP errors; this recovery path does not. Every failure,
// including HTTP 400, is resumable.
//
// The failed assistant tool-call turn and its tool responses are removed once
// before retrying. Repeated failures reuse that same clean request so an older
// tool turn is never accidentally removed as well.
func (s *Service) retryToolContinuation(
	ctx context.Context,
	conversationID string,
	assistantMessageID string,
	yolo bool,
	request *provider.ChatRequest,
	failedResponse provider.ChatResponse,
	failedErr error,
	emit func(Event) error,
	complete toolCompletion,
) (provider.ChatResponse, error) {
	cleanMessages, ignored, ok := withoutLastToolTurn(request.Messages)
	if !ok {
		return failedResponse, failedErr
	}
	request.Messages = cleanMessages

	lastResponse := failedResponse
	lastErr := failedErr
	maxAttempts := s.maxToolRetryAttempts()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return lastResponse, err
		}

		var retryCall *provider.ToolCall
		decision := "approved"
		feedback := ""
		if yolo {
			delay := s.toolRetryWait()
			log.Printf("[AI] tool continuation failed for conversation %s (attempt %d, HTTP %d): %v — YOLO retry in %v", conversationID, attempt, lastResponse.HTTPStatus, lastErr, delay)
			if err := waitForToolRetry(ctx, delay); err != nil {
				return lastResponse, err
			}
		} else {
			var approved bool
			var comment string
			var err error
			retryCall, approved, comment, err = s.requestToolRetryApproval(
				ctx,
				conversationID,
				assistantMessageID,
				attempt,
				ignored,
				lastResponse,
				lastErr,
				emit,
			)
			if err != nil {
				return lastResponse, err
			}
			if !approved && comment == "" {
				if emitErr := emitToolRetryResult(emit, assistantMessageID, retryCall, "rejected", "cancelled", "", lastResponse, lastErr); emitErr != nil {
					return lastResponse, emitErr
				}
				return lastResponse, lastErr
			}
			if comment != "" {
				// Feedback is an explicit request to resume with additional user
				// guidance. It is safe to append after the removed tool turn.
				decision = "commented"
				feedback = comment
				request.Messages = append(request.Messages, provider.Message{Role: "user", Content: comment})
			}
		}

		response, err := complete()
		if err == nil {
			if retryCall != nil {
				if emitErr := emitToolRetryResult(emit, assistantMessageID, retryCall, decision, "resumed", feedback, response, nil); emitErr != nil {
					return response, emitErr
				}
			}
			return response, nil
		}

		if retryCall != nil {
			if emitErr := emitToolRetryResult(emit, assistantMessageID, retryCall, decision, "failed", feedback, response, err); emitErr != nil {
				return response, emitErr
			}
		}
		lastResponse = response
		lastErr = err
	}
	// YOLO loop exhausted all attempts — return the last failure wrapped with context
	return lastResponse, fmt.Errorf("tool retry exhausted after %d attempts: %w", maxAttempts, lastErr)
}

func (s *Service) requestToolRetryApproval(
	ctx context.Context,
	conversationID string,
	assistantMessageID string,
	attempt int,
	ignored []provider.ToolCall,
	failedResponse provider.ChatResponse,
	failedErr error,
	emit func(Event) error,
) (*provider.ToolCall, bool, string, error) {
	call := &provider.ToolCall{
		ID:        store.NewID("retry"),
		Name:      toolRetryName,
		Arguments: string(toolRetryArgumentsJSON(attempt, ignored, failedResponse, failedErr)),
	}
	pending, err := s.beginToolApproval(conversationID, call.ID)
	if err != nil {
		return call, false, "", err
	}
	if emit == nil {
		s.removePendingToolCall(call.ID)
		return call, false, "", fmt.Errorf("manual tool retry approval requires streaming")
	}
	if err := emit(Event{Type: "tool_call", MessageID: assistantMessageID, ToolCall: call, Status: "pending"}); err != nil {
		s.removePendingToolCall(call.ID)
		return call, false, "", err
	}
	approved, comment, err := s.waitForToolDecision(ctx, call.ID, pending)
	return call, approved, comment, err
}

func toolRetryArgumentsJSON(attempt int, ignored []provider.ToolCall, response provider.ChatResponse, err error) []byte {
	code, _, _ := provider.SafeError(err)
	calls := make([]ignoredToolCall, 0, len(ignored))
	for _, call := range ignored {
		calls = append(calls, ignoredToolCall{ID: call.ID, Name: call.Name})
	}
	value, err := json.Marshal(toolRetryArguments{
		Message:          "The AI provider failed while continuing after a tool call. Resume without the last tool-call turn?",
		ErrorCode:        code,
		HTTPStatus:       response.HTTPStatus,
		Attempt:          attempt,
		IgnoredToolCalls: calls,
	})
	if err != nil {
		log.Printf("[AI] failed to marshal retry tool arguments: %v", err)
		return []byte("{}")
	}
	return value
}

func emitToolRetryResult(
	emit func(Event) error,
	assistantMessageID string,
	call *provider.ToolCall,
	decision string,
	status string,
	feedback string,
	response provider.ChatResponse,
	retryErr error,
) error {
	if emit == nil || call == nil {
		return nil
	}
	result := map[string]any{"status": status}
	if feedback != "" {
		result["comment"] = feedback
	}
	messageStatus := "completed"
	if retryErr != nil {
		code, _, _ := provider.SafeError(retryErr)
		result["error"] = code
		if response.HTTPStatus != 0 {
			result["http_status"] = response.HTTPStatus
		}
		messageStatus = "error"
	}
	content, err := json.Marshal(map[string]any{
		"tool":     call.Name,
		"args":     json.RawMessage(call.Arguments),
		"result":   result,
		"decision": decision,
	})
	if err != nil {
		return err
	}
	return emit(Event{
		Type:      "tool_result",
		MessageID: assistantMessageID,
		ToolCall:  call,
		Content:   string(content),
		Status:    messageStatus,
	})
}

// withoutLastToolTurn returns a copy that ends immediately before the most
// recent assistant message containing tool calls. Removing the entire turn is
// required by OpenAI-compatible APIs: retaining either an orphaned assistant
// tool call or an orphaned tool result produces another invalid payload.
func withoutLastToolTurn(messages []provider.Message) ([]provider.Message, []provider.ToolCall, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "assistant" || len(messages[i].ToolCalls) == 0 {
			continue
		}
		clean := append([]provider.Message(nil), messages[:i]...)
		for _, message := range messages[i+1:] {
			if message.Role == "user" {
				clean = append(clean, message)
			}
		}
		ignored := append([]provider.ToolCall(nil), messages[i].ToolCalls...)
		return clean, ignored, true
	}
	return append([]provider.Message(nil), messages...), nil, false
}

func (s *Service) toolRetryWait() time.Duration {
	if s.toolRetryDelay > 0 {
		return s.toolRetryDelay
	}
	return defaultToolRetryWait
}

func (s *Service) maxToolRetryAttempts() int {
	if s.toolRetryAttempts > 0 {
		return s.toolRetryAttempts
	}
	return defaultMaxToolRetryAttempts
}

func waitForToolRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

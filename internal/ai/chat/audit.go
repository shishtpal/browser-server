package chat

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

func (s *Service) auditRequest(callCtx context.Context, source, conversationID, messageID, taskID string, iteration int, providerName, modelID string, resp provider.ChatResponse, callErr error) string {
	if s == nil || s.cfg == nil || !s.cfg.Logging.Enabled || s.store == nil {
		return ""
	}
	id := store.NewID("req")
	status, code, message := "success", "", ""
	if callErr != nil {
		status, code, message = "error", "provider_error", "AI provider request failed"
		code, _, _ = provider.SafeError(callErr)
		if errors.Is(callCtx.Err(), context.Canceled) {
			status, code, message = "cancelled", "cancelled", "generation cancelled"
		}
	}
	rawResponse := resp.RawResponse
	if len(rawResponse) == 0 && callErr == nil {
		rawResponse, _ = json.Marshal(struct {
			Content   string              `json:"content,omitempty"`
			ToolCalls []provider.ToolCall `json:"tool_calls,omitempty"`
			Usage     provider.Usage      `json:"usage"`
		}{Content: resp.Content, ToolCalls: resp.ToolCalls, Usage: resp.Usage})
	}
	req, res, truncated := boundedPayloads(resp.RawRequest, rawResponse, s.cfg.Logging.LogFullPayload, s.cfg.Logging.MaxPayloadBytes)
	providerCfg := s.cfg.Providers[providerName]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.store.InsertRequestLog(ctx, store.RequestLog{ID: id, Source: source, ConversationID: conversationID, MessageID: messageID, TaskID: taskID, Iteration: iteration, Provider: providerName, Model: modelID, Endpoint: strings.TrimRight(providerCfg.BaseURL, "/") + "/chat/completions", RequestPayload: req, ResponsePayload: res, PayloadTruncated: truncated, HTTPStatus: nullableStatus(resp.HTTPStatus), PromptTokens: resp.Usage.PromptTokens, CompletionTokens: resp.Usage.CompletionTokens, TotalTokens: resp.Usage.TotalTokens, LatencyMS: resp.Latency.Milliseconds(), Status: status, ErrorCode: code, ErrorMessage: message})
	if err != nil {
		log.Printf("AI request audit write failed: %v", err)
		return ""
	}
	return id
}

func (s *Service) AuditTool(requestID, messageID, name, arguments string, result []byte, toolErr error, status, decision string, duration time.Duration) {
	if s == nil || s.cfg == nil || !s.cfg.Logging.Enabled || s.store == nil || requestID == "" {
		return
	}
	args, output := "", ""
	truncated := false
	if s.cfg.Logging.LogFullPayload {
		var argumentsTruncated, resultTruncated bool
		args, argumentsTruncated = bound(redact([]byte(arguments)), s.cfg.Logging.MaxPayloadBytes)
		output, resultTruncated = bound(redact(result), s.cfg.Logging.MaxPayloadBytes)
		truncated = argumentsTruncated || resultTruncated
	}
	errMessage := ""
	if toolErr != nil {
		if s.cfg.Logging.LogFullPayload {
			var errorTruncated bool
			errMessage, errorTruncated = bound(redact([]byte(toolErr.Error())), s.cfg.Logging.MaxPayloadBytes)
			truncated = truncated || errorTruncated
		} else {
			errMessage = "tool execution failed"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.store.InsertToolCall(ctx, store.ToolCall{RequestID: requestID, MessageID: messageID, ToolName: name, Arguments: args, Result: output, ErrorMessage: errMessage, Status: status, Decision: decision, DurationMS: duration.Milliseconds(), PayloadTruncated: truncated}); err != nil {
		log.Printf("AI tool audit write failed: %v", err)
	}
}

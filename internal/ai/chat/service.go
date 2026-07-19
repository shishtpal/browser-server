package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
)

const maxMessageBytes = 512 * 1024

var ErrConflict = errors.New("generation already active")

type Service struct {
	cfg      *aiconfig.Config
	store    *store.Store
	clients  map[string]provider.Client
	activeMu sync.Mutex
	active   map[string]context.CancelFunc
	tools    *tools.Registry
}

type SubmitRequest struct {
	Content      string `json:"content"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Stream       *bool  `json:"stream"`
	ToolsEnabled bool   `json:"tools_enabled"`
}

type SubmitResponse struct {
	ConversationID   string          `json:"conversation_id"`
	UserMessage      store.Message   `json:"user_message"`
	AssistantMessage store.Message   `json:"assistant_message"`
	ToolMessages     []store.Message `json:"tool_messages,omitempty"`
	Usage            provider.Usage  `json:"usage"`
}

type Event struct {
	Type      string             `json:"type"`
	MessageID string             `json:"message_id,omitempty"`
	Content   string             `json:"content,omitempty"`
	ToolCall  *provider.ToolCall `json:"tool_call,omitempty"`
	Usage     provider.Usage     `json:"usage,omitempty"`
}

func NewService(cfg *aiconfig.Config, st *store.Store) *Service {
	clients := map[string]provider.Client{}
	for name, item := range cfg.Providers {
		clients[name] = provider.NewOpenAICompatibleClient(item.BaseURL, item.APIKey, time.Duration(item.RequestTimeoutSeconds)*time.Second)
	}
	return &Service{cfg: cfg, store: st, clients: clients, active: map[string]context.CancelFunc{}, tools: tools.New()}
}

func (s *Service) DefaultSelection() (string, string) {
	providerName := s.cfg.DefaultProvider
	model, _ := s.cfg.DefaultModel(providerName)
	return providerName, model.ID
}

func (s *Service) ValidateSelection(providerName, modelID string) error {
	if providerName == "" {
		providerName = s.cfg.DefaultProvider
	}
	if modelID == "" {
		model, ok := s.cfg.DefaultModel(providerName)
		if !ok {
			return fmt.Errorf("no default model configured for provider")
		}
		modelID = model.ID
	}
	_, _, ok := s.cfg.FindModel(providerName, modelID)
	if !ok {
		return fmt.Errorf("unknown provider/model selection")
	}
	return nil
}

func (s *Service) Submit(ctx context.Context, conversationID string, req SubmitRequest) (SubmitResponse, error) {
	return s.SubmitStream(ctx, conversationID, req, nil)
}

func (s *Service) SubmitStream(ctx context.Context, conversationID string, req SubmitRequest, emit func(Event) error) (SubmitResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return SubmitResponse{}, fmt.Errorf("message content is required")
	}
	if len(content) > maxMessageBytes {
		return SubmitResponse{}, fmt.Errorf("message content exceeds %d bytes", maxMessageBytes)
	}
	conversation, _, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return SubmitResponse{}, err
	}
	providerName := req.Provider
	modelID := req.Model
	if providerName == "" {
		providerName = conversation.Provider
	}
	if modelID == "" {
		modelID = conversation.Model
	}
	providerCfg, modelCfg, ok := s.cfg.FindModel(providerName, modelID)
	if !ok {
		return SubmitResponse{}, fmt.Errorf("unknown provider/model selection")
	}
	if req.ToolsEnabled && !modelCfg.SupportsTools {
		return SubmitResponse{}, fmt.Errorf("selected model does not support tools")
	}
	client, ok := s.clients[providerName]
	if !ok {
		return SubmitResponse{}, fmt.Errorf("provider client is unavailable")
	}

	generationCtx, cancel := context.WithCancel(ctx)
	if !s.start(conversationID, cancel) {
		cancel()
		return SubmitResponse{}, ErrConflict
	}
	defer s.finish(conversationID)

	userMessage, assistantMessage, err := s.store.BeginTurn(ctx, conversationID, content)
	if err != nil {
		cancel()
		return SubmitResponse{}, err
	}

	messages, err := s.store.ListMessages(ctx, conversationID, 0)
	if err != nil {
		cancel()
		return SubmitResponse{}, err
	}
	providerMessages := s.providerMessages(messages)
	maxOutput := modelCfg.MaxOutputTokens
	chatReq := provider.ChatRequest{
		Provider:        providerName,
		Model:           modelID,
		Messages:        providerMessages,
		Temperature:     s.cfg.Chat.Temperature,
		MaxOutputTokens: maxOutput,
	}
	if req.ToolsEnabled && s.cfg.Tools.Enabled {
		chatReq.Tools = s.tools.Specs(s.cfg.Tools.Allowed)
	}
	var resp provider.ChatResponse
	var providerErr error
	complete := func() (provider.ChatResponse, error) {
		if emit != nil {
			resp, providerErr = client.Stream(generationCtx, chatReq, func(pe provider.Event) error {
				switch pe.Type {
				case "text_delta":
					return emit(Event{Type: "delta", MessageID: assistantMessage.ID, Content: pe.Text})
				case "tool_call":
					return emit(Event{Type: "tool_call", MessageID: assistantMessage.ID, ToolCall: pe.ToolCall})
				}
				return nil
			})
		} else {
			resp, providerErr = client.Complete(generationCtx, chatReq)
		}
		return resp, providerErr
	}
	resp, providerErr = complete()
	var toolMessages []store.Message
	for iteration := 0; providerErr == nil && len(resp.ToolCalls) > 0; iteration++ {
		if iteration >= s.cfg.Tools.MaxIterations {
			providerErr = &provider.Error{Code: "tool_iteration_limit", Status: 429}
			break
		}
		chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
		for _, call := range resp.ToolCalls {
			if emit != nil {
				_ = emit(Event{Type: "tool_call", MessageID: assistantMessage.ID, ToolCall: &call})
			}
			result, toolErr := s.tools.Execute(generationCtx, call.Name, json.RawMessage(call.Arguments))
			toolStatus := "completed"
			if toolErr != nil {
				toolStatus = "error"
				result = []byte(`{"error":"` + toolErr.Error() + `"}`)
			}
			// Store tool content as JSON with metadata for UI display
			toolContent := `{"tool":"` + call.Name + `","args":` + call.Arguments + `,"result":` + string(result) + `}`
			toolMsg, addErr := s.store.AddMessage(generationCtx, conversationID, "tool", toolContent, toolStatus, call.ID)
			if addErr != nil {
				providerErr = addErr
				break
			}
			toolMessages = append(toolMessages, toolMsg)
			chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "tool", ToolCallID: call.ID, Content: string(result)})
			if emit != nil {
				_ = emit(Event{Type: "tool_result", MessageID: assistantMessage.ID, Content: toolContent})
			}
		}
		if providerErr != nil {
			break
		}
		resp, providerErr = complete()
	}
	status := "completed"
	logStatus := "success"
	errCode := ""
	errMessage := ""
	if providerErr != nil {
		status = "error"
		logStatus = "error"
		errCode = "provider_error"
		errCode, _, _ = provider.SafeError(providerErr)
		errMessage = "AI provider request failed"
		if errors.Is(generationCtx.Err(), context.Canceled) {
			status = "cancelled"
			logStatus = "cancelled"
			errCode = "cancelled"
			errMessage = "generation cancelled"
		}
	}
	contentToSave := resp.Content
	if providerErr != nil && contentToSave == "" {
		contentToSave = ""
	}
	requestPayload, responsePayload, truncated := boundedPayloads(resp.RawRequest, resp.RawResponse, s.cfg.Logging.LogFullPayload, s.cfg.Logging.MaxPayloadBytes)
	httpStatus := nullableStatus(resp.HTTPStatus)
	terminalCtx, terminalCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer terminalCancel()
	err = s.store.FinishTurn(terminalCtx, assistantMessage.ID, contentToSave, status, store.RequestLog{
		ConversationID:   conversationID,
		MessageID:        assistantMessage.ID,
		Provider:         providerName,
		Model:            modelID,
		Endpoint:         strings.TrimRight(providerCfg.BaseURL, "/") + "/chat/completions",
		RequestPayload:   requestPayload,
		ResponsePayload:  responsePayload,
		PayloadTruncated: truncated,
		HTTPStatus:       httpStatus,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		LatencyMS:        resp.Latency.Milliseconds(),
		Status:           logStatus,
		ErrorCode:        errCode,
		ErrorMessage:     errMessage,
	})
	if err != nil {
		cancel()
		return SubmitResponse{}, fmt.Errorf("persist terminal AI result: %w", err)
	}
	assistantMessage.Content = contentToSave
	assistantMessage.Status = status
	if providerErr != nil {
		cancel()
		return SubmitResponse{}, providerErr
	}
	return SubmitResponse{
		ConversationID:   conversationID,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		ToolMessages:     toolMessages,
		Usage:            resp.Usage,
	}, nil
}

func (s *Service) Stop(conversationID string) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	cancel, ok := s.active[conversationID]
	if ok {
		cancel()
	}
	return ok
}

func (s *Service) start(conversationID string, cancel context.CancelFunc) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	if _, ok := s.active[conversationID]; ok {
		return false
	}
	s.active[conversationID] = cancel
	return true
}

func (s *Service) finish(conversationID string) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	delete(s.active, conversationID)
}

func (s *Service) providerMessages(messages []store.Message) []provider.Message {
	out := []provider.Message{{Role: "system", Content: s.cfg.Chat.SystemPrompt}}
	start := 0
	limit := s.cfg.Chat.MaxHistoryMessages
	if limit > 0 && len(messages) > limit {
		start = len(messages) - limit
	}
	for _, message := range messages[start:] {
		if message.Role == "system" || message.Status == "superseded" || message.Status == "pending" || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if message.Role == "assistant" && message.Status != "completed" && message.Status != "cancelled" && message.Status != "error" {
			continue
		}
		pm := provider.Message{Role: message.Role, Content: message.Content}
		if message.Role == "tool" && message.ToolCallID != "" {
			pm.ToolCallID = message.ToolCallID
			// For tool messages, the stored content includes UI metadata. Extract only the result for the provider.
			var parsed struct {
				Result json.RawMessage `json:"result"`
			}
			if json.Unmarshal([]byte(message.Content), &parsed) == nil && parsed.Result != nil {
				pm.Content = string(parsed.Result)
			}
		}
		out = append(out, pm)
	}
	return out
}

func boundedPayloads(request, response []byte, enabled bool, max int) (string, string, bool) {
	if !enabled {
		return "", "", false
	}
	req, reqTruncated := bound(redact(request), max)
	res, resTruncated := bound(redact(response), max)
	return req, res, reqTruncated || resTruncated
}

var secretPattern = regexp.MustCompile(`(?i)(authorization|api[_-]?key)\s*[":=]+\s*(bearer\s+)?[^\s",}]+|bearer\s+[A-Za-z0-9._~+/-]+`)

func redact(value []byte) []byte { return secretPattern.ReplaceAll(value, []byte("$1:[REDACTED]")) }

func (s *Service) IsActive(id string) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	_, ok := s.active[id]
	return ok
}
func (s *Service) Close() {
	s.activeMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.active))
	for _, c := range s.active {
		cancels = append(cancels, c)
	}
	s.activeMu.Unlock()
	for _, c := range cancels {
		c()
	}
	for i := 0; i < 50; i++ {
		s.activeMu.Lock()
		n := len(s.active)
		s.activeMu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
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

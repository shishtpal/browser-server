package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
)

const maxMessageBytes = 512 * 1024

var ErrConflict = errors.New("generation already active")
var ErrToolCallNotPending = errors.New("tool call is not pending approval")

type toolDecision struct {
	approved bool
	comment  string
}

type pendingToolCall struct {
	conversationID string
	decision       chan toolDecision
}

type Service struct {
	cfg       *aiconfig.Config
	store     *store.Store
	profiles  *profiles.Registry
	skills    *skills.Registry
	clients   map[string]provider.Client
	activeMu  sync.Mutex
	active    map[string]context.CancelFunc
	tools     *tools.Registry
	pendingMu sync.Mutex
	pending   map[string]pendingToolCall
}

type SubmitRequest struct {
	Content      string   `json:"content"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Stream       *bool    `json:"stream"`
	ToolsEnabled bool     `json:"tools_enabled"`
	YOLOMode     bool     `json:"yolo_mode"`
	ActiveTools  []string `json:"active_tools,omitempty"`
	Skills       []string `json:"skills,omitempty"`
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
	Status    string             `json:"status,omitempty"`
	Usage     provider.Usage     `json:"usage,omitempty"`
}

func NewService(cfg *aiconfig.Config, st *store.Store, profileReg *profiles.Registry, skillReg *skills.Registry) *Service {
	clients := map[string]provider.Client{}
	for name, item := range cfg.Providers {
		clients[name] = provider.NewOpenAICompatibleClient(
			item.BaseURL,
			item.APIKey,
			time.Duration(item.RequestTimeoutSeconds)*time.Second,
			item.RetryAttempts,
			time.Duration(item.RetryDelaySeconds)*time.Second,
		)
	}
	return &Service{
		cfg: cfg, store: st, profiles: profileReg, skills: skillReg, clients: clients, active: map[string]context.CancelFunc{},
		tools: tools.New(tools.Options{Memory: cfg.Memory, Skills: skillReg}), pending: map[string]pendingToolCall{},
	}
}

func (s *Service) DefaultSelection() (string, string) {
	providerName := s.cfg.DefaultProvider
	model, _ := s.cfg.DefaultModel(providerName)
	return providerName, model.ID
}

// ToolCategories returns a map of tool name → category for all allowed tools.
func (s *Service) ToolCategories() map[string]string {
	return s.tools.Categories(s.cfg.Tools.Allowed)
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

// resolveActiveTools computes the effective tool list for a request.
// If skills specify tool whitelists, only their union (intersected with server allowed) is used.
// If the client sends an explicit ActiveTools list, it's intersected further.
// Skill meta-tools are always included.
func (s *Service) resolveActiveTools(clientActive []string, activeSkills []*skills.Skill) []string {
	allowed := s.cfg.Tools.Allowed

	// If any active skill specifies a tools list, union them to form the base
	hasToolRestriction := false
	skillToolSet := make(map[string]bool)
	for _, skill := range activeSkills {
		if len(skill.Tools) > 0 {
			hasToolRestriction = true
			for _, name := range skill.Tools {
				skillToolSet[name] = true
			}
		}
	}

	if hasToolRestriction {
		// Always include skill meta-tools
		for _, name := range tools.SkillToolNames() {
			skillToolSet[name] = true
		}
		allowedSet := make(map[string]bool, len(allowed))
		for _, name := range allowed {
			allowedSet[name] = true
		}
		var skillAllowed []string
		for name := range skillToolSet {
			if allowedSet[name] {
				skillAllowed = append(skillAllowed, name)
			}
		}
		allowed = skillAllowed
	}

	if clientActive == nil {
		return allowed
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = true
	}
	// Always include skill meta-tools even if client didn't list them (only if skills exist)
	hasSkills := s.skills != nil && len(s.skills.List()) > 0
	if hasSkills {
		for _, name := range tools.SkillToolNames() {
			allowedSet[name] = true
		}
	}
	var result []string
	for _, name := range clientActive {
		if allowedSet[name] {
			result = append(result, name)
		}
	}
	// Ensure skill tools are always present when skills exist
	if hasSkills {
		for _, name := range tools.SkillToolNames() {
			found := false
			for _, r := range result {
				if r == name {
					found = true
					break
				}
			}
			if !found {
				result = append(result, name)
			}
		}
	}
	return result
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
	if req.ToolsEnabled && !req.YOLOMode && emit == nil {
		return SubmitResponse{}, fmt.Errorf("manual tool approval requires streaming")
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
	// Resolve system prompt: use profile content if conversation has one, else config default
	systemPrompt := s.cfg.Chat.SystemPrompt
	if conversation.Profile != "" {
		if content, ok := s.profiles.Get(conversation.Profile); ok {
			systemPrompt = content
		} else {
			log.Printf("WARN: conversation %s references unknown profile %q, using default", conversationID, conversation.Profile)
		}
	}

	// Initialize session skills from request (user-toggled) or conversation state
	var sessionSkills []*skills.Skill
	reqSkills := req.Skills
	if len(reqSkills) == 0 && len(conversation.Skills) > 0 {
		reqSkills = conversation.Skills
	}
	for _, name := range reqSkills {
		if sk, ok := s.skills.Get(name); ok {
			sessionSkills = append(sessionSkills, sk)
		}
	}

	// Build the full system prompt with skills preamble + active skill content
	basePrompt := systemPrompt
	fullPrompt := s.buildFullPrompt(basePrompt, sessionSkills)
	providerMessages := s.providerMessages(messages, fullPrompt)
	maxOutput := modelCfg.MaxOutputTokens
	chatReq := provider.ChatRequest{
		Provider:        providerName,
		Model:           modelID,
		Messages:        providerMessages,
		Temperature:     s.cfg.Chat.Temperature,
		MaxOutputTokens: maxOutput,
	}
	activeToolSet := map[string]bool{}
	if req.ToolsEnabled && s.cfg.Tools.Enabled {
		activeTools := s.resolveActiveTools(req.ActiveTools, sessionSkills)
		chatReq.Tools = s.tools.Specs(activeTools)
		for _, name := range activeTools {
			activeToolSet[name] = true
		}
	}
	var resp provider.ChatResponse
	var providerErr error
	complete := func() (provider.ChatResponse, error) {
		if emit != nil {
			resp, providerErr = client.Stream(generationCtx, chatReq, func(pe provider.Event) error {
				switch pe.Type {
				case "text_delta":
					return emit(Event{Type: "delta", MessageID: assistantMessage.ID, Content: pe.Text})
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
	iterationLimitReached := false
	for iteration := 0; providerErr == nil && len(resp.ToolCalls) > 0; iteration++ {
		if iteration >= s.cfg.Tools.MaxIterations {
			iterationLimitReached = true
			break
		}
		chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
		for callIndex, call := range resp.ToolCalls {
			if call.ID == "" {
				call.ID = store.NewID("call")
				resp.ToolCalls[callIndex].ID = call.ID
				chatReq.Messages[len(chatReq.Messages)-1].ToolCalls[callIndex].ID = call.ID
			}

			// Intercept skill meta-tools (they modify session state, not executed by registry)
			if skillResult, handled := s.handleSkillTool(call, sessionSkills, basePrompt, &chatReq, req.ActiveTools, &activeToolSet); handled {
				// Update sessionSkills from the handler
				sessionSkills = s.getUpdatedSessionSkills(call, sessionSkills)
				toolContentBytes, _ := json.Marshal(map[string]any{
					"tool": call.Name, "args": json.RawMessage(call.Arguments), "result": json.RawMessage(skillResult), "decision": "approved",
				})
				toolMsg, addErr := s.store.AddMessage(generationCtx, conversationID, "tool", string(toolContentBytes), "completed", call.ID)
				if addErr != nil {
					providerErr = addErr
					break
				}
				toolMessages = append(toolMessages, toolMsg)
				chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "tool", ToolCallID: call.ID, Content: string(skillResult)})
				if emit != nil {
					_ = emit(Event{Type: "tool_result", MessageID: assistantMessage.ID, ToolCall: &call, Content: string(toolContentBytes), Status: "completed"})
				}
				continue
			}

			authorized := activeToolSet[call.Name]
			approved := authorized && req.YOLOMode
			var pending pendingToolCall
			if authorized && !approved {
				pending, providerErr = s.beginToolApproval(conversationID, call.ID)
				if providerErr != nil {
					break
				}
			}
			if emit != nil {
				status := "approved"
				if !authorized {
					status = "error"
				} else if !approved {
					status = "pending"
				}
				if emitErr := emit(Event{Type: "tool_call", MessageID: assistantMessage.ID, ToolCall: &call, Status: status}); emitErr != nil {
					s.removePendingToolCall(call.ID)
					providerErr = emitErr
					break
				}
			}
			var comment string
			if authorized && !approved {
				approved, comment, providerErr = s.waitForToolDecision(generationCtx, call.ID, pending)
				if providerErr != nil {
					break
				}
			}
			var result []byte
			var toolErr error
			toolStatus := "completed"
			decision := "approved"
			providerToolContent := ""
			if !authorized {
				decision = "rejected"
				toolErr = fmt.Errorf("tool %q is not enabled for this request", call.Name)
				toolStatus = "error"
				result, _ = json.Marshal(map[string]string{"error": toolErr.Error()})
				providerToolContent = string(result)
			} else if comment != "" {
				// User supplied feedback instead of running the tool; feed the
				// comment back to the model as the tool result so it can adjust.
				decision = "commented"
				result, _ = json.Marshal(map[string]string{"comment": comment})
				providerToolContent = comment
			} else if approved {
				result, toolErr = s.tools.Execute(generationCtx, call.Name, json.RawMessage(call.Arguments))
				if toolErr != nil {
					toolStatus = "error"
					result, _ = json.Marshal(map[string]string{"error": toolErr.Error()})
				}
				providerToolContent = string(result)
			} else {
				decision = "rejected"
				toolErr = errors.New("rejected by user")
				toolStatus = "error"
				result, _ = json.Marshal(map[string]string{"error": toolErr.Error()})
				providerToolContent = string(result)
			}
			var displayArgs any
			if json.Unmarshal([]byte(call.Arguments), &displayArgs) != nil {
				displayArgs = call.Arguments
			}
			toolContentBytes, marshalErr := json.Marshal(map[string]any{
				"tool": call.Name, "args": displayArgs, "result": json.RawMessage(result), "decision": decision,
			})
			if marshalErr != nil {
				providerErr = marshalErr
				break
			}
			toolContent := string(toolContentBytes)
			toolMsg, addErr := s.store.AddMessage(generationCtx, conversationID, "tool", toolContent, toolStatus, call.ID)
			if addErr != nil {
				providerErr = addErr
				break
			}
			toolMessages = append(toolMessages, toolMsg)
			chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "tool", ToolCallID: call.ID, Content: providerToolContent})
			if emit != nil {
				_ = emit(Event{Type: "tool_result", MessageID: assistantMessage.ID, ToolCall: &call, Content: toolContent, Status: toolStatus})
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
	// Graceful stop when iteration limit is reached: save whatever content
	// we have and notify the user they can continue the conversation.
	if iterationLimitReached && providerErr == nil {
		if contentToSave == "" {
			contentToSave = fmt.Sprintf("\n\n---\n*Tool use limit reached (%d iterations). Send a message to continue where I left off.*", s.cfg.Tools.MaxIterations)
		} else {
			contentToSave += fmt.Sprintf("\n\n---\n*Tool use limit reached (%d iterations). Send a message to continue where I left off.*", s.cfg.Tools.MaxIterations)
		}
		if emit != nil {
			_ = emit(Event{Type: "delta", MessageID: assistantMessage.ID, Content: fmt.Sprintf("\n\n---\n*Tool use limit reached (%d iterations). Send a message to continue where I left off.*", s.cfg.Tools.MaxIterations)})
		}
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
	// Persist final session skills state
	if len(sessionSkills) > 0 {
		skillNames := make([]string, len(sessionSkills))
		for i, sk := range sessionSkills {
			skillNames[i] = sk.Name
		}
		_ = s.store.UpdateConversationSkills(context.Background(), conversationID, skillNames)
	} else if len(conversation.Skills) > 0 {
		_ = s.store.UpdateConversationSkills(context.Background(), conversationID, []string{})
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

func (s *Service) DecideToolCall(conversationID, callID string, approved bool, comment string) error {
	s.pendingMu.Lock()
	pending, ok := s.pending[callID]
	if ok && pending.conversationID == conversationID {
		delete(s.pending, callID)
	} else {
		ok = false
	}
	s.pendingMu.Unlock()
	if !ok {
		return ErrToolCallNotPending
	}
	pending.decision <- toolDecision{approved: approved, comment: comment}
	return nil
}

func (s *Service) beginToolApproval(conversationID, callID string) (pendingToolCall, error) {
	pending := pendingToolCall{conversationID: conversationID, decision: make(chan toolDecision, 1)}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if _, exists := s.pending[callID]; exists {
		return pendingToolCall{}, fmt.Errorf("duplicate tool call id")
	}
	s.pending[callID] = pending
	return pending, nil
}

func (s *Service) waitForToolDecision(ctx context.Context, callID string, pending pendingToolCall) (bool, string, error) {
	select {
	case decision := <-pending.decision:
		return decision.approved, decision.comment, nil
	case <-ctx.Done():
		s.removePendingToolCall(callID)
		return false, "", ctx.Err()
	}
}

func (s *Service) removePendingToolCall(callID string) {
	s.pendingMu.Lock()
	delete(s.pending, callID)
	s.pendingMu.Unlock()
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

func (s *Service) providerMessages(messages []store.Message, systemPrompt string) []provider.Message {
	out := []provider.Message{{Role: "system", Content: systemPrompt}}
	start := 0
	limit := s.cfg.Chat.MaxHistoryMessages
	if limit > 0 && len(messages) > limit {
		start = len(messages) - limit
	}

	// First pass: identify tool_call_ids present in tool messages so we can
	// reconstruct assistant+tool pairs from history.
	toolCallIDs := map[string]bool{}
	for _, message := range messages[start:] {
		if message.Role == "tool" && message.ToolCallID != "" && message.Status == "completed" {
			toolCallIDs[message.ToolCallID] = true
		}
	}

	// Second pass: build the provider message list.
	// For tool messages, we need them paired with an assistant message that
	// contains the corresponding tool_calls array. Since we only store the
	// flattened tool result messages (not the raw assistant tool_calls envelope),
	// we reconstruct a synthetic assistant envelope when we encounter a group
	// of tool results that follows a non-tool message.
	//
	// The sequence in stored messages is:
	//   user → assistant(content) → tool(call1) → tool(call2) → assistant(next) ...
	// We need to produce:
	//   user → assistant{content, tool_calls:[call1,call2]} → tool(call1) → tool(call2) → assistant(next) ...
	//
	// Strategy: collect consecutive tool messages, then emit a synthetic
	// assistant with tool_calls before emitting the tool results.

	type toolGroup struct {
		calls   []provider.ToolCall
		results []provider.Message
	}

	var pendingTools toolGroup

	flushTools := func() {
		if len(pendingTools.calls) == 0 {
			return
		}
		// Emit a synthetic assistant message with tool_calls
		out = append(out, provider.Message{Role: "assistant", ToolCalls: pendingTools.calls})
		// Emit tool result messages
		out = append(out, pendingTools.results...)
		pendingTools = toolGroup{}
	}

	for _, message := range messages[start:] {
		if message.Role == "system" || message.Status == "superseded" || message.Status == "pending" || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if message.Role == "assistant" && message.Status != "completed" && message.Status != "cancelled" && message.Status != "error" {
			continue
		}

		if message.Role == "tool" {
			if message.ToolCallID == "" || !toolCallIDs[message.ToolCallID] {
				continue
			}
			// Extract the actual result content from the stored JSON envelope.
			// Stored format: {"tool":"name","args":...,"result":...,"decision":"approved"}
			toolContent := extractToolResult(message.Content)
			pendingTools.calls = append(pendingTools.calls, provider.ToolCall{
				ID:   message.ToolCallID,
				Name: extractToolName(message.Content),
			})
			pendingTools.results = append(pendingTools.results, provider.Message{
				Role:       "tool",
				ToolCallID: message.ToolCallID,
				Content:    toolContent,
			})
			continue
		}

		// Non-tool message: flush any pending tool group first
		flushTools()

		pm := provider.Message{Role: message.Role, Content: message.Content}
		out = append(out, pm)
	}
	// Flush any trailing tool group
	flushTools()

	return out
}

// extractToolResult extracts the "result" field from the stored tool message JSON.
// Falls back to the raw content if parsing fails.
func extractToolResult(content string) string {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil || envelope.Result == nil {
		return content
	}
	// If the result is a string, unquote it; otherwise return as-is
	var s string
	if json.Unmarshal(envelope.Result, &s) == nil {
		return s
	}
	return string(envelope.Result)
}

// extractToolName extracts the "tool" field from the stored tool message JSON.
func extractToolName(content string) string {
	var envelope struct {
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		return "unknown"
	}
	return envelope.Tool
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

// buildFullPrompt composes the system prompt from base + skills preamble + active skill content.
func (s *Service) buildFullPrompt(basePrompt string, activeSkills []*skills.Skill) string {
	var b strings.Builder
	b.WriteString(basePrompt)

	// Always include skills preamble so the agent knows what's available
	b.WriteString(s.skillsPreamble())

	// Append active skill instructions
	if len(activeSkills) > 0 {
		b.WriteString("\n\n---\n\n## Active Skills\n")
		for _, skill := range activeSkills {
			b.WriteString(fmt.Sprintf("\n### %s\n\n", skill.Label))
			b.WriteString(skill.Content)
			b.WriteString("\n")
		}

		// Collect and inject context documents from all active skills
		seen := map[string]bool{}
		var contextFiles []string
		for _, skill := range activeSkills {
			for _, path := range skill.Context {
				if !seen[path] {
					seen[path] = true
					contextFiles = append(contextFiles, path)
				}
			}
		}
		if len(contextFiles) > 0 {
			b.WriteString("\n## Reference Documents\n")
			count := 0
			for _, relPath := range contextFiles {
				if count >= 5 {
					break
				}
				absPath := s.cfg.ResolvePath(relPath)
				content, err := os.ReadFile(absPath)
				if err != nil {
					continue
				}
				if len(content) > 32*1024 {
					content = content[:32*1024]
				}
				b.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", relPath, string(content)))
				count++
			}
		}
	}

	return b.String()
}

// skillsPreamble generates a brief catalog of available skills for the agent.
func (s *Service) skillsPreamble() string {
	if s.skills == nil {
		return ""
	}
	list := s.skills.List()
	if len(list) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Available Skills\n")
	b.WriteString("You can activate skills to gain focused instructions and tools using `activate_skill`.\n")
	b.WriteString("Active skills can be deactivated with `deactivate_skill`. Use `get_active_skills` to check current state.\n\n")
	for _, sk := range list {
		b.WriteString(fmt.Sprintf("- **%s** (`%s`): %s", sk.Label, sk.Name, sk.Description))
		if len(sk.Tools) > 0 {
			b.WriteString(fmt.Sprintf(" [tools: %s]", strings.Join(sk.Tools, ", ")))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// handleSkillTool intercepts skill meta-tool calls and returns (result, handled).
// If handled is true, the caller should skip normal tool execution.
func (s *Service) handleSkillTool(call provider.ToolCall, sessionSkills []*skills.Skill, basePrompt string, chatReq *provider.ChatRequest, clientActive []string, activeToolSet *map[string]bool) ([]byte, bool) {
	switch call.Name {
	case "activate_skill":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			result, _ := json.Marshal(map[string]string{"error": "invalid arguments"})
			return result, true
		}
		skill, ok := s.skills.Get(args.Name)
		if !ok {
			result, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("unknown skill %q", args.Name)})
			return result, true
		}
		if containsSkill(sessionSkills, args.Name) {
			result, _ := json.Marshal(map[string]any{"status": "already_active", "skill": args.Name})
			return result, true
		}
		if len(sessionSkills) >= s.skills.MaxActive() {
			result, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("maximum %d active skills reached", s.skills.MaxActive())})
			return result, true
		}
		// Will be added by getUpdatedSessionSkills
		newSkills := append(sessionSkills, skill)
		chatReq.Messages[0].Content = s.buildFullPrompt(basePrompt, newSkills)
		newActiveTools := s.resolveActiveTools(clientActive, newSkills)
		chatReq.Tools = s.tools.Specs(newActiveTools)
		*activeToolSet = make(map[string]bool, len(newActiveTools))
		for _, name := range newActiveTools {
			(*activeToolSet)[name] = true
		}
		result, _ := json.Marshal(map[string]any{"status": "activated", "skill": args.Name, "tools_added": skill.Tools})
		return result, true

	case "deactivate_skill":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			result, _ := json.Marshal(map[string]string{"error": "invalid arguments"})
			return result, true
		}
		if !containsSkill(sessionSkills, args.Name) {
			result, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("skill %q is not active", args.Name)})
			return result, true
		}
		newSkills := removeSkill(sessionSkills, args.Name)
		chatReq.Messages[0].Content = s.buildFullPrompt(basePrompt, newSkills)
		newActiveTools := s.resolveActiveTools(clientActive, newSkills)
		chatReq.Tools = s.tools.Specs(newActiveTools)
		*activeToolSet = make(map[string]bool, len(newActiveTools))
		for _, name := range newActiveTools {
			(*activeToolSet)[name] = true
		}
		result, _ := json.Marshal(map[string]any{"status": "deactivated", "skill": args.Name})
		return result, true

	case "get_active_skills":
		names := make([]string, len(sessionSkills))
		for i, sk := range sessionSkills {
			names[i] = sk.Name
		}
		result, _ := json.Marshal(map[string]any{"active": names})
		return result, true

	case "list_skills":
		// list_skills has a real Execute function in the registry, let it through
		return nil, false
	}
	return nil, false
}

// getUpdatedSessionSkills returns the updated session skills after a skill tool call.
func (s *Service) getUpdatedSessionSkills(call provider.ToolCall, current []*skills.Skill) []*skills.Skill {
	switch call.Name {
	case "activate_skill":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal([]byte(call.Arguments), &args)
		if sk, ok := s.skills.Get(args.Name); ok && !containsSkill(current, args.Name) {
			return append(current, sk)
		}
	case "deactivate_skill":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal([]byte(call.Arguments), &args)
		return removeSkill(current, args.Name)
	}
	return current
}

func containsSkill(list []*skills.Skill, name string) bool {
	for _, sk := range list {
		if sk.Name == name {
			return true
		}
	}
	return false
}

func removeSkill(list []*skills.Skill, name string) []*skills.Skill {
	var out []*skills.Skill
	for _, sk := range list {
		if sk.Name != name {
			out = append(out, sk)
		}
	}
	return out
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

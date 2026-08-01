package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

// Service is the public façade for the AI chat turn pipeline. Helpers split
// across sibling files (generation, tool_approval, tool_selection, tool_execution,
// prompts, messages, terminal, tool_retry) act on the same instance so the
// orchestration in SubmitStream can stay compact and readable.
type Service struct {
	cfg               *aiconfig.Config
	store             *store.Store
	profiles          *profiles.Registry
	skills            *skills.Registry
	clients           map[string]provider.Client
	activeMu          sync.Mutex
	active            map[string]context.CancelFunc
	appendMu          sync.Mutex
	appendWindows     map[string]*appendWindow
	tools             *tools.Registry
	pendingMu         sync.Mutex
	pending           map[string]pendingToolCall
	toolRetryDelay    time.Duration
	toolRetryAttempts int
}

type SubmitRequest struct {
	Content                   string   `json:"content"`
	Provider                  string   `json:"provider"`
	Model                     string   `json:"model"`
	Stream                    *bool    `json:"stream"`
	ToolsEnabled              bool     `json:"tools_enabled"`
	YOLOMode                  bool     `json:"yolo_mode"`
	IncludeAllToolDefinitions bool     `json:"include_all_tool_definitions"`
	ActiveTools               []string `json:"active_tools,omitempty"`
	Skills                    []string `json:"skills,omitempty"`
	// RawToolOutput overrides tool output encoding for this request: true
	// forces raw output for every tool that supports it, false forces JSON.
	// nil (omitted) uses the tools.raw_output config allowlist. Lets the UI
	// toggle raw vs JSON per message without editing config.
	RawToolOutput *bool `json:"raw_tool_output,omitempty"`
	regenerate    bool
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
		cfg: cfg, store: st, profiles: profileReg, skills: skillReg, clients: clients, active: map[string]context.CancelFunc{}, appendWindows: map[string]*appendWindow{},
		tools: tools.New(tools.Options{Memory: cfg.Memory, Skills: skillReg, WebSearch: cfg.WebSearch, FileTools: cfg.FileTools, Tools: cfg.Tools, Allowed: cfg.Tools.Allowed, Paths: cfg.Paths}), pending: map[string]pendingToolCall{},
		toolRetryDelay:    time.Duration(cfg.Chat.ToolRetryDelaySeconds) * time.Second,
		toolRetryAttempts: cfg.Chat.ToolRetryAttempts,
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

func (s *Service) Submit(ctx context.Context, conversationID string, req SubmitRequest) (SubmitResponse, error) {
	return s.SubmitStream(ctx, conversationID, req, nil)
}

func (s *Service) Regenerate(ctx context.Context, conversationID string) (SubmitResponse, error) {
	return s.SubmitStream(ctx, conversationID, SubmitRequest{regenerate: true}, nil)
}

func (s *Service) SubmitStream(ctx context.Context, conversationID string, req SubmitRequest, emit func(Event) error) (SubmitResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" && !req.regenerate {
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

	// Per-request raw/JSON output toggle: wrap the generation context so
	// Registry.Execute() honors it for every tool call in this turn.
	if req.RawToolOutput != nil {
		generationCtx = tools.WithRawOutputOverride(generationCtx, req.RawToolOutput)
	}

	var userMessage, assistantMessage store.Message
	if req.regenerate {
		userMessage, assistantMessage, err = s.store.BeginRegeneration(ctx, conversationID)
	} else {
		userMessage, assistantMessage, err = s.store.BeginTurn(ctx, conversationID, content)
	}
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
	loadedToolSet := map[string]bool{}
	if req.ToolsEnabled && s.cfg.Tools.Enabled {
		activeTools := s.resolveActiveTools(req.ActiveTools, sessionSkills)
		activeToolSet = s.configureToolDefinitions(&chatReq, activeTools, loadedToolSet, req.IncludeAllToolDefinitions)
		if activeToolSet["web_search"] || activeToolSet["web_fetch"] {
			providerMessages[0].Content += webSearchPromptFragment
		}
	}
	complete := func() (provider.ChatResponse, error) {
		if emit != nil {
			return client.Stream(generationCtx, chatReq, func(pe provider.Event) error {
				switch pe.Type {
				case "text_delta":
					return emit(Event{Type: "delta", MessageID: assistantMessage.ID, Content: pe.Text})
				}
				return nil
			})
		}
		return client.Complete(generationCtx, chatReq)
	}
	resp, providerErr := complete()
	var toolMessages []store.Message
	iterationLimitReached := false
	for iteration := 0; providerErr == nil && len(resp.ToolCalls) > 0; iteration++ {
		if iteration >= s.cfg.Tools.MaxIterations {
			iterationLimitReached = true
			break
		}
		chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
		appendWindow, windowErr := s.openAppendWindow(generationCtx, conversationID)
		if windowErr != nil {
			providerErr = windowErr
			break
		}
		if emit != nil {
			if emitErr := emit(Event{Type: "append_window", Status: "open"}); emitErr != nil {
				s.closeAppendWindow(conversationID, appendWindow)
				providerErr = emitErr
				break
			}
		}
		loadedAtResponseStart := make(map[string]bool, len(loadedToolSet))
		for name := range loadedToolSet {
			loadedAtResponseStart[name] = true
		}
		var iterationToolMessages []store.Message
		iterationToolMessages, providerErr = s.processToolCalls(
			generationCtx,
			conversationID,
			assistantMessage.ID,
			&chatReq,
			&resp,
			loadedAtResponseStart,
			activeToolSet,
			loadedToolSet,
			basePrompt,
			sessionSkills,
			&req,
			emit,
		)
		toolMessages = append(toolMessages, iterationToolMessages...)
		appendedMessages := s.closeAppendWindow(conversationID, appendWindow)
		if emit != nil {
			if emitErr := emit(Event{Type: "append_window", Status: "closed"}); emitErr != nil && providerErr == nil {
				providerErr = emitErr
			}
		}
		if providerErr != nil {
			break
		}
		for _, message := range appendedMessages {
			chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "user", Content: message.Content})
		}
		resp, providerErr = complete()
		if providerErr != nil {
			resp, providerErr = s.retryToolContinuation(
				generationCtx,
				conversationID,
				assistantMessage.ID,
				req.YOLOMode,
				&chatReq,
				resp,
				providerErr,
				emit,
				complete,
			)
		}
	}
	finishReq := finishTurnRequest{
		conversationID:        conversationID,
		assistantMessage:      &assistantMessage,
		providerName:          providerName,
		modelID:               modelID,
		providerCfg:           providerCfg,
		resp:                  resp,
		providerErr:           providerErr,
		iterationLimitReached: iterationLimitReached,
	}
	status, _, _, _, contentToSave, finishErr := s.buildTerminalResult(generationCtx, finishReq)
	if finishErr != nil {
		cancel()
		return SubmitResponse{}, finishErr
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

// questionAnswerResult turns the existing tool-decision comment channel into
// the ask_questions tool's structured result. JSON answers are accepted when a UI
// wants to submit several answers; plain text answers the first question.
func questionAnswerResult(request tools.QuestionRequest, comment string) []byte {
	var structured struct {
		Answers json.RawMessage `json:"answers"`
	}
	if json.Unmarshal([]byte(comment), &structured) == nil && len(structured.Answers) > 0 {
		var answers []any
		if json.Unmarshal(structured.Answers, &answers) == nil {
			result, _ := json.Marshal(map[string]any{"status": "answered", "answers": answers})
			return result
		}
	}
	answers := make([]map[string]any, 0, len(request.Questions))
	for i, question := range request.Questions {
		answer := ""
		skipped := true
		if i == 0 {
			answer = comment
			skipped = false
		}
		answers = append(answers, map[string]any{
			"id": question.ID, "prompt": question.Prompt, "answer": answer, "skipped": skipped,
		})
	}
	result, _ := json.Marshal(map[string]any{
		"status": "answered", "answers": answers,
	})
	return result
}

package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-server/internal/ai/provider"
)

// This file exposes the minimum surface a background task agent needs: one
// bounded model call and one tool execution. It deliberately does not reuse
// SubmitStream — that pipeline is built around a conversation turn with an
// approval channel and a pending-message row, neither of which exists for
// unattended work. Sharing it would mean bending the turn lifecycle around a
// caller that has no user waiting on the other end.

// AgentStepRequest is one model call with an explicit, caller-owned message
// list. The caller (the task worker) owns history via its checkpoint, so no
// conversation rows are read or written here.
type AgentStepRequest struct {
	Provider string
	Model    string
	// Messages is the complete conversation to send, including the system
	// message. It is used verbatim.
	Messages []provider.Message
	// ToolNames are the tools to expose. Names not in the server's allowed list
	// are dropped.
	ToolNames []string
	// OnProgress, when set, is called for every stream chunk received. It makes
	// the request stream rather than block, so a generation that stalls midway
	// stops signalling liveness while the HTTP connection is still open. Without
	// it a wedged provider looks identical to a slow one until StepTimeout.
	OnProgress     func()
	TaskID         string
	ConversationID string
	Iteration      int
}

// AgentStepResponse is the model's reply to one step.
type AgentStepResponse struct {
	Content   string
	ToolCalls []provider.ToolCall
	Usage     provider.Usage
	RequestID string
}

// AgentStep performs a single model call, streaming when the caller wants
// per-chunk liveness and otherwise blocking on a single request.
func (s *Service) AgentStep(ctx context.Context, req AgentStepRequest) (AgentStepResponse, error) {
	providerName := req.Provider
	modelID := req.Model
	if providerName == "" {
		providerName = s.cfg.DefaultProvider
	}
	if modelID == "" {
		model, ok := s.cfg.DefaultModel(providerName)
		if !ok {
			return AgentStepResponse{}, fmt.Errorf("no default model configured for provider %q", providerName)
		}
		modelID = model.ID
	}
	_, modelCfg, ok := s.cfg.FindModel(providerName, modelID)
	if !ok {
		return AgentStepResponse{}, fmt.Errorf("unknown provider/model selection")
	}
	client, ok := s.clients[providerName]
	if !ok {
		return AgentStepResponse{}, fmt.Errorf("provider client is unavailable")
	}
	if len(req.Messages) == 0 {
		return AgentStepResponse{}, fmt.Errorf("agent step requires at least one message")
	}

	chatReq := provider.ChatRequest{
		Provider:        providerName,
		Model:           modelID,
		Messages:        req.Messages,
		Temperature:     s.cfg.Chat.Temperature,
		MaxOutputTokens: modelCfg.MaxOutputTokens,
	}
	if len(req.ToolNames) > 0 && s.cfg.Tools.Enabled && modelCfg.SupportsTools {
		chatReq.Tools = s.tools.Specs(s.filterAllowed(req.ToolNames))
	}

	var resp provider.ChatResponse
	var err error
	if req.OnProgress != nil {
		resp, err = client.Stream(ctx, chatReq, func(provider.Event) error {
			req.OnProgress()
			return nil
		})
	} else {
		resp, err = client.Complete(ctx, chatReq)
	}
	requestID := s.auditRequest(ctx, "task_agent", req.ConversationID, "", req.TaskID, req.Iteration, providerName, modelID, resp, err)
	if err != nil {
		return AgentStepResponse{}, err
	}
	return AgentStepResponse{Content: resp.Content, ToolCalls: resp.ToolCalls, Usage: resp.Usage, RequestID: requestID}, nil
}

// ExecuteTool runs one allowed tool. Task execution is unattended, so there is
// no approval step: authorization is decided entirely by the allowed list.
func (s *Service) ExecuteTool(ctx context.Context, name string, args json.RawMessage) ([]byte, error) {
	if !s.cfg.Tools.Enabled {
		return nil, fmt.Errorf("tools are disabled")
	}
	if !s.isAllowedTool(name) {
		return nil, fmt.Errorf("tool %q is not enabled", name)
	}
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	return s.tools.Execute(ctx, name, args)
}

// AllowedTools returns the server's configured tool allowlist.
func (s *Service) AllowedTools() []string {
	return append([]string{}, s.cfg.Tools.Allowed...)
}

// ResolveSelection fills in defaults and validates a provider/model pair.
func (s *Service) ResolveSelection(providerName, modelID string) (string, string, error) {
	if providerName == "" {
		providerName = s.cfg.DefaultProvider
	}
	if modelID == "" {
		model, ok := s.cfg.DefaultModel(providerName)
		if !ok {
			return "", "", fmt.Errorf("no default model configured for provider %q", providerName)
		}
		modelID = model.ID
	}
	if _, _, ok := s.cfg.FindModel(providerName, modelID); !ok {
		return "", "", fmt.Errorf("unknown provider/model selection")
	}
	return providerName, modelID, nil
}

// SupportsTools reports whether the selected model can be given tool specs.
func (s *Service) SupportsTools(providerName, modelID string) bool {
	_, modelCfg, ok := s.cfg.FindModel(providerName, modelID)
	return ok && modelCfg.SupportsTools
}

// BaseSystemPrompt returns the configured system prompt, or a profile's content
// when the named profile exists.
func (s *Service) BaseSystemPrompt(profile string) string {
	if profile != "" && s.profiles != nil {
		if content, ok := s.profiles.Get(profile); ok {
			return content
		}
	}
	return s.cfg.Chat.SystemPrompt
}

func (s *Service) isAllowedTool(name string) bool {
	for _, allowed := range s.cfg.Tools.Allowed {
		if allowed == name {
			return true
		}
	}
	return false
}

func (s *Service) filterAllowed(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if s.isAllowedTool(name) {
			out = append(out, name)
		}
	}
	return out
}

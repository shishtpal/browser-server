package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/ai/chat"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

// DecisionStatus values the agent may report.
const (
	DecisionContinue  = "continue"
	DecisionCompleted = "completed"
	DecisionBlocked   = "blocked"
)

// AgentDecision is the structured verdict the model must emit to end a task.
// Completion is never inferred from prose: a model that says "I'm done" in a
// sentence has not ended the task, because that phrasing also appears in
// progress narration and would terminate tasks that are still mid-plan.
type AgentDecision struct {
	Status   string `json:"status"`
	Response string `json:"response"`
	NextTask *struct {
		Prompt         string `json:"prompt"`
		ConversationID string `json:"conversation_id,omitempty"`
	} `json:"next_task,omitempty"`
}

// ErrBlocked is the terminal failure for a task the agent cannot advance.
// It is not transient: retrying a blocked task burns tokens on the same wall.
var ErrBlocked = fmt.Errorf("agent reported the task as blocked")

// decisionProtocol is appended to the system prompt. The model needs an
// unambiguous, machine-checkable way to signal termination, and a fenced JSON
// object is the only channel that survives every provider's text formatting.
const decisionProtocol = `

AUTONOMOUS TASK MODE
You are executing a background task with no user available to answer questions.
Work in small steps. Call tools when you need information or need to act.

When — and only when — you have nothing further to do, reply with a single JSON
object and no other text:

{"status": "completed", "response": "<summary of what was accomplished>"}

If you cannot make further progress (missing credentials, ambiguous
requirements, a tool that is unavailable), reply with:

{"status": "blocked", "response": "<what is blocking you>"}

To hand off follow-up work, include "next_task": {"prompt": "..."} alongside a
completed status. Do not ask the user questions; there is nobody to answer.
Never emit the JSON object while you still intend to call another tool.`

// ChatAgent implements Agent on top of the chat service. Each RunStep is exactly
// one model call plus the tool calls that call requests, which is what keeps the
// unit of work small enough to checkpoint.
type ChatAgent struct {
	Service *chat.Service
	// Store backs the idempotency ledger. A checkpoint records completed tool
	// calls too, but the ledger survives the exact crash window the checkpoint
	// cannot: between a tool's side effect and the checkpoint write.
	Store  *store.Store
	Config Config
}

// NewChatAgent builds the default agent implementation.
func NewChatAgent(service *chat.Service, st *store.Store, cfg Config) *ChatAgent {
	cfg.Normalize()
	return &ChatAgent{Service: service, Store: st, Config: cfg}
}

// RunStep executes one bounded step against the checkpointed conversation.
func (a *ChatAgent) RunStep(ctx context.Context, task *store.Task, checkpoint *ConversationCheckpoint, progress *Progress) (StepResult, error) {
	if a.Service == nil {
		return StepResult{}, fmt.Errorf("chat service is unavailable")
	}
	providerName, modelID, err := a.Service.ResolveSelection(a.Config.Provider, a.Config.Model)
	if err != nil {
		return StepResult{}, err
	}

	systemPrompt := a.Service.BaseSystemPrompt("") + decisionProtocol
	messages := checkpoint.ProviderMessages(systemPrompt)

	// Seed the transcript on the first step, or tell the model it is resuming.
	if !checkpoint.Resumed() {
		seed := provider.Message{Role: "user", Content: task.Prompt}
		checkpoint.Append(seed)
		messages = append(messages, seed)
	} else if notice := checkpoint.ResumePrompt(); notice != "" && checkpoint.Step > 0 && lastRole(checkpoint) == "assistant" {
		// Only inject the notice where the transcript would otherwise end on an
		// assistant turn with nothing prompting the next one.
		resume := provider.Message{Role: "user", Content: notice}
		checkpoint.Append(resume)
		messages = append(messages, resume)
	}

	var toolNames []string
	if a.Config.ToolsEnabled && a.Service.SupportsTools(providerName, modelID) {
		toolNames = a.Service.AllowedTools()
	}

	resp, err := a.Service.AgentStep(ctx, chat.AgentStepRequest{
		Provider:   providerName,
		Model:      modelID,
		Messages:   messages,
		ToolNames:  toolNames,
		OnProgress: progress.Mark,
	})
	if err != nil {
		return StepResult{}, err
	}

	if len(resp.ToolCalls) > 0 {
		return a.runToolCalls(ctx, task, checkpoint, resp, progress)
	}

	checkpoint.Append(provider.Message{Role: "assistant", Content: resp.Content})

	decision, ok := parseDecision(resp.Content)
	if !ok {
		// No verdict yet: the model narrated without calling a tool. Nudge it
		// and keep going — the step budget bounds how long this can repeat.
		checkpoint.Append(provider.Message{
			Role:    "user",
			Content: "Continue working toward the goal. Call a tool, or reply with the JSON decision object if the task is finished or blocked.",
		})
		encoded, encodeErr := checkpoint.Encode()
		if encodeErr != nil {
			return StepResult{}, encodeErr
		}
		return StepResult{Checkpoint: encoded}, nil
	}

	switch decision.Status {
	case DecisionCompleted:
		encoded, encodeErr := checkpoint.Encode()
		if encodeErr != nil {
			return StepResult{}, encodeErr
		}
		output, _ := json.Marshal(map[string]any{
			"status":   DecisionCompleted,
			"response": decision.Response,
			"steps":    checkpoint.Step + 1,
		})
		result := StepResult{Checkpoint: encoded, Output: output, Done: true}
		if decision.NextTask != nil && strings.TrimSpace(decision.NextTask.Prompt) != "" {
			conversationID := decision.NextTask.ConversationID
			if conversationID == "" {
				conversationID = task.ConversationID
			}
			result.NextTask = &store.NewTask{
				ConversationID: conversationID,
				Prompt:         decision.NextTask.Prompt,
				MaxAttempts:    a.Config.MaxAttempts,
			}
		}
		return result, nil
	case DecisionBlocked:
		reason := strings.TrimSpace(decision.Response)
		if reason == "" {
			reason = "no reason given"
		}
		return StepResult{}, fmt.Errorf("%w: %s", ErrBlocked, reason)
	default:
		encoded, encodeErr := checkpoint.Encode()
		if encodeErr != nil {
			return StepResult{}, encodeErr
		}
		return StepResult{Checkpoint: encoded}, nil
	}
}

// runToolCalls executes every tool the model requested, records each under its
// idempotency key, and checkpoints the results.
func (a *ChatAgent) runToolCalls(ctx context.Context, task *store.Task, checkpoint *ConversationCheckpoint, resp chat.AgentStepResponse, progress *Progress) (StepResult, error) {
	calls := make([]provider.ToolCall, 0, len(resp.ToolCalls))
	for _, call := range resp.ToolCalls {
		if call.ID == "" {
			call.ID = store.NewID("call")
		}
		calls = append(calls, call)
	}
	checkpoint.Append(provider.Message{Role: "assistant", Content: resp.Content, ToolCalls: calls})

	for _, call := range calls {
		key := IdempotencyKey(task.ID, checkpoint.Step, call.ID)
		content, replayed := checkpoint.CompletedToolCall(key)
		if !replayed {
			// The checkpoint may predate the crash that lost it; the ledger is
			// the durable record of what actually ran.
			stored, found, lookupErr := a.lookupToolCall(ctx, task.ID, key)
			if lookupErr != nil {
				return StepResult{}, lookupErr
			}
			if found {
				content, replayed = stored, true
			}
		}
		if !replayed {
			content = a.executeTool(ctx, call)
			if _, _, err := a.recordToolCall(ctx, task.ID, key, content); err != nil {
				return StepResult{}, err
			}
		}
		checkpoint.RecordToolCall(key, content)
		checkpoint.Append(provider.Message{Role: "tool", ToolCallID: call.ID, Content: content})
		// A step with many slow tools is working, not wedged.
		progress.Mark()
	}

	encoded, err := checkpoint.Encode()
	if err != nil {
		return StepResult{}, err
	}
	return StepResult{Checkpoint: encoded}, nil
}

// executeTool runs one tool, converting failures into a result the model can
// react to. A tool error is information for the agent, not a task failure — the
// model may well recover by calling something else.
func (a *ChatAgent) executeTool(ctx context.Context, call provider.ToolCall) string {
	args := json.RawMessage(call.Arguments)
	if len(strings.TrimSpace(call.Arguments)) == 0 {
		args = json.RawMessage("{}")
	}
	result, err := a.Service.ExecuteTool(ctx, call.Name, args)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(payload)
	}
	return string(result)
}

func (a *ChatAgent) lookupToolCall(ctx context.Context, taskID, key string) (string, bool, error) {
	if a.Store == nil {
		return "", false, nil
	}
	return a.Store.LookupToolCall(ctx, taskID, key)
}

func (a *ChatAgent) recordToolCall(ctx context.Context, taskID, key, result string) (string, bool, error) {
	if a.Store == nil {
		return result, false, nil
	}
	return a.Store.RecordToolCall(ctx, taskID, key, result)
}

// parseDecision extracts the structured verdict from a model reply. Providers
// wrap JSON in prose or code fences often enough that a strict Unmarshal on the
// whole body would misclassify most valid completions as "keep going".
func parseDecision(content string) (AgentDecision, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return AgentDecision{}, false
	}
	candidates := []string{trimmed}
	if fenced := extractFencedJSON(trimmed); fenced != "" {
		candidates = append(candidates, fenced)
	}
	if object := extractLastJSONObject(trimmed); object != "" {
		candidates = append(candidates, object)
	}
	for _, candidate := range candidates {
		var decision AgentDecision
		if err := json.Unmarshal([]byte(candidate), &decision); err != nil {
			continue
		}
		switch decision.Status {
		case DecisionContinue, DecisionCompleted, DecisionBlocked:
			return decision, true
		}
	}
	return AgentDecision{}, false
}

func extractFencedJSON(content string) string {
	start := strings.Index(content, "```")
	if start < 0 {
		return ""
	}
	rest := content[start+3:]
	if newline := strings.IndexByte(rest, '\n'); newline >= 0 {
		rest = rest[newline+1:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// extractLastJSONObject returns the final balanced {...} span, which is where a
// model that narrates before deciding puts its verdict.
func extractLastJSONObject(content string) string {
	end := strings.LastIndexByte(content, '}')
	if end < 0 {
		return ""
	}
	depth := 0
	for i := end; i >= 0; i-- {
		switch content[i] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				return content[i : end+1]
			}
		}
	}
	return ""
}

func lastRole(checkpoint *ConversationCheckpoint) string {
	if len(checkpoint.Messages) == 0 {
		return ""
	}
	return checkpoint.Messages[len(checkpoint.Messages)-1].Role
}

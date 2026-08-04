package tasks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

// CheckpointSchemaVersion is bumped whenever the checkpoint layout changes in a
// way older code cannot read. Checkpoints outlive deploys, so a resume that
// silently misparses an old checkpoint is worse than one that refuses to run.
const CheckpointSchemaVersion = 1

// ConversationCheckpoint is the durable resume point for a task. It holds enough
// to reconstruct the conversation and to know which side effects already landed.
type ConversationCheckpoint struct {
	SchemaVersion int `json:"schema_version"`
	// Messages is the reconstructed provider conversation, excluding the system
	// prompt (which is rebuilt from live config on resume so prompt edits take
	// effect without invalidating in-flight tasks).
	Messages []CheckpointMessage `json:"messages"`
	// CompletedToolCalls maps an idempotency key to the result already produced.
	// Any tool call whose key appears here must not run again.
	CompletedToolCalls map[string]string `json:"completed_tool_calls,omitempty"`
	CurrentGoal        string            `json:"current_goal"`
	Step               int               `json:"step"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// CheckpointMessage is a provider message in its persisted form. Image parts are
// intentionally not carried: they are rebuilt from attachment records, and
// inlining base64 data URLs would blow past MaxCheckpointBytes within a few steps.
type CheckpointMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	ToolCalls  []CheckpointToolCall `json:"tool_calls,omitempty"`
}

type CheckpointToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewCheckpoint returns the initial checkpoint for a task that has not run yet.
func NewCheckpoint(goal string) *ConversationCheckpoint {
	return &ConversationCheckpoint{
		SchemaVersion:      CheckpointSchemaVersion,
		CompletedToolCalls: map[string]string{},
		CurrentGoal:        goal,
		Step:               0,
		UpdatedAt:          time.Now().UTC(),
	}
}

// LoadCheckpoint decodes a persisted checkpoint. A task with no checkpoint yet
// gets a fresh one seeded from its prompt.
func LoadCheckpoint(task *store.Task) (*ConversationCheckpoint, error) {
	if len(task.Checkpoint) == 0 {
		return NewCheckpoint(task.Prompt), nil
	}
	var checkpoint ConversationCheckpoint
	if err := json.Unmarshal(task.Checkpoint, &checkpoint); err != nil {
		return nil, fmt.Errorf("decode checkpoint: %w", err)
	}
	if checkpoint.SchemaVersion != CheckpointSchemaVersion {
		return nil, fmt.Errorf("checkpoint schema v%d is not readable by this build (expected v%d)",
			checkpoint.SchemaVersion, CheckpointSchemaVersion)
	}
	if checkpoint.CompletedToolCalls == nil {
		checkpoint.CompletedToolCalls = map[string]string{}
	}
	if strings.TrimSpace(checkpoint.CurrentGoal) == "" {
		checkpoint.CurrentGoal = task.Prompt
	}
	return &checkpoint, nil
}

// Encode serializes the checkpoint for persistence.
func (c *ConversationCheckpoint) Encode() ([]byte, error) {
	c.SchemaVersion = CheckpointSchemaVersion
	c.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	if len(data) > store.MaxCheckpointBytes {
		return nil, fmt.Errorf("checkpoint would exceed %d bytes; the agent is accumulating context it should summarize", store.MaxCheckpointBytes)
	}
	return data, nil
}

// Resumed reports whether this checkpoint represents work already in progress.
func (c *ConversationCheckpoint) Resumed() bool {
	return c.Step > 0 || len(c.Messages) > 0
}

// ProviderMessages renders the checkpoint as provider messages, prepending the
// supplied system prompt.
func (c *ConversationCheckpoint) ProviderMessages(systemPrompt string) []provider.Message {
	out := make([]provider.Message, 0, len(c.Messages)+1)
	if systemPrompt != "" {
		out = append(out, provider.Message{Role: "system", Content: systemPrompt})
	}
	for _, m := range c.Messages {
		message := provider.Message{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			message.ToolCalls = append(message.ToolCalls, provider.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
		}
		out = append(out, message)
	}
	return out
}

// Append records a provider message into the checkpoint.
func (c *ConversationCheckpoint) Append(m provider.Message) {
	entry := CheckpointMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
	for _, tc := range m.ToolCalls {
		entry.ToolCalls = append(entry.ToolCalls, CheckpointToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
	}
	c.Messages = append(c.Messages, entry)
}

// IdempotencyKey derives the deterministic key for one tool call. Execution is
// at-least-once — a crash can land after a tool ran but before its result was
// checkpointed — so every side-effecting call is guarded by this key.
func IdempotencyKey(taskID string, step int, toolCallID string) string {
	return taskID + ":" + strconv.Itoa(step) + ":" + toolCallID
}

// CompletedToolCall returns the recorded result for a key, if the call already ran.
func (c *ConversationCheckpoint) CompletedToolCall(key string) (string, bool) {
	result, ok := c.CompletedToolCalls[key]
	return result, ok
}

// RecordToolCall marks a tool call as executed so a resume will not repeat it.
func (c *ConversationCheckpoint) RecordToolCall(key, result string) {
	if c.CompletedToolCalls == nil {
		c.CompletedToolCalls = map[string]string{}
	}
	c.CompletedToolCalls[key] = result
}

// resumeNotice is prepended on resume so the model knows it is continuing rather
// than starting over. Without it the model re-plans work whose results are
// already in the transcript, and burns its step budget redoing them.
const resumeNotice = "The previous session was interrupted and is now resuming. " +
	"The messages above are the prior transcript, including results from tool calls that already completed. " +
	"Do not repeat completed work; continue from the next unfinished step toward the stated goal."

// ResumePrompt returns the notice to inject when continuing an interrupted task,
// or "" for a fresh start.
func (c *ConversationCheckpoint) ResumePrompt() string {
	if !c.Resumed() {
		return ""
	}
	notice := resumeNotice
	if c.CurrentGoal != "" {
		notice += "\n\nGoal: " + c.CurrentGoal
	}
	return notice
}

package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
)

// processToolCalls runs a single response's tool-calls through authorization,
// approval/ask_questions/search/registry dispatch, and persists each result.
// The behaviour matches the original inline loop in SubmitStream.
func (s *Service) processToolCalls(
	generationCtx context.Context,
	conversationID string,
	assistantMessageID string,
	chatReq *provider.ChatRequest,
	resp *provider.ChatResponse,
	loadedAtResponseStart map[string]bool,
	activeToolSet map[string]bool,
	loadedToolSet map[string]bool,
	basePrompt string,
	sessionSkills []*skills.Skill,
	req *SubmitRequest,
	emit func(Event) error,
) ([]store.Message, error) {
	var toolMessages []store.Message
	for callIndex, call := range resp.ToolCalls {
		if call.ID == "" {
			call.ID = store.NewID("call")
			resp.ToolCalls[callIndex].ID = call.ID
			chatReq.Messages[len(chatReq.Messages)-1].ToolCalls[callIndex].ID = call.ID
		}

		authorized := isToolCallable(call.Name, activeToolSet, loadedAtResponseStart, req.IncludeAllToolDefinitions)

		// Intercept skill meta-tools (they modify session state, not executed by registry)
		if authorized {
			if skillResult, handled := s.handleSkillTool(call, sessionSkills, basePrompt, chatReq, req.ActiveTools, &activeToolSet, loadedToolSet, req.IncludeAllToolDefinitions); handled {
				// Update sessionSkills from the handler
				sessionSkills = s.getUpdatedSessionSkills(call, sessionSkills)
				toolContentBytes, _ := json.Marshal(map[string]any{
					"tool": call.Name, "args": json.RawMessage(call.Arguments), "result": toolResultField(skillResult), "decision": "approved",
				})
				toolMsg, addErr := s.store.AddMessage(generationCtx, conversationID, "tool", string(toolContentBytes), "completed", call.ID)
				if addErr != nil {
					return toolMessages, addErr
				}
				toolMessages = append(toolMessages, toolMsg)
				chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "tool", ToolCallID: call.ID, Content: string(skillResult)})
				if emit != nil {
					_ = emit(Event{Type: "tool_result", MessageID: assistantMessageID, ToolCall: &call, Content: string(toolContentBytes), Status: "completed"})
				}
				continue
			}
		}

		approved := authorized && (req.YOLOMode || call.Name == tools.SearchToolName)
		var pending pendingToolCall
		var err error
		if authorized && !approved {
			pending, err = s.beginToolApproval(conversationID, call.ID)
			if err != nil {
				return toolMessages, err
			}
		}
		if emit != nil {
			status := "approved"
			if !authorized {
				status = "error"
			} else if !approved {
				status = "pending"
			}
			if emitErr := emit(Event{Type: "tool_call", MessageID: assistantMessageID, ToolCall: &call, Status: status}); emitErr != nil {
				s.removePendingToolCall(call.ID)
				return toolMessages, emitErr
			}
		}
		var comment string
		if authorized && !approved {
			var dErr error
			approved, comment, dErr = s.waitForToolDecision(generationCtx, call.ID, pending)
			if dErr != nil {
				return toolMessages, dErr
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
		} else if call.Name == "ask_questions" && comment != "" {
			request, validationErr := tools.ValidateQuestionArguments(json.RawMessage(call.Arguments))
			if validationErr != nil {
				toolErr = validationErr
				toolStatus = "error"
				result, _ = json.Marshal(map[string]string{"error": validationErr.Error()})
				providerToolContent = string(result)
			} else {
				decision = "answered"
				result = questionAnswerResult(request, comment)
				providerToolContent = string(result)
			}
		} else if comment != "" {
			// User supplied feedback instead of running the tool; feed the
			// comment back to the model as the tool result so it can adjust.
			decision = "commented"
			result, _ = json.Marshal(map[string]string{"comment": comment})
			providerToolContent = comment
		} else if approved {
			if call.Name == tools.SearchToolName {
				var searchResult tools.SearchToolResult
				searchResult, toolErr = s.tools.Search(json.RawMessage(call.Arguments))
				if toolErr == nil {
					for i := range searchResult.Matches {
						match := &searchResult.Matches[i]
						match.Active = activeToolSet[match.Name]
						if match.Active {
							match.Loaded = true
							loadedToolSet[match.Name] = true
							searchResult.Loaded = append(searchResult.Loaded, match.Name)
						}
					}
					activeToolSet = s.configureToolDefinitions(
						chatReq,
						s.activeToolNames(activeToolSet),
						loadedToolSet,
						req.IncludeAllToolDefinitions,
					)
					result, toolErr = json.Marshal(searchResult)
				}
			} else {
				result, toolErr = s.tools.Execute(generationCtx, call.Name, json.RawMessage(call.Arguments))
			}
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
			"tool": call.Name, "args": displayArgs, "result": toolResultField(result), "decision": decision,
		})
		if marshalErr != nil {
			return toolMessages, marshalErr
		}
		toolContent := string(toolContentBytes)
		toolMsg, addErr := s.store.AddMessage(generationCtx, conversationID, "tool", toolContent, toolStatus, call.ID)
		if addErr != nil {
			return toolMessages, addErr
		}
		toolMessages = append(toolMessages, toolMsg)
		chatReq.Messages = append(chatReq.Messages, provider.Message{Role: "tool", ToolCallID: call.ID, Content: providerToolContent})
		if emit != nil {
			_ = emit(Event{Type: "tool_result", MessageID: assistantMessageID, ToolCall: &call, Content: toolContent, Status: toolStatus})
		}
	}
	return toolMessages, nil
}

// toolResultField converts a tool result for storage/emission as a JSON value
// when possible, falling back to a plain string. Tool results can be arbitrary
// text in raw-output mode (e.g. a git diff or directory tree), which is not
// valid JSON and therefore cannot be embedded as json.RawMessage.
func toolResultField(result []byte) any {
	if len(result) == 0 {
		return ""
	}
	if json.Valid(result) {
		return json.RawMessage(result)
	}
	return string(result)
}

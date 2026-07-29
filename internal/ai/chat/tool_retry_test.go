package chat

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
)

func TestWithoutLastToolTurnRemovesWholeLatestTurn(t *testing.T) {
	messages := []provider.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "first"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "old", Name: "old_tool", Arguments: "{}"}}},
		{Role: "tool", ToolCallID: "old", Content: "old result"},
		{Role: "assistant", Content: "old answer"},
		{Role: "user", Content: "latest"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{
			{ID: "latest-1", Name: "first_tool", Arguments: "{}"},
			{ID: "latest-2", Name: "second_tool", Arguments: "{}"},
		}},
		{Role: "tool", ToolCallID: "latest-1", Content: "one"},
		{Role: "tool", ToolCallID: "latest-2", Content: "two"},
	}

	clean, ignored, ok := withoutLastToolTurn(messages)
	if !ok {
		t.Fatal("expected a tool turn to be removed")
	}
	if len(clean) != 6 || clean[len(clean)-1].Role != "user" || clean[len(clean)-1].Content != "latest" {
		t.Fatalf("clean messages = %#v", clean)
	}
	if len(ignored) != 2 || ignored[0].ID != "latest-1" || ignored[1].ID != "latest-2" {
		t.Fatalf("ignored calls = %#v", ignored)
	}
	if len(messages) != 9 {
		t.Fatal("input slice was modified")
	}
}

func TestWithoutLastToolTurnLeavesPayloadWithoutToolCallsUnchanged(t *testing.T) {
	messages := []provider.Message{{Role: "system", Content: "system"}, {Role: "user", Content: "hello"}}
	clean, ignored, ok := withoutLastToolTurn(messages)
	if ok || ignored != nil || len(clean) != len(messages) {
		t.Fatalf("clean=%#v ignored=%#v ok=%v", clean, ignored, ok)
	}
}

func TestManualToolRetryPromptsAndIgnoresLastTurn(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}}
	request := provider.ChatRequest{Messages: []provider.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "do work"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "write_file", Arguments: `{}`}}},
		{Role: "tool", ToolCallID: "call-1", Content: `{"ok":true}`},
	}}
	badRequest := &provider.Error{Code: "provider_error", Status: 502, Diagnostic: "upstream HTTP 400"}
	var eventTypes []string
	var eventMu sync.Mutex

	response, err := s.retryToolContinuation(
		context.Background(),
		"conversation-1",
		"message-1",
		false,
		&request,
		provider.ChatResponse{HTTPStatus: 400},
		badRequest,
		func(event Event) error {
			eventMu.Lock()
			eventTypes = append(eventTypes, event.Type)
			eventMu.Unlock()
			if event.Type == "tool_call" {
				if event.ToolCall == nil || event.ToolCall.Name != toolRetryName || event.Status != "pending" {
					t.Fatalf("unexpected retry prompt: %#v", event)
				}
				if decisionErr := s.DecideToolCall("conversation-1", event.ToolCall.ID, true, ""); decisionErr != nil {
					t.Fatalf("approve retry: %v", decisionErr)
				}
			}
			return nil
		},
		func() (provider.ChatResponse, error) {
			if len(request.Messages) != 2 {
				t.Fatalf("retry payload retained the failed tool turn: %#v", request.Messages)
			}
			for _, message := range request.Messages {
				if len(message.ToolCalls) > 0 || message.Role == "tool" {
					t.Fatalf("retry payload contains tool-call residue: %#v", request.Messages)
				}
			}
			return provider.ChatResponse{Content: "recovered", HTTPStatus: 200}, nil
		},
	)
	if err != nil || response.Content != "recovered" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	eventMu.Lock()
	defer eventMu.Unlock()
	if len(eventTypes) != 2 || eventTypes[0] != "tool_call" || eventTypes[1] != "tool_result" {
		t.Fatalf("events = %v", eventTypes)
	}
}

func TestManualToolRetryPromptsAgainAfterAnyFailure(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}}
	request := provider.ChatRequest{Messages: []provider.Message{
		{Role: "user", Content: "do work"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "tool", Arguments: `{}`}}},
		{Role: "tool", ToolCallID: "call-1", Content: "result"},
	}}
	firstErr := &provider.Error{Code: "provider_error", Status: 502, Diagnostic: "HTTP 400"}
	secondErr := errors.New("non-provider failure")
	prompts := 0
	attempts := 0

	_, err := s.retryToolContinuation(
		context.Background(), "conversation-1", "message-1", false, &request,
		provider.ChatResponse{HTTPStatus: 400}, firstErr,
		func(event Event) error {
			if event.Type != "tool_call" {
				return nil
			}
			prompts++
			approved := prompts == 1
			return s.DecideToolCall("conversation-1", event.ToolCall.ID, approved, "")
		},
		func() (provider.ChatResponse, error) {
			attempts++
			return provider.ChatResponse{HTTPStatus: 418}, secondErr
		},
	)
	if !errors.Is(err, secondErr) {
		t.Fatalf("err=%v, want second failure", err)
	}
	if prompts != 2 || attempts != 1 {
		t.Fatalf("prompts=%d attempts=%d, want prompts=2 attempts=1", prompts, attempts)
	}
}

func TestYOLOToolRetryRunsEveryDelayForAllFailures(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}, toolRetryDelay: time.Millisecond}
	request := provider.ChatRequest{Messages: []provider.Message{
		{Role: "user", Content: "do work"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "tool", Arguments: `{}`}}},
		{Role: "tool", ToolCallID: "call-1", Content: "result"},
	}}
	attempts := 0
	started := time.Now()
	response, err := s.retryToolContinuation(
		context.Background(), "conversation-1", "message-1", true, &request,
		provider.ChatResponse{HTTPStatus: 400}, errors.New("bad request"), nil,
		func() (provider.ChatResponse, error) {
			attempts++
			if attempts < 3 {
				return provider.ChatResponse{HTTPStatus: 400}, errors.New("still bad")
			}
			return provider.ChatResponse{Content: "ok", HTTPStatus: 200}, nil
		},
	)
	if err != nil || response.Content != "ok" || attempts != 3 {
		t.Fatalf("attempts=%d response=%#v err=%v", attempts, response, err)
	}
	if elapsed := time.Since(started); elapsed < 3*time.Millisecond {
		t.Fatalf("YOLO retries did not wait before every attempt: %v", elapsed)
	}
	if len(request.Messages) != 1 || request.Messages[0].Role != "user" {
		t.Fatalf("retry payload = %#v", request.Messages)
	}
}

func TestYOLOToolRetryStopsDuringDelay(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}, toolRetryDelay: time.Hour}
	request := provider.ChatRequest{Messages: []provider.Message{
		{Role: "user", Content: "do work"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "tool"}}},
		{Role: "tool", ToolCallID: "call-1", Content: "result"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.retryToolContinuation(
		ctx, "conversation-1", "message-1", true, &request,
		provider.ChatResponse{HTTPStatus: 400}, errors.New("bad request"), nil,
		func() (provider.ChatResponse, error) {
			t.Fatal("completion should not run after cancellation")
			return provider.ChatResponse{}, nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context canceled", err)
	}
}

type toolRetryIntegrationClient struct {
	t       *testing.T
	attempt int
}

func (c *toolRetryIntegrationClient) Complete(context.Context, provider.ChatRequest) (provider.ChatResponse, error) {
	c.t.Fatal("streaming client expected")
	return provider.ChatResponse{}, nil
}

func (c *toolRetryIntegrationClient) Stream(_ context.Context, request provider.ChatRequest, _ func(provider.Event) error) (provider.ChatResponse, error) {
	c.attempt++
	switch c.attempt {
	case 1:
		return provider.ChatResponse{HTTPStatus: 200, ToolCalls: []provider.ToolCall{{
			ID: "call-1", Name: "get_current_time", Arguments: `{}`,
		}}}, nil
	case 2:
		if len(request.Messages) < 4 || request.Messages[len(request.Messages)-1].Role != "tool" {
			c.t.Fatalf("tool continuation payload = %#v", request.Messages)
		}
		return provider.ChatResponse{HTTPStatus: 400}, &provider.Error{
			Code: "provider_error", Status: 502, Diagnostic: "upstream HTTP 400",
		}
	case 3:
		for _, message := range request.Messages {
			if message.Role == "tool" || len(message.ToolCalls) > 0 {
				c.t.Fatalf("recovery payload retained last tool turn: %#v", request.Messages)
			}
		}
		return provider.ChatResponse{HTTPStatus: 200, Content: "recovered"}, nil
	default:
		c.t.Fatalf("unexpected provider attempt %d", c.attempt)
		return provider.ChatResponse{}, nil
	}
}

func TestSubmitStreamRecoversHTTP400AfterToolCall(t *testing.T) {
	temp := t.TempDir()
	st, err := store.Open(filepath.Join(temp, "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	profileRegistry, err := profiles.Load(temp)
	if err != nil {
		t.Fatal(err)
	}
	skillRegistry, err := skills.Load(temp)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Path:            filepath.Join(temp, "bs-ai-config.json"),
		DefaultProvider: "test",
		Providers: map[string]config.ProviderConfig{
			"test": {
				BaseURL:               "http://localhost",
				APIKey:                "test",
				RequestTimeoutSeconds: 1,
				RetryDelaySeconds:     1,
				Models: []config.ModelConfig{{
					ID: "model", SupportsTools: true, MaxOutputTokens: 100,
				}},
			},
		},
		Tools: config.ToolsConfig{
			Enabled: true, Allowed: []string{"get_current_time"}, MaxIterations: 5,
		},
		Chat: config.ChatConfig{SystemPrompt: "system", MaxHistoryMessages: 10},
	}
	conversation, err := st.CreateConversation(context.Background(), "test", "test", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(cfg, st, profileRegistry, skillRegistry)
	client := &toolRetryIntegrationClient{t: t}
	service.clients["test"] = client
	prompts := 0

	result, err := service.SubmitStream(context.Background(), conversation.ID, SubmitRequest{
		Content: "what time is it?", ToolsEnabled: true, ActiveTools: []string{"get_current_time"},
	}, func(event Event) error {
		if event.Type == "tool_call" && event.Status == "pending" {
			prompts++
			return service.DecideToolCall(conversation.ID, event.ToolCall.ID, true, "")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantMessage.Content != "recovered" || result.AssistantMessage.Status != "completed" {
		t.Fatalf("assistant = %#v", result.AssistantMessage)
	}
	if client.attempt != 3 || prompts != 2 {
		t.Fatalf("provider attempts=%d prompts=%d, want 3 and 2", client.attempt, prompts)
	}
	if len(result.ToolMessages) != 1 || result.ToolMessages[0].ToolCallID != "call-1" {
		t.Fatalf("tool messages = %#v", result.ToolMessages)
	}
}

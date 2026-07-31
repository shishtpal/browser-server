package chat

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
)

type appendIntegrationClient struct {
	t       *testing.T
	attempt int
}

func (c *appendIntegrationClient) Complete(context.Context, provider.ChatRequest) (provider.ChatResponse, error) {
	c.t.Fatal("streaming client expected")
	return provider.ChatResponse{}, nil
}

func (c *appendIntegrationClient) Stream(_ context.Context, request provider.ChatRequest, _ func(provider.Event) error) (provider.ChatResponse, error) {
	c.attempt++
	if c.attempt == 1 {
		return provider.ChatResponse{HTTPStatus: 200, ToolCalls: []provider.ToolCall{{
			ID: "call-append", Name: "get_current_time", Arguments: `{}`,
		}}}, nil
	}
	if c.attempt != 2 {
		c.t.Fatalf("unexpected provider attempt %d", c.attempt)
	}
	messages := request.Messages
	if len(messages) < 6 {
		c.t.Fatalf("continuation messages = %#v", messages)
	}
	tail := messages[len(messages)-4:]
	if tail[0].Role != "assistant" || len(tail[0].ToolCalls) != 1 ||
		tail[1].Role != "tool" ||
		tail[2].Role != "user" || tail[2].Content != "first addition" ||
		tail[3].Role != "user" || tail[3].Content != "second addition" {
		c.t.Fatalf("invalid append ordering: %#v", tail)
	}
	return provider.ChatResponse{HTTPStatus: 200, Content: "used additions"}, nil
}

func TestSubmitStreamAppendsContextAfterToolResults(t *testing.T) {
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
				BaseURL: "http://localhost", APIKey: "test", RequestTimeoutSeconds: 1, RetryDelaySeconds: 1,
				Models: []config.ModelConfig{{ID: "model", SupportsTools: true, MaxOutputTokens: 100}},
			},
		},
		Tools: config.ToolsConfig{Enabled: true, Allowed: []string{"get_current_time"}, MaxIterations: 5},
		Chat:  config.ChatConfig{SystemPrompt: "system", MaxHistoryMessages: 10},
	}
	conversation, err := st.CreateConversation(context.Background(), "test", "test", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(cfg, st, profileRegistry, skillRegistry)
	client := &appendIntegrationClient{t: t}
	service.clients["test"] = client
	var windowEvents []string

	result, err := service.SubmitStream(context.Background(), conversation.ID, SubmitRequest{
		Content: "start", ToolsEnabled: true, ActiveTools: []string{"get_current_time"},
	}, func(event Event) error {
		switch event.Type {
		case "append_window":
			windowEvents = append(windowEvents, event.Status)
			if event.Status == "open" {
				if _, err := service.AppendMessage(context.Background(), conversation.ID, "first addition"); err != nil {
					return err
				}
				if _, err := service.AppendMessage(context.Background(), conversation.ID, "second addition"); err != nil {
					return err
				}
			}
		case "tool_call":
			if event.Status == "pending" {
				return service.DecideToolCall(conversation.ID, event.ToolCall.ID, true, "")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantMessage.Content != "used additions" || client.attempt != 2 {
		t.Fatalf("result=%#v attempts=%d", result.AssistantMessage, client.attempt)
	}
	if len(windowEvents) != 2 || windowEvents[0] != "open" || windowEvents[1] != "closed" {
		t.Fatalf("append window events = %v", windowEvents)
	}
	if _, err := service.AppendMessage(context.Background(), conversation.ID, "too late"); !errors.Is(err, ErrAppendWindowClosed) {
		t.Fatalf("late append error = %v", err)
	}
	_, messages, err := st.GetConversation(context.Background(), conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 5 || messages[len(messages)-1].ID != result.AssistantMessage.ID {
		t.Fatalf("persisted messages = %#v", messages)
	}
}

func TestWithoutLastToolTurnPreservesAppendedUsers(t *testing.T) {
	toolCall := provider.ToolCall{ID: "call-1", Name: "read_file", Arguments: `{}`}
	messages := []provider.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "initial"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{toolCall}},
		{Role: "tool", ToolCallID: toolCall.ID, Content: "result"},
		{Role: "user", Content: "append one"},
		{Role: "user", Content: "append two"},
	}
	clean, ignored, ok := withoutLastToolTurn(messages)
	if !ok || len(ignored) != 1 {
		t.Fatalf("ok=%v ignored=%#v", ok, ignored)
	}
	if len(clean) != 4 || clean[2].Content != "append one" || clean[3].Content != "append two" {
		t.Fatalf("clean messages = %#v", clean)
	}
}

func TestAppendWindowLimitsAndStop(t *testing.T) {
	temp := t.TempDir()
	st, err := store.Open(filepath.Join(temp, "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	conversation, err := st.CreateConversation(context.Background(), "test", "provider", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{
		store: st, active: map[string]context.CancelFunc{}, appendWindows: map[string]*appendWindow{},
	}
	if _, err := service.openAppendWindow(context.Background(), conversation.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxAppendMessages; i++ {
		if _, err := service.AppendMessage(context.Background(), conversation.ID, "context"); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	if _, err := service.AppendMessage(context.Background(), conversation.ID, "one too many"); !errors.Is(err, ErrAppendMessageLimit) {
		t.Fatalf("message limit error = %v", err)
	}
	service.Stop(conversation.ID)
	if _, err := service.AppendMessage(context.Background(), conversation.ID, "after stop"); !errors.Is(err, ErrAppendWindowClosed) {
		t.Fatalf("append after stop error = %v", err)
	}
}

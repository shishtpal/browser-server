package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
)

func TestProviderMessagesExcludeToolHistory(t *testing.T) {
	s := &Service{cfg: &config.Config{Chat: config.ChatConfig{MaxHistoryMessages: 2}}}
	messages := []store.Message{
		{Role: "user", Content: "older user message", Status: "completed"},
		{Role: "tool", Content: `{"tool":"read_file","result":"secret tool output"}`, ToolCallID: "call-1", Status: "completed"},
		{Role: "assistant", Content: "older assistant message", Status: "completed"},
		{Role: "user", Content: "latest user message", Status: "completed"},
	}

	got := s.providerMessages(messages, "system prompt")
	if len(got) != 3 {
		t.Fatalf("provider messages length = %d, want 3: %#v", len(got), got)
	}
	if got[0].Role != "system" || got[0].Content != "system prompt" {
		t.Fatalf("system message = %#v", got[0])
	}
	if got[1].Role != "assistant" || got[1].Content != "older assistant message" {
		t.Fatalf("first history message = %#v", got[1])
	}
	if got[2].Role != "user" || got[2].Content != "latest user message" {
		t.Fatalf("second history message = %#v", got[2])
	}
}

func TestResolveActiveToolsDistinguishesOmittedAndEmpty(t *testing.T) {
	s := &Service{cfg: &config.Config{Tools: config.ToolsConfig{Allowed: []string{"read_file", "write_file"}}}}
	if got := s.resolveActiveTools(nil, nil); len(got) != 2 {
		t.Fatalf("omitted active tools returned %v, want configured defaults", got)
	}
	if got := s.resolveActiveTools([]string{}, nil); len(got) != 0 {
		t.Fatalf("explicit empty active tools returned %v, want none", got)
	}
	got := s.resolveActiveTools([]string{"read_file", "not_allowed"}, nil)
	if len(got) != 1 || got[0] != "read_file" {
		t.Fatalf("filtered active tools = %v, want [read_file]", got)
	}
}

func TestConfigureToolDefinitionsUsesSearchByDefault(t *testing.T) {
	allowed := []string{tools.SearchToolName, "read_file", "write_file"}
	s := &Service{
		cfg:   &config.Config{Tools: config.ToolsConfig{Allowed: allowed}},
		tools: tools.New(tools.Options{Allowed: allowed}),
	}
	request := provider.ChatRequest{}
	active := s.configureToolDefinitions(&request, allowed, map[string]bool{}, false)
	if len(request.Tools) != 1 || request.Tools[0].Name != tools.SearchToolName {
		t.Fatalf("initial specs = %#v, want search_tool only", request.Tools)
	}
	if !active["read_file"] || !active["write_file"] {
		t.Fatalf("active execution set = %#v, want all active tools", active)
	}
}

func TestConfigureToolDefinitionsIncludesAllOrFallsBackWithoutSearch(t *testing.T) {
	allowed := []string{tools.SearchToolName, "read_file", "write_file"}
	s := &Service{
		cfg:   &config.Config{Tools: config.ToolsConfig{Allowed: allowed}},
		tools: tools.New(tools.Options{Allowed: allowed}),
	}
	request := provider.ChatRequest{}
	s.configureToolDefinitions(&request, allowed, map[string]bool{}, true)
	if len(request.Tools) != 3 {
		t.Fatalf("all-definition specs = %#v, want 3", request.Tools)
	}

	request.Tools = nil
	s.configureToolDefinitions(&request, []string{"read_file", "write_file"}, map[string]bool{}, false)
	if len(request.Tools) != 2 {
		t.Fatalf("fallback specs = %#v, want 2", request.Tools)
	}
}

func TestConfigureToolDefinitionsLoadsOnlyActiveMatches(t *testing.T) {
	allowed := []string{tools.SearchToolName, "read_file", "write_file"}
	s := &Service{
		cfg:   &config.Config{Tools: config.ToolsConfig{Allowed: allowed}},
		tools: tools.New(tools.Options{Allowed: allowed}),
	}
	request := provider.ChatRequest{}
	loaded := map[string]bool{"read_file": true, "write_file": true}
	s.configureToolDefinitions(&request, []string{tools.SearchToolName, "read_file"}, loaded, false)
	if len(request.Tools) != 2 || request.Tools[0].Name != tools.SearchToolName || request.Tools[1].Name != "read_file" {
		t.Fatalf("loaded specs = %#v, want search_tool and read_file", request.Tools)
	}
	if loaded["write_file"] {
		t.Fatal("inactive write_file remained loaded")
	}
}

func TestIsToolCallableRequiresDiscoveryInSearchMode(t *testing.T) {
	active := map[string]bool{tools.SearchToolName: true, "read_file": true, "write_file": false}
	loaded := map[string]bool{}
	if !isToolCallable(tools.SearchToolName, active, loaded, false) {
		t.Fatal("search_tool should be callable before discovery")
	}
	if isToolCallable("read_file", active, loaded, false) {
		t.Fatal("hidden read_file was callable before discovery")
	}
	loaded["read_file"] = true
	if !isToolCallable("read_file", active, loaded, false) {
		t.Fatal("discovered read_file should be callable")
	}
	if isToolCallable("write_file", active, map[string]bool{"write_file": true}, false) {
		t.Fatal("inactive write_file should remain blocked after discovery")
	}
	if !isToolCallable("read_file", active, map[string]bool{}, true) {
		t.Fatal("active read_file should be callable when all definitions are included")
	}
}

func TestRestrictedSkillToolsRetainSearchTool(t *testing.T) {
	s := &Service{cfg: &config.Config{Tools: config.ToolsConfig{Allowed: []string{tools.SearchToolName, "read_file"}}}}
	activeSkills := []*skills.Skill{{Tools: []string{"read_file"}}}
	got := s.resolveActiveTools(nil, activeSkills)
	set := make(map[string]bool, len(got))
	for _, name := range got {
		set[name] = true
	}
	if !set[tools.SearchToolName] || !set["read_file"] {
		t.Fatalf("restricted tools = %v, want search_tool and read_file", got)
	}
}

func TestToolDecisionIsScopedAndDelivered(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}}
	pending, err := s.beginToolApproval("conversation-1", "call-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DecideToolCall("conversation-2", "call-1", true, ""); !errors.Is(err, ErrToolCallNotPending) {
		t.Fatalf("expected scoped rejection, got %v", err)
	}
	if err := s.DecideToolCall("conversation-1", "call-1", false, ""); err != nil {
		t.Fatal(err)
	}
	approved, comment, err := s.waitForToolDecision(context.Background(), "call-1", pending)
	if err != nil || approved {
		t.Fatalf("approved=%v err=%v", approved, err)
	}
	if comment != "" {
		t.Fatalf("expected empty comment, got %q", comment)
	}
}

func TestToolDecisionStopsWaitingOnCancellation(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}}
	pending, err := s.beginToolApproval("conversation-1", "call-1")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, _, err := s.waitForToolDecision(ctx, "call-1", pending); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if err := s.DecideToolCall("conversation-1", "call-1", true, ""); !errors.Is(err, ErrToolCallNotPending) {
		t.Fatalf("expected pending call cleanup, got %v", err)
	}
}

func TestToolDecisionDeliversComment(t *testing.T) {
	s := &Service{pending: map[string]pendingToolCall{}}
	pending, err := s.beginToolApproval("conversation-1", "call-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DecideToolCall("conversation-1", "call-1", false, "use a different argument"); err != nil {
		t.Fatal(err)
	}
	approved, comment, err := s.waitForToolDecision(context.Background(), "call-1", pending)
	if err != nil || approved {
		t.Fatalf("approved=%v err=%v", approved, err)
	}
	if comment != "use a different argument" {
		t.Fatalf("expected comment delivered, got %q", comment)
	}
}

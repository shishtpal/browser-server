package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestBeginFinishAndDelete(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	c, err := s.CreateConversation(ctx, "test", "p", "m")
	if err != nil {
		t.Fatal(err)
	}
	_, a, err := s.BeginTurn(ctx, c.ID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishTurn(ctx, a.ID, "world", "completed", RequestLog{ConversationID: c.ID, MessageID: a.ID, Provider: "p", Model: "m", Endpoint: "x", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteConversation(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteConversation(ctx, c.ID); !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

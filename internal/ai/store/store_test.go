package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestBeginFinishAndDelete(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	c, err := s.CreateConversation(ctx, "test", "p", "m", "")
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

func TestForkConversation(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()

	src, err := s.CreateConversation(ctx, "Original", "prov", "model", "coder")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateConversationSkills(ctx, src.ID, []string{"debug"}); err != nil {
		t.Fatal(err)
	}

	// Turn 1
	u1, a1, err := s.BeginTurn(ctx, src.ID, "first question")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishTurn(ctx, a1.ID, "first answer", "completed", RequestLog{ConversationID: src.ID, MessageID: a1.ID, Provider: "prov", Model: "model", Endpoint: "x", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	// Turn 2
	_, a2, err := s.BeginTurn(ctx, src.ID, "second question")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishTurn(ctx, a2.ID, "second answer", "completed", RequestLog{ConversationID: src.ID, MessageID: a2.ID, Provider: "prov", Model: "model", Endpoint: "x", Status: "success"}); err != nil {
		t.Fatal(err)
	}

	// Fork up to and including the first assistant answer -> should copy exactly 2 messages.
	forked, err := s.ForkConversation(ctx, src.ID, a1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if forked.ID == src.ID {
		t.Fatal("forked conversation must have a new id")
	}
	if forked.Provider != "prov" || forked.Model != "model" || forked.Profile != "coder" {
		t.Fatalf("forked conversation did not inherit settings: %+v", forked)
	}
	if len(forked.Skills) != 1 || forked.Skills[0] != "debug" {
		t.Fatalf("forked conversation did not inherit skills: %+v", forked.Skills)
	}

	_, msgs, err := s.GetConversation(ctx, forked.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 copied messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "first question" {
		t.Fatalf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "first answer" {
		t.Fatalf("unexpected second message: %+v", msgs[1])
	}

	// Ensure source is untouched.
	_, srcMsgs, err := s.GetConversation(ctx, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(srcMsgs) != 4 {
		t.Fatalf("source conversation should still have 4 messages, got %d", len(srcMsgs))
	}
	_ = u1

	// Unknown message id -> not found.
	if _, err := s.ForkConversation(ctx, src.ID, "msg_missing"); !IsNotFound(err) {
		t.Fatalf("expected not found for unknown message, got %v", err)
	}
}

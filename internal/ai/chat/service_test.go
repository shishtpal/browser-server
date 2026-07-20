package chat

import (
	"context"
	"errors"
	"testing"
	"time"
)

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

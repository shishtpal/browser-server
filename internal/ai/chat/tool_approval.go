package chat

import (
	"context"
	"fmt"
)

// Tool-call approval bookkeeping. The pending map stays on Service to preserve
// existing test initialization patterns; helpers in this file operate on it
// directly so we don't introduce a second source of truth.

type toolDecision struct {
	approved bool
	comment  string
}

type pendingToolCall struct {
	conversationID string
	decision       chan toolDecision
}

func (s *Service) DecideToolCall(conversationID, callID string, approved bool, comment string) error {
	s.pendingMu.Lock()
	pending, ok := s.pending[callID]
	if ok && pending.conversationID == conversationID {
		delete(s.pending, callID)
	} else {
		ok = false
	}
	s.pendingMu.Unlock()
	if !ok {
		return ErrToolCallNotPending
	}
	pending.decision <- toolDecision{approved: approved, comment: comment}
	return nil
}

func (s *Service) beginToolApproval(conversationID, callID string) (pendingToolCall, error) {
	pending := pendingToolCall{conversationID: conversationID, decision: make(chan toolDecision, 1)}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if _, exists := s.pending[callID]; exists {
		return pendingToolCall{}, fmt.Errorf("duplicate tool call id")
	}
	s.pending[callID] = pending
	return pending, nil
}

func (s *Service) waitForToolDecision(ctx context.Context, callID string, pending pendingToolCall) (bool, string, error) {
	select {
	case decision := <-pending.decision:
		return decision.approved, decision.comment, nil
	case <-ctx.Done():
		s.removePendingToolCall(callID)
		return false, "", ctx.Err()
	}
}

func (s *Service) removePendingToolCall(callID string) {
	s.pendingMu.Lock()
	delete(s.pending, callID)
	s.pendingMu.Unlock()
}

package chat

import (
	"context"
	"sync"
	"time"
)

// Generation lifecycle helpers. These back Start/Stop/Close semantics on
// Service. The active map stays on Service to preserve existing test
// initialization patterns (`&Service{active: map[string]context.CancelFunc{}}`).

func (s *Service) start(conversationID string, cancel context.CancelFunc) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	if _, ok := s.active[conversationID]; ok {
		return false
	}
	s.active[conversationID] = cancel
	return true
}

func (s *Service) finish(conversationID string) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	delete(s.active, conversationID)
}

func (s *Service) IsActive(id string) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	_, ok := s.active[id]
	return ok
}

func (s *Service) Stop(conversationID string) bool {
	s.activeMu.Lock()
	cancel, ok := s.active[conversationID]
	if ok {
		cancel()
	}
	s.activeMu.Unlock()
	s.closeConversationAppendWindow(conversationID)
	return ok
}

func (s *Service) Close() {
	s.activeMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.active))
	for _, c := range s.active {
		cancels = append(cancels, c)
	}
	s.activeMu.Unlock()
	for _, c := range cancels {
		c()
	}
	s.closeAllAppendWindows()
	for i := 0; i < 50; i++ {
		s.activeMu.Lock()
		n := len(s.active)
		s.activeMu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Ensure unused imports don't break the build when tests compile against this
// package without directly using every symbol.
var _ = sync.Mutex{}

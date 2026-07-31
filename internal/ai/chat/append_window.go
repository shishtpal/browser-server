package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"browser-server/internal/ai/store"
)

const (
	maxAppendMessages    = 16
	maxAppendWindowBytes = maxMessageBytes
)

var (
	ErrAppendWindowClosed = errors.New("append window is closed")
	ErrAppendMessageLimit = errors.New("append window message limit reached")
	ErrAppendByteLimit    = errors.New("append window byte limit reached")
)

type appendWindow struct {
	mu       sync.Mutex
	closed   bool
	messages []store.Message
	bytes    int
}

func (s *Service) openAppendWindow(ctx context.Context, conversationID string) (*appendWindow, error) {
	s.appendMu.Lock()
	defer s.appendMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.appendWindows == nil {
		s.appendWindows = make(map[string]*appendWindow)
	}
	if _, exists := s.appendWindows[conversationID]; exists {
		return nil, fmt.Errorf("append window already open for conversation %s", conversationID)
	}
	window := &appendWindow{}
	s.appendWindows[conversationID] = window
	return window, nil
}

// closeAppendWindow atomically ends admission and returns every message that
// won the admission race. Callers decide whether those messages can be sent to
// the provider; accepted rows intentionally remain persisted on cancellation.
func (s *Service) closeAppendWindow(conversationID string, window *appendWindow) []store.Message {
	s.appendMu.Lock()
	if current := s.appendWindows[conversationID]; current == window {
		delete(s.appendWindows, conversationID)
	}
	s.appendMu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()
	window.closed = true
	messages := append([]store.Message(nil), window.messages...)
	window.messages = nil
	return messages
}

func (s *Service) closeConversationAppendWindow(conversationID string) {
	s.appendMu.Lock()
	window := s.appendWindows[conversationID]
	if window != nil {
		delete(s.appendWindows, conversationID)
	}
	s.appendMu.Unlock()
	if window == nil {
		return
	}
	window.mu.Lock()
	window.closed = true
	window.mu.Unlock()
}

func (s *Service) closeAllAppendWindows() {
	s.appendMu.Lock()
	windows := make([]*appendWindow, 0, len(s.appendWindows))
	for id, window := range s.appendWindows {
		delete(s.appendWindows, id)
		windows = append(windows, window)
	}
	s.appendMu.Unlock()
	for _, window := range windows {
		window.mu.Lock()
		window.closed = true
		window.mu.Unlock()
	}
}

func (s *Service) AppendMessage(ctx context.Context, conversationID, content string) (store.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return store.Message{}, fmt.Errorf("message content is required")
	}
	if len(content) > maxMessageBytes {
		return store.Message{}, fmt.Errorf("message content exceeds %d bytes", maxMessageBytes)
	}
	if _, _, err := s.store.GetConversation(ctx, conversationID); err != nil {
		return store.Message{}, err
	}

	s.appendMu.Lock()
	window := s.appendWindows[conversationID]
	s.appendMu.Unlock()
	if window == nil {
		return store.Message{}, ErrAppendWindowClosed
	}

	window.mu.Lock()
	defer window.mu.Unlock()
	if window.closed {
		return store.Message{}, ErrAppendWindowClosed
	}
	if len(window.messages) >= maxAppendMessages {
		return store.Message{}, ErrAppendMessageLimit
	}
	if window.bytes+len(content) > maxAppendWindowBytes {
		return store.Message{}, ErrAppendByteLimit
	}
	message, err := s.store.AddMessage(ctx, conversationID, "user", content, "completed", "")
	if err != nil {
		return store.Message{}, err
	}
	window.messages = append(window.messages, message)
	window.bytes += len(content)
	return message, nil
}

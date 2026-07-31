package chat

import (
	"strings"

	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

func (s *Service) providerMessages(messages []store.Message, systemPrompt string) []provider.Message {
	out := []provider.Message{{Role: "system", Content: systemPrompt}}
	history := make([]store.Message, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" || message.Role == "tool" || message.Status == "superseded" || message.Status == "pending" || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if message.Role == "assistant" && message.Status != "completed" && message.Status != "cancelled" && message.Status != "error" {
			continue
		}
		history = append(history, message)
	}

	start := 0
	limit := s.cfg.Chat.MaxHistoryMessages
	if limit > 0 && len(history) > limit {
		start = len(history) - limit
	}
	for _, message := range history[start:] {
		out = append(out, provider.Message{Role: message.Role, Content: message.Content})
	}

	return out
}

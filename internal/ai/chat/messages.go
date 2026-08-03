package chat

import (
	"context"
	"encoding/base64"
	"log"
	"strings"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/store"
)

func (s *Service) providerMessages(ctx context.Context, messages []store.Message, systemPrompt string) []provider.Message {
	out := []provider.Message{{Role: "system", Content: systemPrompt}}
	history := make([]store.Message, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" || message.Role == "tool" || message.Status == "superseded" || message.Status == "pending" {
			continue
		}
		if message.Role == "user" && strings.TrimSpace(message.Content) == "" && len(message.Attachments) == 0 {
			continue
		}
		if message.Role == "assistant" && message.Status != "completed" && message.Status != "cancelled" && message.Status != "error" {
			continue
		}
		if message.Role != "user" && strings.TrimSpace(message.Content) == "" {
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
		out = append(out, s.providerMessage(ctx, message))
	}

	return out
}

// providerMessage converts a persisted message into a provider message, adding
// OpenAI-compatible image content parts for user messages that carry
// attachments. Data URLs are built from validated private files at request
// time and are never stored or logged.
func (s *Service) providerMessage(ctx context.Context, message store.Message) provider.Message {
	pm := provider.Message{Role: message.Role, Content: message.Content}
	if message.Role != "user" || len(message.Attachments) == 0 {
		return pm
	}
	for _, att := range message.Attachments {
		data, err := attachments.Read(s.attachmentsDir, message.ConversationID, att.StorageKey)
		if err != nil {
			log.Printf("WARN: attachment %s (%s) unreadable: %v", att.ID, att.StorageKey, err)
			continue
		}
		pm.ImageParts = append(pm.ImageParts, provider.ImagePart{
			DataURL: "data:" + att.ContentType + ";base64," + base64.StdEncoding.EncodeToString(data),
		})
	}
	return pm
}

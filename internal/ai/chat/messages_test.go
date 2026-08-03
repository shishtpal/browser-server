package chat

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/config"
	"browser-server/internal/ai/store"
)

// encodePNG renders a tiny PNG so attachment files can be written for tests.
func encodePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestProviderMessageAttachesImageDataURLs(t *testing.T) {
	dir := t.TempDir()
	convID := "conv-1"
	data := encodePNG(t)
	storageKey, err := attachments.Write(dir, convID, "att-1", "image/png", data)
	if err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	s := &Service{cfg: &config.Config{}, attachmentsDir: dir}

	msg := store.Message{
		Role:           "user",
		Content:        "describe this",
		ConversationID: convID,
		Attachments: []store.Attachment{{
			ID:          "att-1",
			StorageKey:  storageKey,
			ContentType: "image/png",
		}},
	}
	pm := s.providerMessage(context.Background(), msg)
	if pm.Role != "user" || pm.Content != "describe this" {
		t.Fatalf("text content lost: %+v", pm)
	}
	if len(pm.ImageParts) != 1 {
		t.Fatalf("expected 1 image part, got %d", len(pm.ImageParts))
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	if pm.ImageParts[0].DataURL != want {
		t.Fatalf("data URL mismatch:\n got %s\nwant %s", pm.ImageParts[0].DataURL, want)
	}
}

func TestProviderMessageSkipsMissingAttachmentFiles(t *testing.T) {
	// If an attachment file is missing on disk, the message text is preserved
	// and the unreadable attachment is skipped (logged), not fatal.
	s := &Service{cfg: &config.Config{}, attachmentsDir: t.TempDir()}
	msg := store.Message{
		Role:           "user",
		Content:        "describe this",
		ConversationID: "conv-1",
		Attachments: []store.Attachment{{
			ID:          "att-missing",
			StorageKey:  "att-missing.png",
			ContentType: "image/png",
		}},
	}
	pm := s.providerMessage(context.Background(), msg)
	if pm.Content != "describe this" {
		t.Fatalf("text content should be preserved: %+v", pm)
	}
	if len(pm.ImageParts) != 0 {
		t.Fatalf("missing attachment should not produce an image part: %+v", pm.ImageParts)
	}
}

func TestProviderMessagesPreservesImageHistory(t *testing.T) {
	// History with images must flow through providerMessages unchanged in text
	// and carry the image parts; tool/assistant text messages stay text-only.
	dir := t.TempDir()
	convID := "conv-1"
	data := encodePNG(t)
	storageKey, err := attachments.Write(dir, convID, "att-1", "image/png", data)
	if err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	s := &Service{cfg: &config.Config{Chat: config.ChatConfig{MaxHistoryMessages: 10}}, attachmentsDir: dir}
	messages := []store.Message{
		{Role: "user", Content: "describe this", Status: "completed", ConversationID: convID, Attachments: []store.Attachment{{ID: "att-1", StorageKey: storageKey, ContentType: "image/png"}}},
		{Role: "assistant", Content: "it is a small image", Status: "completed"},
		{Role: "user", Content: "thanks", Status: "completed"},
	}
	got := s.providerMessages(context.Background(), messages, "system")
	// system + 3 history messages.
	if len(got) != 4 {
		t.Fatalf("provider messages length = %d, want 4: %#v", len(got), got)
	}
	if got[1].Role != "user" || got[1].Content != "describe this" || len(got[1].ImageParts) != 1 {
		t.Fatalf("image user message not preserved: %+v", got[1])
	}
	// Assistant and later user messages stay text-only.
	if len(got[2].ImageParts) != 0 || len(got[3].ImageParts) != 0 {
		t.Fatalf("text-only messages should have no image parts: %+v", got)
	}
}

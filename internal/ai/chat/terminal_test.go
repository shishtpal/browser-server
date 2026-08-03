package chat

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedactStripsImageDataURLs(t *testing.T) {
	// Image bytes must never be retained in request logs, even when full-payload
	// logging is enabled. The data-URL redaction runs after secret redaction.
	input := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"describe"},{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUg== somebody"}}]}],"authorization":"Bearer abc123"}`)
	out := redact(input)
	s := string(out)
	if strings.Contains(s, "iVBORw0KGgoAAAANSUhEUg") {
		t.Fatalf("base64 image bytes leaked into redacted log: %s", s)
	}
	if strings.Contains(s, "data:image/png;base64,") {
		t.Fatalf("full data URL survived redaction: %s", s)
	}
	if !strings.Contains(s, "data:image/[REDACTED]") {
		t.Fatalf("expected data:image/[REDACTED] placeholder, got: %s", s)
	}
	// Secret redaction must still apply alongside image redaction.
	if strings.Contains(s, "Bearer abc123") {
		t.Fatalf("bearer token leaked into redacted log: %s", s)
	}
}

func TestRedactLeavesTextContentIntact(t *testing.T) {
	// Plain text content and non-image data URLs must be preserved.
	input := []byte(`{"content":"describe this image in detail"}`)
	out := redact(input)
	if !bytes.Equal(out, input) {
		t.Fatalf("text content should be unchanged, got: %s", string(out))
	}
}

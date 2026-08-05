package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/bootstrap"
	"browser-server/internal/ai/store"
)

// buildPrompt assembles the user message: --file contents are inlined first
// (matching how the model best attends to instructions after context), then the
// prompt text. When there are no files the prompt is returned unchanged.
func buildPrompt(files []string, prompt string, maxReadBytes int) (string, error) {
	section, err := inlineFiles(files, maxReadBytes)
	if err != nil {
		return "", err
	}
	combined := section + strings.TrimSpace(prompt)
	return strings.TrimSpace(combined), nil
}

// inlineFiles reads each file, rejects non-UTF-8 content, and bounds the size by
// the configured file-tool read limit so a huge file cannot blow up the prompt.
func inlineFiles(files []string, maxReadBytes int) (string, error) {
	if len(files) == 0 {
		return "", nil
	}
	var b strings.Builder
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		if !utf8.Valid(data) {
			return "", fmt.Errorf("%s is not valid UTF-8 and cannot be inlined", path)
		}
		if len(data) > maxReadBytes {
			return "", fmt.Errorf("%s exceeds the %d byte file-inline limit (file_tools.max_read_bytes)", path, maxReadBytes)
		}
		fmt.Fprintf(&b, "<file path=%q>\n%s\n</file>\n\n", path, data)
	}
	return b.String(), nil
}

// splitList splits a comma-separated flag value, trimming whitespace and
// dropping empty entries.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// readStdinIfPiped returns stdin's contents when stdin is a pipe (not a
// terminal), and an empty string when it is interactive.
func readStdinIfPiped() (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// stageImages validates each --image with the same primitives as the HTTP
// upload handler (signature sniffing, decompression-bomb guard, configured
// byte limits), writes it under the attachment root, and records a staged
// attachment row. On failure it removes every file and staged row written so
// far, so a failed run leaves no orphans.
func stageImages(ctx context.Context, rt *bootstrap.Runtime, convID string, images []string) ([]string, error) {
	cfg := rt.Config.Chat.Attachments
	if !cfg.Enabled {
		return nil, fmt.Errorf("image attachments are disabled in bs-ai-config.json")
	}
	if len(images) > cfg.MaxImages {
		return nil, fmt.Errorf("too many images: got %d, limit %d", len(images), cfg.MaxImages)
	}
	var ids []string
	var staged []store.Attachment
	var total int64
	cleanup := func() {
		for _, att := range staged {
			_, _ = rt.Store.DeleteStagedAttachment(context.Background(), convID, att.ID)
			_ = attachments.Remove(rt.AttachmentsDir, convID, att.StorageKey)
		}
	}
	for _, path := range images {
		data, err := os.ReadFile(path)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("read image %s: %w", path, err)
		}
		if len(data) > cfg.MaxImageBytes {
			cleanup()
			return nil, fmt.Errorf("image %s exceeds the %d byte single-image limit", path, cfg.MaxImageBytes)
		}
		contentType, width, height, err := attachments.ValidateImage(data, cfg.AllowedMIMETypes)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("image %s: %v", path, err)
		}
		total += int64(len(data))
		if cfg.MaxTotalBytes > 0 && total > int64(cfg.MaxTotalBytes) {
			cleanup()
			return nil, fmt.Errorf("images exceed the %d byte total limit", cfg.MaxTotalBytes)
		}
		attachmentID := store.NewID("att")
		storageKey, err := attachments.Write(rt.AttachmentsDir, convID, attachmentID, contentType, data)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("store image %s: %w", path, err)
		}
		filename := filepath.Base(path)
		att, err := rt.Store.CreateStagedAttachment(ctx, attachmentID, convID, filename, contentType, int64(len(data)), width, height, storageKey)
		if err != nil {
			_ = attachments.Remove(rt.AttachmentsDir, convID, storageKey)
			cleanup()
			return nil, fmt.Errorf("record image %s: %w", path, err)
		}
		staged = append(staged, att)
		ids = append(ids, att.ID)
	}
	return ids, nil
}

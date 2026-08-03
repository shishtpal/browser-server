// Package attachments owns the on-disk lifecycle of chat image attachments:
// writing validated image bytes under the private AI data directory, reading
// them back for provider requests, and removing them on cancel/cleanup. The
// database only ever stores the server-generated storage key returned here.
package attachments

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxDimension caps either image edge in pixels (defense against absurd
	// header claims that could crash decoders or exhaust memory).
	MaxDimension = 10000
	// MaxPixels caps the decoded pixel area (decompression-bomb guard).
	MaxPixels = 40_000_000
)

// Dir returns the private attachment root below the AI data directory.
func Dir(dataDir string) string {
	return filepath.Join(dataDir, "ai-attachments")
}

// ConversationDir returns the directory that holds one conversation's files
// beneath the attachment root. The caller supplies the attachment root (the
// value returned by Dir); passing the raw data directory would duplicate the
// "ai-attachments" segment.
func ConversationDir(root, conversationID string) string {
	return filepath.Join(root, sanitizeSegment(conversationID))
}

// PathFor resolves a stored attachment's absolute path from its storage key
// beneath the attachment root.
func PathFor(root, conversationID, storageKey string) string {
	return filepath.Join(ConversationDir(root, conversationID), sanitizeSegment(storageKey))
}

// sanitizeSegment makes an id/key safe to use as a single path element; the
// values are server-generated (NewID) so this is belt-and-braces only.
func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "..", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	if value == "" || value == "." {
		return "_"
	}
	return value
}

// ValidateImage sniffs the file signature (not the client-supplied header),
// checks it against the allowed MIME list, and decodes safe dimensions for the
// formats the standard library understands. WEBP is signature-validated but its
// dimensions are not decoded (no stdlib/webp decoder), so it returns 0,0.
func ValidateImage(data []byte, allowed []string) (contentType string, width, height int, err error) {
	if len(data) == 0 {
		return "", 0, 0, errors.New("empty image")
	}
	sniffed := sniffImageType(data)
	if sniffed == "" || !contains(allowed, sniffed) {
		return "", 0, 0, fmt.Errorf("unsupported image type")
	}
	if sniffed == "image/webp" {
		return sniffed, 0, 0, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, fmt.Errorf("corrupt image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", 0, 0, errors.New("image has no dimensions")
	}
	if cfg.Width > MaxDimension || cfg.Height > MaxDimension {
		return "", 0, 0, fmt.Errorf("image dimensions %dx%d exceed the %d pixel limit", cfg.Width, cfg.Height, MaxDimension)
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return "", 0, 0, fmt.Errorf("image area %dx%d exceeds the %d pixel limit", cfg.Width, cfg.Height, MaxPixels)
	}
	return sniffed, cfg.Width, cfg.Height, nil
}

// ExtFor returns the on-disk extension for an allowed image MIME type.
func ExtFor(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return ".img"
}

func sniffImageType(data []byte) string {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	}
	return ""
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// Write atomically writes image bytes to <conversationDir>/<attachmentID><ext>
// using a temp file + rename, creating the conversation directory with
// restrictive permissions. root is the attachment root (Dir(dataDir)). It
// returns the storage key (relative filename).
func Write(root, conversationID, attachmentID, contentType string, data []byte) (string, error) {
	dir := ConversationDir(root, conversationID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	storageKey := sanitizeSegment(attachmentID) + ExtFor(contentType)
	final := filepath.Join(dir, storageKey)
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, final); err != nil {
		return "", err
	}
	return storageKey, nil
}

// Read returns the raw bytes for a stored attachment beneath the attachment root.
func Read(root, conversationID, storageKey string) ([]byte, error) {
	return os.ReadFile(PathFor(root, conversationID, storageKey))
}

// Remove deletes a stored attachment file beneath the attachment root,
// ignoring a missing file so cleanup is idempotent.
func Remove(root, conversationID, storageKey string) error {
	err := os.Remove(PathFor(root, conversationID, storageKey))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RemoveConversationDir deletes a conversation's whole attachment directory
// beneath the attachment root (used when a conversation is deleted). Missing
// directories are ignored.
func RemoveConversationDir(root, conversationID string) error {
	err := os.RemoveAll(ConversationDir(root, conversationID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

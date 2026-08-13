package attachments

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// validPNG encodes a 2x2 PNG so ValidateImage can decode real dimensions.
func validPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestValidateImageSniffsSignatureNotHeader(t *testing.T) {
	// A PNG body advertised as image/jpeg by the client must still be detected
	// as image/png (the server sniffs bytes, never trusts the header).
	ct, w, h, err := ValidateImage(validPNG(t), []string{"image/png", "image/jpeg"})
	if err != nil {
		t.Fatalf("ValidateImage: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("content type = %q, want image/png", ct)
	}
	if w != 2 || h != 2 {
		t.Fatalf("dimensions = %dx%d, want 2x2", w, h)
	}
}

func TestValidateImageRejectsUnsupportedAndCorrupt(t *testing.T) {
	allowed := []string{"image/png", "image/jpeg", "image/webp", "image/gif"}
	if _, _, _, err := ValidateImage(nil, allowed); err == nil {
		t.Fatal("expected error for empty image")
	}
	// Spoofed header: bytes are plain text but pretend to be an image.
	if _, _, _, err := ValidateImage([]byte("not an image at all"), allowed); err == nil {
		t.Fatal("expected error for non-image bytes")
	}
	// A valid PNG signature is not in the allowed list.
	pngBytes := validPNG(t)
	if _, _, _, err := ValidateImage(pngBytes, []string{"image/jpeg"}); err == nil {
		t.Fatal("expected error when sniffed type is not in the allowed list")
	}
	// Truncated PNG (signature present but body cut) is corrupt.
	truncated := validPNG(t)[:8]
	if _, _, _, err := ValidateImage(truncated, allowed); err == nil {
		t.Fatal("expected error for corrupt/truncated image")
	}
}

func TestValidateImageWebPReturnsZeroDimensions(t *testing.T) {
	// WEBP has no stdlib decoder; ValidateImage signature-validates it but
	// returns 0,0 dimensions (the catalog still accepts the upload).
	webp := []byte("RIFF\x00\x00\x00\x00WEBP")
	ct, w, h, err := ValidateImage(webp, []string{"image/webp"})
	if err != nil {
		t.Fatalf("ValidateImage webp: %v", err)
	}
	if ct != "image/webp" || w != 0 || h != 0 {
		t.Fatalf("webp = %q %dx%d, want image/webp 0x0", ct, w, h)
	}
}

func TestWriteReadRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	convID := "conv-123"
	attID := "att-456"
	data := validPNG(t)

	storageKey, err := Write(dir, convID, attID, "image/png", data)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if storageKey != "att-456.png" {
		t.Fatalf("storage key = %q, want att-456.png", storageKey)
	}
	// File lives under the private conversation directory, not the client path.
	got, err := Read(dir, convID, storageKey)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("read bytes do not match written bytes")
	}
	// The on-disk file is not world/group readable on platforms that honor
	// Unix permission bits (Windows ignores Chmod, so skip the check there).
	if runtime.GOOS != "windows" {
		info, err := os.Stat(PathFor(dir, convID, storageKey))
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("attachment file is group/world readable: %v", info.Mode())
		}
	}
	// Remove is idempotent: a second call does not error.
	if err := Remove(dir, convID, storageKey); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := Remove(dir, convID, storageKey); err != nil {
		t.Fatalf("idempotent Remove: %v", err)
	}
}

func TestSanitizeSegmentNeutralizesTraversal(t *testing.T) {
	cases := map[string]string{
		"..":     "_",
		"../etc": "__etc",
		"a/b":    "a_b",
		`a\b`:    "a_b",
		"":       "_",
		".":      "_",
		"normal": "normal",
	}
	for in, want := range cases {
		if got := sanitizeSegment(in); got != want {
			t.Fatalf("sanitizeSegment(%q) = %q, want %q", in, got, want)
		}
	}
	// A crafted key can never escape the conversation directory.
	dir := t.TempDir()
	convID := "conv"
	escaped := sanitizeSegment("../../escape")
	final := PathFor(dir, convID, escaped)
	if strings.Contains(filepath.ToSlash(final), "/../") {
		t.Fatalf("sanitized path still traverses: %s", final)
	}
	// The resolved path must remain inside the conversation dir.
	convDir := ConversationDir(dir, convID)
	rel, err := filepath.Rel(convDir, final)
	if err != nil || strings.HasPrefix(filepath.ToSlash(rel), "..") {
		t.Fatalf("attachment path escapes conversation dir: rel=%q err=%v", rel, err)
	}
}

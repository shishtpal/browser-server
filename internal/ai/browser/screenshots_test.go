package browser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScreenshotStoreSaveReturnsURLAndPath(t *testing.T) {
	dir := t.TempDir()
	s := NewScreenshotStore(dir)
	ctx := context.Background()

	ref, err := s.Save(ctx, "cmd_abc123", []byte("png-bytes"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	wantPath := filepath.Join(dir, "browser-screenshots", "cmd_abc123.png")
	if ref.URL != "/api/browser/screenshots/cmd_abc123.png" {
		t.Fatalf("url = %q, want %q", ref.URL, "/api/browser/screenshots/cmd_abc123.png")
	}
	if ref.Path != wantPath {
		t.Fatalf("path = %q, want %q", ref.Path, wantPath)
	}
	info, err := os.Stat(ref.Path)
	if err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
	if info.Size() != int64(len("png-bytes")) {
		t.Fatalf("saved size = %d, want %d", info.Size(), len("png-bytes"))
	}
}

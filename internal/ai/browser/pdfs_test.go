package browser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPdfStoreSaveReturnsURLAndPath(t *testing.T) {
	dir := t.TempDir()
	s := NewPdfStore(dir)
	ctx := context.Background()

	ref, err := s.Save(ctx, "cmd_abc123", []byte("%PDF-1.4 pdf-bytes"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	wantPath := filepath.Join(dir, "browser-pdfs", "cmd_abc123.pdf")
	if ref.URL != "/api/browser/pdfs/cmd_abc123.pdf" {
		t.Fatalf("url = %q, want %q", ref.URL, "/api/browser/pdfs/cmd_abc123.pdf")
	}
	if ref.Path != wantPath {
		t.Fatalf("path = %q, want %q", ref.Path, wantPath)
	}
	info, err := os.Stat(ref.Path)
	if err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
	if info.Size() != int64(len("%PDF-1.4 pdf-bytes")) {
		t.Fatalf("saved size = %d, want %d", info.Size(), len("%PDF-1.4 pdf-bytes"))
	}
}

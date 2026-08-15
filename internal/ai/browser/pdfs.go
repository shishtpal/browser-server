package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corebrowser "browser-server/internal/browser"
)

// PdfStore persists browser PDF payloads (the extension's base64 PDF data URL
// from CDP Page.printToPDF) to disk under the browser data directory. Tool
// output keeps a small server-relative URL instead of inlining multi-hundred-KB
// base64, which would blow the model output budget, plus the local file path so
// downstream tools can read the stored PDF directly.
type PdfStore struct {
	dir string
}

// NewPdfStore roots the store at <dataDir>/browser-pdfs.
func NewPdfStore(dataDir string) *PdfStore {
	return &PdfStore{dir: filepath.Join(dataDir, "browser-pdfs")}
}

// Dir returns the directory PDFs are stored in.
func (s *PdfStore) Dir() string {
	return s.dir
}

// Save writes pdf to <commandID>.pdf and returns the server-relative URL that
// serves it back together with the absolute file path on disk. commandID comes
// from the bus as "cmd_<hex>", so the filename cannot contain path separators.
func (s *PdfStore) Save(ctx context.Context, commandID string, pdf []byte) (corebrowser.PdfRef, error) {
	if err := ctx.Err(); err != nil {
		return corebrowser.PdfRef{}, err
	}
	if commandID == "" {
		return corebrowser.PdfRef{}, fmt.Errorf("pdf: command id is required")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return corebrowser.PdfRef{}, fmt.Errorf("pdf: create dir: %w", err)
	}
	name := commandID + ".pdf"
	if err := os.WriteFile(filepath.Join(s.dir, name), pdf, 0o644); err != nil {
		return corebrowser.PdfRef{}, fmt.Errorf("pdf: write file: %w", err)
	}
	return corebrowser.PdfRef{
		URL:  "/api/browser/pdfs/" + name,
		Path: filepath.Join(s.dir, name),
	}, nil
}

package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corebrowser "browser-server/internal/browser"
)

// ScreenshotStore persists browser screenshot payloads (the extension's base64
// PNG data URL) to disk under the browser data directory. Tool output keeps a
// small server-relative URL instead of inlining multi-hundred-KB base64, which
// would blow the model output budget, plus the local file path so downstream
// tools (e.g. ocr_image) can read the stored PNG directly.
type ScreenshotStore struct {
	dir string
}

// NewScreenshotStore roots the store at <dataDir>/browser-screenshots.
func NewScreenshotStore(dataDir string) *ScreenshotStore {
	return &ScreenshotStore{dir: filepath.Join(dataDir, "browser-screenshots")}
}

// Dir returns the directory screenshots are stored in.
func (s *ScreenshotStore) Dir() string {
	return s.dir
}

// Save writes png to <commandID>.png and returns the server-relative URL that
// serves it back together with the absolute file path on disk. commandID comes
// from the bus as "cmd_<hex>", so the filename cannot contain path separators.
func (s *ScreenshotStore) Save(ctx context.Context, commandID string, png []byte) (corebrowser.ScreenshotRef, error) {
	if err := ctx.Err(); err != nil {
		return corebrowser.ScreenshotRef{}, err
	}
	if commandID == "" {
		return corebrowser.ScreenshotRef{}, fmt.Errorf("screenshot: command id is required")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return corebrowser.ScreenshotRef{}, fmt.Errorf("screenshot: create dir: %w", err)
	}
	name := commandID + ".png"
	if err := os.WriteFile(filepath.Join(s.dir, name), png, 0o644); err != nil {
		return corebrowser.ScreenshotRef{}, fmt.Errorf("screenshot: write file: %w", err)
	}
	return corebrowser.ScreenshotRef{
		URL:  "/api/browser/screenshots/" + name,
		Path: filepath.Join(s.dir, name),
	}, nil
}

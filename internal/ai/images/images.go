// Package images provides provider-neutral image generation and a local gallery.
package images

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ErrProvider marks failures that originated upstream rather than in the
// caller's request, so handlers can answer 502 instead of 400.
var ErrProvider = errors.New("image provider request failed")

// supportedImageTypes enumerates the MIME types the gallery accepts from a
// provider response or a local source image.
var supportedImageTypes = []string{"image/png", "image/jpeg", "image/webp"}

// Model describes a single image-generation model exposed by a provider.
type Model struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Default         bool     `json:"default"`
	SupportsEditing bool     `json:"supports_editing"`
	ImageSizes      []string `json:"image_sizes"`
	AspectRatios    []string `json:"aspect_ratios,omitempty"`
	MaxImages       int      `json:"max_images,omitempty"`
	SupportsSeed    bool     `json:"supports_seed,omitempty"`
	// ThinkingLevel sets the model's thinking budget for image generation
	// (Gemini's generation_config.thinking_level). It is model-specific:
	// e.g. gemini-3-flash-image allows "minimal"/"high" while
	// gemini-3-pro-image allows "high"/"low". When empty the field is omitted
	// from the request so models that reject it are unaffected.
	ThinkingLevel string `json:"thinking_level,omitempty"`
}

// Provider groups the connection details and model catalog for one vendor.
type Provider struct {
	Type                  string  `json:"type"`
	BaseURL               string  `json:"base_url"`
	APIKey                string  `json:"api_key"`
	RequestTimeoutSeconds int     `json:"request_timeout_seconds"`
	Models                []Model `json:"models"`
}

// Config is the parsed bs-ai-image-models.json document.
type Config struct {
	Enabled         bool                `json:"enabled"`
	DefaultProvider string              `json:"default_provider"`
	Providers       map[string]Provider `json:"providers"`
	Path            string              `json:"-"`
}

// Image is one persisted gallery entry.
type Image struct {
	ID          string    `json:"id"`
	Prompt      string    `json:"prompt"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	ImageSize   string    `json:"image_size"`
	ContentType string    `json:"content_type"`
	Filename    string    `json:"filename"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// GenerateRequest is the normalized generation request handed to the service.
type GenerateRequest struct {
	Prompt      string   `json:"prompt"`
	Provider    string   `json:"provider,omitempty"`
	Model       string   `json:"model,omitempty"`
	ImageSize   string   `json:"image_size,omitempty"`
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	N           int      `json:"n,omitempty"`
	Seed        *int     `json:"seed,omitempty"`
	Sources     [][]byte `json:"-"`
}

// Service owns the gallery database and the HTTP client used to reach providers.
type Service struct {
	cfg    Config
	db     *sql.DB
	root   string
	client *http.Client
}

// New opens (or creates) the gallery database under dataDir. A nil service is
// returned when the feature is disabled.
func New(cfg Config, dataDir string) (*Service, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "ai-images"), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(dataDir, "ai-images.db"))+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS ai_images (id TEXT PRIMARY KEY,prompt TEXT NOT NULL,provider TEXT NOT NULL,model TEXT NOT NULL,image_size TEXT NOT NULL,content_type TEXT NOT NULL,filename TEXT NOT NULL,size_bytes INTEGER NOT NULL,created_at TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Service{cfg: cfg, db: db, root: filepath.Join(dataDir, "ai-images"), client: &http.Client{}}, nil
}

// Close releases the gallery database. It is safe to call on a nil service.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	return s.db.Close()
}

// Config returns the configuration the service was constructed from.
func (s *Service) Config() Config { return s.cfg }

// Package images provides provider-neutral image generation and a local gallery.
package images

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/store"

	_ "github.com/mattn/go-sqlite3"
)

const modelsFile = "bs-ai-image-models.json"

// ErrProvider marks failures that originated upstream rather than in the
// caller's request, so handlers can answer 502 instead of 400.
var ErrProvider = errors.New("image provider request failed")

var supportedImageTypes = []string{"image/png", "image/jpeg", "image/webp"}

type Model struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Default         bool     `json:"default"`
	SupportsEditing bool     `json:"supports_editing"`
	ImageSizes      []string `json:"image_sizes"`
	AspectRatios    []string `json:"aspect_ratios,omitempty"`
	MaxImages       int      `json:"max_images,omitempty"`
	SupportsSeed    bool     `json:"supports_seed,omitempty"`
}
type Provider struct {
	Type                  string  `json:"type"`
	BaseURL               string  `json:"base_url"`
	APIKey                string  `json:"api_key"`
	RequestTimeoutSeconds int     `json:"request_timeout_seconds"`
	Models                []Model `json:"models"`
}
type Config struct {
	Enabled         bool                `json:"enabled"`
	DefaultProvider string              `json:"default_provider"`
	Providers       map[string]Provider `json:"providers"`
	Path            string              `json:"-"`
}
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
type block struct {
	Type string `json:"type"`
	Data string `json:"data"`
}
type Service struct {
	cfg    Config
	db     *sql.DB
	root   string
	client *http.Client
}

func Load(base string) (Config, error) {
	return LoadPath(filepath.Join(base, modelsFile))
}

// LoadPath loads an explicit image config. A missing file is a valid disabled
// configuration.
func LoadPath(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Path: path, Providers: map[string]Provider{}}, nil
	}
	if err != nil {
		return Config{}, err
	}
	return parseConfig(content, path)
}

// ValidateBytes performs semantic validation without opening the gallery DB.
func ValidateBytes(content []byte) error {
	_, err := parseConfig(content, modelsFile)
	return err
}

func parseConfig(content []byte, path string) (Config, error) {
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("parse image models: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return config, errors.New("parse image models: multiple JSON values")
		}
		return config, fmt.Errorf("parse image models: %w", err)
	}
	config.Path = path
	if config.Providers == nil {
		config.Providers = map[string]Provider{}
	}
	if !config.Enabled {
		return config, nil
	}
	if config.DefaultProvider == "" {
		return config, errors.New("image default_provider is required")
	}
	if _, ok := config.Providers[config.DefaultProvider]; !ok {
		return config, errors.New("image default_provider is not configured")
	}
	for name, provider := range config.Providers {
		if provider.Type != "gemini_interactions" && provider.Type != "openrouter_images" {
			return config, fmt.Errorf("image provider %q has unsupported type", name)
		}
		if provider.APIKey == "" {
			return config, fmt.Errorf("image provider %q api_key is required", name)
		}
		if env, ok := strings.CutPrefix(provider.APIKey, "env:"); ok {
			provider.APIKey = os.Getenv(env)
			if provider.APIKey == "" {
				return config, fmt.Errorf("image provider %q API key environment variable %q is empty", name, env)
			}
		}
		if provider.BaseURL == "" && provider.Type == "gemini_interactions" {
			provider.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		if provider.BaseURL == "" {
			provider.BaseURL = "https://openrouter.ai/api/v1"
		}
		if provider.RequestTimeoutSeconds == 0 {
			// Image providers can spend several minutes rendering an image;
			// keep this independent from the shorter chat completion timeout.
			provider.RequestTimeoutSeconds = 600
		}
		if len(provider.Models) == 0 {
			return config, fmt.Errorf("image provider %q needs models", name)
		}
		config.Providers[name] = provider
	}
	return config, nil
}

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
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	return s.db.Close()
}
func (s *Service) Config() Config { return s.cfg }
func (s *Service) List(ctx context.Context, limit int) ([]Image, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,prompt,provider,model,image_size,content_type,filename,size_bytes,created_at FROM ai_images ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Image{}
	for rows.Next() {
		var x Image
		var t string
		if err := rows.Scan(&x.ID, &x.Prompt, &x.Provider, &x.Model, &x.ImageSize, &x.ContentType, &x.Filename, &x.SizeBytes, &t); err != nil {
			return nil, err
		}
		x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Service) Get(ctx context.Context, id string) (Image, error) {
	var x Image
	var t string
	err := s.db.QueryRowContext(ctx, `SELECT id,prompt,provider,model,image_size,content_type,filename,size_bytes,created_at FROM ai_images WHERE id=?`, id).Scan(&x.ID, &x.Prompt, &x.Provider, &x.Model, &x.ImageSize, &x.ContentType, &x.Filename, &x.SizeBytes, &t)
	x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
	return x, err
}
func (s *Service) Delete(ctx context.Context, id string) error {
	x, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM ai_images WHERE id=?`, id); err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(s.root, x.Filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
func (s *Service) Read(ctx context.Context, id string) (Image, []byte, error) {
	x, err := s.Get(ctx, id)
	if err != nil {
		return x, nil, err
	}
	b, err := os.ReadFile(filepath.Join(s.root, x.Filename))
	return x, b, err
}
func (s *Service) Generate(ctx context.Context, r GenerateRequest) (Image, error) {
	images, err := s.GenerateMany(ctx, r)
	if err != nil {
		return Image{}, err
	}
	return images[0], nil
}

// GenerateMany persists every requested image. Providers that only return one
// image per request are called once per requested result to preserve the API's
// n contract without dropping successful outputs.
func (s *Service) GenerateMany(ctx context.Context, r GenerateRequest) ([]Image, error) {
	count := r.N
	if count == 0 {
		count = 1
	}
	if count < 1 || count > 6 {
		return nil, errors.New("n must be between 1 and 6")
	}
	r.N = 1
	result := make([]Image, 0, count)
	for range count {
		x, err := s.generateOne(ctx, r)
		if err != nil {
			return result, err
		}
		result = append(result, x)
	}
	return result, nil
}

func (s *Service) generateOne(ctx context.Context, r GenerateRequest) (Image, error) {
	if strings.TrimSpace(r.Prompt) == "" {
		return Image{}, errors.New("prompt is required")
	}
	pn := r.Provider
	if pn == "" {
		pn = s.cfg.DefaultProvider
	}
	p, ok := s.cfg.Providers[pn]
	if !ok {
		return Image{}, errors.New("unknown image provider")
	}
	m := r.Model
	var mc Model
	if m == "" {
		mc = p.Models[0]
		for _, v := range p.Models {
			if v.Default {
				mc = v
				break
			}
		}
		m = mc.ID
	} else {
		for _, v := range p.Models {
			if v.ID == m {
				mc = v
				break
			}
		}
	}
	if mc.ID == "" {
		return Image{}, errors.New("unknown image model")
	}
	size := r.ImageSize
	if size == "" {
		if len(mc.ImageSizes) > 0 {
			size = mc.ImageSizes[0]
		} else {
			size = "1K"
		}
	}
	if len(mc.ImageSizes) > 0 && !slices.Contains(mc.ImageSizes, size) {
		return Image{}, fmt.Errorf("image size %q is not supported by %s", size, mc.ID)
	}
	if r.AspectRatio != "" && len(mc.AspectRatios) > 0 && !slices.Contains(mc.AspectRatios, r.AspectRatio) {
		return Image{}, fmt.Errorf("aspect ratio %q is not supported by %s", r.AspectRatio, mc.ID)
	}
	if r.Seed != nil && !mc.SupportsSeed {
		return Image{}, errors.New("selected model does not support seed")
	}
	if len(r.Sources) > 0 && !mc.SupportsEditing {
		return Image{}, errors.New("selected model does not support editing")
	}
	input := make([]any, 0, len(r.Sources)+1)
	for _, b := range r.Sources {
		ct, _, _, e := attachments.ValidateImage(b, supportedImageTypes)
		if e != nil {
			return Image{}, e
		}
		input = append(input, map[string]string{"type": "image", "mime_type": ct, "data": base64.StdEncoding.EncodeToString(b)})
	}
	input = append(input, map[string]string{"type": "text", "text": r.Prompt})
	payload := map[string]any{
		"model":             strings.TrimPrefix(m, "models/"),
		"input":             input,
		"response_format":   map[string]string{"type": "image", "image_size": size},
		"generation_config": map[string]any{"thinking_level": "low"},
	}
	var raw []byte
	if p.Type == "openrouter_images" {
		payload = map[string]any{"model": m, "prompt": r.Prompt, "resolution": size, "n": 1}
		if r.AspectRatio != "" {
			payload["aspect_ratio"] = r.AspectRatio
		}
		if r.Seed != nil {
			payload["seed"] = *r.Seed
		}
		if len(r.Sources) > 0 {
			refs := make([]map[string]any, 0, len(r.Sources))
			for _, b := range r.Sources {
				contentType, _, _, err := attachments.ValidateImage(b, supportedImageTypes)
				if err != nil {
					return Image{}, err
				}
				refs = append(refs, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(b)}})
			}
			payload["input_references"] = refs
		}
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return Image{}, e
	}
	u := strings.TrimRight(p.BaseURL, "/") + "/interactions"
	if p.Type == "openrouter_images" {
		u = strings.TrimRight(p.BaseURL, "/") + "/images"
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if e != nil {
		return Image{}, e
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Type == "openrouter_images" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	} else {
		req.Header.Set("x-goog-api-key", p.APIKey)
	}
	c := *s.client
	c.Timeout = time.Duration(p.RequestTimeoutSeconds) * time.Second
	resp, e := c.Do(req)
	if e != nil {
		return Image{}, fmt.Errorf("%w: %v", ErrProvider, e)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Image{}, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	data, ct, e := extract(body)
	if p.Type == "openrouter_images" {
		data, ct, e = extractOpenRouter(body)
	}
	if e != nil {
		return Image{}, fmt.Errorf("%w: %v", ErrProvider, e)
	}
	id := store.NewID("img")
	ext := attachments.ExtFor(ct)
	fn := id + ext
	if e = os.WriteFile(filepath.Join(s.root, fn), data, 0600); e != nil {
		return Image{}, e
	}
	x := Image{ID: id, Prompt: r.Prompt, Provider: pn, Model: m, ImageSize: size, ContentType: ct, Filename: fn, SizeBytes: int64(len(data)), CreatedAt: time.Now().UTC()}
	_, e = s.db.ExecContext(ctx, `INSERT INTO ai_images VALUES(?,?,?,?,?,?,?,?,?)`, x.ID, x.Prompt, x.Provider, x.Model, x.ImageSize, x.ContentType, x.Filename, x.SizeBytes, x.CreatedAt.Format(time.RFC3339Nano))
	if e != nil {
		_ = os.Remove(filepath.Join(s.root, fn))
		return Image{}, e
	}
	return x, nil
}

func extractOpenRouter(b []byte) ([]byte, string, error) {
	var v struct {
		Data []struct {
			B64JSON   string `json:"b64_json"`
			MediaType string `json:"media_type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &v); err != nil || len(v.Data) == 0 || v.Data[0].B64JSON == "" {
		return nil, "", errors.New("image provider response contains no image")
	}
	out, err := base64.StdEncoding.DecodeString(v.Data[0].B64JSON)
	if err != nil {
		return nil, "", errors.New("invalid image data")
	}
	ct, _, _, err := attachments.ValidateImage(out, supportedImageTypes)
	if err != nil {
		return nil, "", err
	}
	if v.Data[0].MediaType != "" {
		ct = v.Data[0].MediaType
	}
	return out, ct, nil
}

// providerMessage pulls the human-readable reason out of a Google API error
// envelope so operators see "API key not valid" instead of a bare 400.
func providerMessage(b []byte) string {
	var v struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &v) == nil && v.Error.Message != "" {
		if v.Error.Status != "" {
			return v.Error.Status + ": " + v.Error.Message
		}
		return v.Error.Message
	}
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		s = s[:300]
	}
	if s == "" {
		return "no response body"
	}
	return s
}

// extract walks the Interactions response for the last generated image block.
// Blocks live under steps[].content[] (and steps[].summary[] for thoughts);
// output_image is the convenience field the SDKs expose.
func extract(b []byte) ([]byte, string, error) {
	var v struct {
		OutputImage *block `json:"output_image"`
		Steps       []struct {
			Content []block `json:"content"`
			Summary []block `json:"summary"`
		} `json:"steps"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil, "", errors.New("malformed image provider response")
	}
	var found *block
	if v.OutputImage != nil && v.OutputImage.Data != "" {
		found = v.OutputImage
	}
	for _, step := range v.Steps {
		for _, blocks := range [][]block{step.Content, step.Summary} {
			for i, x := range blocks {
				if x.Type == "image" && x.Data != "" {
					found = &blocks[i]
				}
			}
		}
	}
	if found == nil {
		return nil, "", errors.New("image provider response contains no image")
	}
	out, e := base64.StdEncoding.DecodeString(found.Data)
	if e != nil {
		return nil, "", errors.New("invalid image data")
	}
	ct, _, _, e := attachments.ValidateImage(out, supportedImageTypes)
	if e != nil {
		return nil, "", e
	}
	return out, ct, nil
}

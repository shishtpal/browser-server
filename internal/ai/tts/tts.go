// Package tts provides provider-neutral text-to-speech generation, a local
// gallery of generated MP3 files under .data/ai-voices, and SQLite metadata.
// It mirrors the internal/ai/images architecture (see Decisions.md).
// providerSpec endpoint note: for fish.audio, defaultFishBaseURL already
// includes the /v1 prefix, so endpoint "/tts" resolves to
// "https://api.fish.audio/v1/tts" (the upstream fish.audio contract).
package tts

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"browser-server/internal/ai/store"

	_ "github.com/mattn/go-sqlite3"
)

const (
	modelsFile             = "bs-ai-tts.json"
	providerTypeOpenRouter = "openrouter_speech"
	providerTypeFishAudio  = "fish_audio"
	defaultBaseURL         = "https://openrouter.ai/api/v1"
	defaultFishBaseURL     = "https://api.fish.audio/v1"
	defaultTimeoutSeconds  = 120
	maxTextChars           = 4096
	maxAudioBytes          = 20 << 20
	refererURL             = "https://github.com/shishtpal/browser-server"
	openRouterTitle        = "Browser Server"
	contentTypeMPEG        = "audio/mpeg"
)

// ErrProvider marks failures that originated upstream rather than in the
// caller's request, so handlers can answer 502 instead of 400.
var ErrProvider = errors.New("tts provider request failed")

// providerSpec captures the per-provider request shape so adding a new TTS
// provider is a single switch case: field names, endpoint path, extra
// headers, and whether the model travels in the header vs the body. The
// shared Generate loop then assembles the request uniformly.
type providerSpec struct {
	endpoint      string
	inputKey      string
	voiceKey      string
	formatKey     string
	modelInHeader bool
	extraHeaders  func(model string) map[string]string
}

// providerSpecFor returns the request shape for a known provider type. Unknown
// types are rejected at Load time, so this lookup is total in practice.
func providerSpecFor(t string) providerSpec {
	switch t {
	case providerTypeFishAudio:
		return providerSpec{
			endpoint:      "/tts",
			inputKey:      "text",
			voiceKey:      "reference_id",
			formatKey:     "format",
			modelInHeader: true,
			extraHeaders:  func(model string) map[string]string { return map[string]string{"model": model} },
		}
	default:
		// openrouter_speech and any future OpenAI-compatible provider.
		return providerSpec{
			endpoint:  "/audio/speech",
			inputKey:  "input",
			voiceKey:  "voice",
			formatKey: "response_format",
			extraHeaders: func(model string) map[string]string {
				return map[string]string{
					"HTTP-Referer":       refererURL,
					"Referer":            refererURL,
					"X-OpenRouter-Title": openRouterTitle,
				}
			},
		}
	}
}

type Voice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Model struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Default        bool    `json:"default"`
	Voices         []Voice `json:"voices,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
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

type Speech struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Voice        string    `json:"voice"`
	ContentType  string    `json:"content_type"`
	Filename     string    `json:"filename"`
	SizeBytes    int64     `json:"size_bytes"`
	GenerationID string    `json:"generation_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type GenerateRequest struct {
	Text     string `json:"text"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Voice    string `json:"voice,omitempty"`
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

// LoadPath loads an explicit TTS config. A missing file is a valid disabled
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
		return config, fmt.Errorf("parse tts models: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return config, errors.New("parse tts models: multiple JSON values")
		}
		return config, fmt.Errorf("parse tts models: %w", err)
	}
	config.Path = path
	if config.Providers == nil {
		config.Providers = map[string]Provider{}
	}
	if !config.Enabled {
		return config, nil
	}
	if config.DefaultProvider == "" {
		return config, errors.New("tts default_provider is required")
	}
	if _, ok := config.Providers[config.DefaultProvider]; !ok {
		return config, errors.New("tts default_provider is not configured")
	}
	for name, provider := range config.Providers {
		switch provider.Type {
		case providerTypeOpenRouter, providerTypeFishAudio:
			// supported
		default:
			return config, fmt.Errorf("tts provider %q has unsupported type", name)
		}
		if provider.APIKey == "" {
			return config, fmt.Errorf("tts provider %q api_key is required", name)
		}
		if env, ok := strings.CutPrefix(provider.APIKey, "env:"); ok {
			provider.APIKey = os.Getenv(env)
			if provider.APIKey == "" {
				return config, fmt.Errorf("tts provider %q API key environment variable %q is empty", name, env)
			}
		}
		if provider.BaseURL == "" {
			if provider.Type == providerTypeFishAudio {
				provider.BaseURL = defaultFishBaseURL
			} else {
				provider.BaseURL = defaultBaseURL
			}
		}
		if provider.RequestTimeoutSeconds == 0 {
			provider.RequestTimeoutSeconds = defaultTimeoutSeconds
		}
		if len(provider.Models) == 0 {
			return config, fmt.Errorf("tts provider %q needs models", name)
		}
		config.Providers[name] = provider
	}
	return config, nil
}

func New(cfg Config, dataDir string) (*Service, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "ai-voices"), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(dataDir, "ai-voices.db"))+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS ai_voices (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		voice TEXT NOT NULL,
		content_type TEXT NOT NULL,
		filename TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		generation_id TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Service{cfg: cfg, db: db, root: filepath.Join(dataDir, "ai-voices"), client: &http.Client{}}, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Service) Config() Config { return s.cfg }

// FilePath returns the absolute gallery path for a persisted speech filename.
func (s *Service) FilePath(filename string) string {
	return filepath.Join(s.root, filename)
}

func (s *Service) List(ctx context.Context, limit int) ([]Speech, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,text,provider,model,voice,content_type,filename,size_bytes,generation_id,created_at FROM ai_voices ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Speech{}
	for rows.Next() {
		var x Speech
		var t string
		if err := rows.Scan(&x.ID, &x.Text, &x.Provider, &x.Model, &x.Voice, &x.ContentType, &x.Filename, &x.SizeBytes, &x.GenerationID, &t); err != nil {
			return nil, err
		}
		x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (Speech, error) {
	var x Speech
	var t string
	err := s.db.QueryRowContext(ctx, `SELECT id,text,provider,model,voice,content_type,filename,size_bytes,generation_id,created_at FROM ai_voices WHERE id=?`, id).Scan(&x.ID, &x.Text, &x.Provider, &x.Model, &x.Voice, &x.ContentType, &x.Filename, &x.SizeBytes, &x.GenerationID, &t)
	x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
	return x, err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	x, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM ai_voices WHERE id=?`, id); err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(s.root, x.Filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Service) Read(ctx context.Context, id string) (Speech, []byte, error) {
	x, err := s.Get(ctx, id)
	if err != nil {
		return x, nil, err
	}
	b, err := os.ReadFile(filepath.Join(s.root, x.Filename))
	return x, b, err
}

func (s *Service) Generate(ctx context.Context, r GenerateRequest) (Speech, error) {
	// Chat messages are markdown; strip to spoken prose before synthesis so
	// the provider never reads syntax aloud. The stripped text is also what
	// gets stored in the gallery row.
	r.Text = markdownToText(r.Text)
	if strings.TrimSpace(r.Text) == "" {
		return Speech{}, errors.New("text is required")
	}
	if utf8.RuneCountInString(r.Text) > maxTextChars {
		return Speech{}, fmt.Errorf("text exceeds %d characters", maxTextChars)
	}
	pn := r.Provider
	if pn == "" {
		pn = s.cfg.DefaultProvider
	}
	p, ok := s.cfg.Providers[pn]
	if !ok {
		return Speech{}, errors.New("unknown tts provider")
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
		return Speech{}, errors.New("unknown tts model")
	}
	voice := r.Voice
	if len(mc.Voices) > 0 {
		if voice == "" {
			voice = mc.Voices[0].ID
		} else {
			found := false
			for _, v := range mc.Voices {
				if v.ID == voice {
					found = true
					break
				}
			}
			if !found {
				return Speech{}, fmt.Errorf("unknown voice %q for model %s", voice, mc.ID)
			}
		}
	} else if voice != "" {
		// Voiceless models (e.g. fish.audio free tier) cannot honor a
		// caller-provided voice: we have no ids to validate against and the
		// provider would synthesize in its default voice, so fail loudly
		// instead of silently downgrading.
		return Speech{}, fmt.Errorf("model %s has no configured voices", mc.ID)
	}
	spec := providerSpecFor(p.Type)
	payload := map[string]any{spec.inputKey: r.Text}
	if !spec.modelInHeader {
		payload["model"] = m
	}
	// Providers reject a present-but-empty voice ("too_small") instead of
	// defaulting it, so drop the key entirely when no voice is configured
	// for the model and none was requested.
	if voice != "" {
		payload[spec.voiceKey] = voice
	}
	// Don't hardcode the format: send it only when the model's config says
	// to (some providers reject unknown params with a 400), matching how
	// files are still saved as .mp3 regardless.
	if mc.ResponseFormat != "" {
		payload[spec.formatKey] = mc.ResponseFormat
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Speech{}, err
	}
	u := strings.TrimRight(p.BaseURL, "/") + spec.endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payloadBytes))
	if err != nil {
		return Speech{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	for hk, hv := range spec.extraHeaders(m) {
		req.Header.Set(hk, hv)
	}
	c := *s.client
	c.Timeout = time.Duration(p.RequestTimeoutSeconds) * time.Second
	resp, err := c.Do(req)
	if err != nil {
		return Speech{}, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxAudioBytes+1))
	// Status before size: a huge non-2xx body must surface the provider's
	// message, not a misleading "audio exceeds" error.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Speech{}, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	if len(body) > maxAudioBytes {
		return Speech{}, fmt.Errorf("%w: audio exceeds %d bytes", ErrProvider, maxAudioBytes)
	}
	if len(body) == 0 {
		return Speech{}, fmt.Errorf("%w: empty audio body", ErrProvider)
	}
	id := store.NewID("tts")
	fn := id + ".mp3"
	if err = os.WriteFile(filepath.Join(s.root, fn), body, 0600); err != nil {
		return Speech{}, err
	}
	// Remove the mp3 if anything below fails or if a retry loop reruns
	// Generate with the same request; never leave orphaned gallery files.
	persisted := false
	defer func() {
		if !persisted {
			_ = os.Remove(filepath.Join(s.root, fn))
		}
	}()
	x := Speech{
		ID:           id,
		Text:         r.Text,
		Provider:     pn,
		Model:        m,
		Voice:        voice,
		ContentType:  contentTypeMPEG,
		Filename:     fn,
		SizeBytes:    int64(len(body)),
		GenerationID: resp.Header.Get("X-Generation-Id"),
		CreatedAt:    time.Now().UTC(),
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO ai_voices VALUES(?,?,?,?,?,?,?,?,?,?)`, x.ID, x.Text, x.Provider, x.Model, x.Voice, x.ContentType, x.Filename, x.SizeBytes, x.GenerationID, x.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Speech{}, err
	}
	persisted = true
	return x, nil
}

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

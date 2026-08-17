package voice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	defaultFile               = "bs-ai-voice.json"
	providerTypeSarvam        = "sarvam_streaming"
	providerTypeOpenRouterSTT = "openrouter_stt"
	defaultOpenRouterBaseURL  = "https://openrouter.ai/api/v1"
	openRouterReferer         = "https://github.com/shishtpal/browser-server"
	openRouterTitle           = "Browser Server"
)

type Config struct {
	Enabled         bool                `json:"enabled"`
	DefaultProvider string              `json:"default_provider"`
	Languages       []Language          `json:"languages"`
	Recording       Recording           `json:"recording"`
	Providers       map[string]Provider `json:"providers"`
	Path            string              `json:"-"`
}

type Language struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Recording struct {
	SilenceDurationMS int     `json:"silence_duration_ms"`
	SpeechThreshold   float64 `json:"speech_threshold"`
	MaxDurationSecs   int     `json:"max_duration_seconds"`
	MaxFrameBytes     int64   `json:"max_frame_bytes"`
	MaxAudioBytes     int64   `json:"max_audio_bytes"`
}

type Provider struct {
	Type                  string  `json:"type"`
	Enabled               bool    `json:"enabled"`
	BaseURL               string  `json:"base_url"`
	APIKey                string  `json:"api_key"`
	RequestTimeoutSeconds int     `json:"request_timeout_seconds"`
	Models                []Model `json:"models"`
}

type Model struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	SampleRate      int    `json:"sample_rate"`
	Mode            string `json:"mode"`
	InputAudioCodec string `json:"input_audio_codec"`
	Default         bool   `json:"default"`
}

type Selection struct {
	ProviderID string
	Provider   Provider
	Model      Model
	Language   Language
}

type SanitizedProvider struct {
	Type    string  `json:"type"`
	Enabled bool    `json:"enabled"`
	Models  []Model `json:"models"`
}

type SanitizedConfig struct {
	Enabled         bool                         `json:"enabled"`
	DefaultProvider string                       `json:"default_provider,omitempty"`
	Languages       []Language                   `json:"languages"`
	Recording       Recording                    `json:"recording"`
	Providers       map[string]SanitizedProvider `json:"providers"`
}

func Load(baseDir string) (*Config, error) {
	path := os.Getenv("BS_AI_VOICE_PATH")
	if path == "" {
		path = filepath.Join(baseDir, defaultFile)
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return LoadPath(path)
}

// LoadPath loads an explicit voice config path, bypassing environment path
// overrides. A missing file is a valid disabled configuration.
func LoadPath(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read voice config: %w", err)
	}
	return parseBytes(content, path)
}

// ValidateBytes validates a candidate without publishing or opening a network
// connection.
func ValidateBytes(content []byte) error {
	_, err := parseBytes(content, defaultFile)
	return err
}

func parseBytes(content []byte, path string) (*Config, error) {
	config := &Config{Path: path}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("parse voice config: %w", err)
	}
	// Reject a second JSON value; Decoder.Decode alone would otherwise accept
	// a valid object followed by unrelated content.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("parse voice config: multiple JSON values")
		}
		return nil, fmt.Errorf("parse voice config: %w", err)
	}
	config.Path = path
	if err := config.defaultsAndValidate(); err != nil {
		return nil, fmt.Errorf("validate voice config: %w", err)
	}
	return config, nil
}

func (c *Config) defaultsAndValidate() error {
	if !c.Enabled {
		return nil
	}
	if c.Recording.SilenceDurationMS == 0 {
		c.Recording.SilenceDurationMS = 1400
	}
	if c.Recording.SpeechThreshold == 0 {
		c.Recording.SpeechThreshold = .025
	}
	if c.Recording.MaxDurationSecs == 0 {
		c.Recording.MaxDurationSecs = 120
	}
	if c.Recording.MaxFrameBytes == 0 {
		c.Recording.MaxFrameBytes = 64 * 1024
	}
	if c.Recording.MaxAudioBytes == 0 {
		c.Recording.MaxAudioBytes = 4 * 1024 * 1024
	}
	if c.Recording.SilenceDurationMS < 200 || c.Recording.SilenceDurationMS > 10000 || c.Recording.SpeechThreshold <= 0 || c.Recording.SpeechThreshold > 1 || c.Recording.MaxDurationSecs < 1 || c.Recording.MaxDurationSecs > 300 || c.Recording.MaxFrameBytes < 1024 || c.Recording.MaxFrameBytes > 1024*1024 || c.Recording.MaxAudioBytes < c.Recording.MaxFrameBytes || c.Recording.MaxAudioBytes > 32*1024*1024 {
		return errors.New("recording limits are out of range")
	}
	langs := map[string]bool{}
	for _, l := range c.Languages {
		if l.Code == "" || l.Label == "" || langs[l.Code] {
			return errors.New("language codes and labels must be non-empty and codes unique")
		}
		langs[l.Code] = true
	}
	if len(langs) == 0 {
		return errors.New("at least one language is required")
	}
	if len(c.Providers) == 0 {
		return errors.New("at least one provider is required")
	}
	firstProvider := ""
	firstEnabled := ""
	for id, p := range c.Providers {
		if id == "" {
			return errors.New("provider ID is empty")
		}
		if firstProvider == "" || id < firstProvider {
			firstProvider = id
		}
		switch p.Type {
		case providerTypeSarvam:
			if err := validateSarvamProvider(id, &p); err != nil {
				return err
			}
		case providerTypeOpenRouterSTT:
			if err := validateOpenRouterSTTProvider(id, &p); err != nil {
				return err
			}
		default:
			return fmt.Errorf("provider %q has unsupported type %q", id, p.Type)
		}
		if p.RequestTimeoutSeconds == 0 {
			p.RequestTimeoutSeconds = 150
		}
		if p.RequestTimeoutSeconds < 5 || p.RequestTimeoutSeconds > 300 {
			return fmt.Errorf("provider %q timeout is out of range", id)
		}
		if p.Enabled {
			if strings.HasPrefix(p.APIKey, "env:") {
				name := strings.TrimPrefix(p.APIKey, "env:")
				if name == "" {
					return fmt.Errorf("provider %q has invalid api_key environment reference", id)
				}
				p.APIKey = os.Getenv(name)
			}
			if p.APIKey == "" {
				return fmt.Errorf("provider %q api_key is not configured", id)
			}
		}
		seen, defaultCount := map[string]bool{}, 0
		for i := range p.Models {
			m := &p.Models[i]
			if m.ID == "" || m.Label == "" || seen[m.ID] {
				return fmt.Errorf("provider %q has invalid or duplicate model", id)
			}
			seen[m.ID] = true
			if p.Type == providerTypeSarvam {
				if m.SampleRate != 8000 && m.SampleRate != 16000 {
					return fmt.Errorf("model %q has unsupported sample_rate", m.ID)
				}
				if !oneOf(m.Mode, "transcribe", "translate", "verbatim", "translit", "codemix") {
					return fmt.Errorf("model %q has unsupported mode", m.ID)
				}
				if m.InputAudioCodec != "pcm_s16le" {
					return fmt.Errorf("model %q must use pcm_s16le", m.ID)
				}
			}
			if m.Default {
				defaultCount++
			}
		}
		if len(p.Models) == 0 || defaultCount > 1 {
			return fmt.Errorf("provider %q must have models and at most one default", id)
		}
		if defaultCount == 0 {
			p.Models[0].Default = true
		}
		c.Providers[id] = p
		if p.Enabled && (firstEnabled == "" || id < firstEnabled) {
			firstEnabled = id
		}
	}
	if firstEnabled == "" {
		return errors.New("at least one provider must be enabled")
	}
	if c.DefaultProvider == "" {
		c.DefaultProvider = firstEnabled
	}
	if p, ok := c.Providers[c.DefaultProvider]; !ok || !p.Enabled {
		return errors.New("default_provider does not resolve to an enabled provider")
	}
	return nil
}

func (c *Config) Select(providerID, modelID, languageCode string) (Selection, error) {
	if c == nil || !c.Enabled {
		return Selection{}, errors.New("voice typing is disabled")
	}
	p, ok := c.Providers[providerID]
	if !ok || !p.Enabled || (p.Type != providerTypeSarvam && p.Type != providerTypeOpenRouterSTT) {
		return Selection{}, errors.New("invalid voice provider")
	}
	var model *Model
	for i := range p.Models {
		if p.Models[i].ID == modelID {
			model = &p.Models[i]
			break
		}
	}
	if model == nil {
		return Selection{}, errors.New("invalid voice model")
	}
	var language *Language
	for i := range c.Languages {
		if c.Languages[i].Code == languageCode {
			language = &c.Languages[i]
			break
		}
	}
	if language == nil {
		return Selection{}, errors.New("invalid voice language")
	}
	return Selection{ProviderID: providerID, Provider: p, Model: *model, Language: *language}, nil
}

func (c *Config) Sanitized() SanitizedConfig {
	out := SanitizedConfig{Enabled: c != nil && c.Enabled, Languages: []Language{}, Providers: map[string]SanitizedProvider{}}
	if c == nil || !c.Enabled {
		return out
	}
	out.DefaultProvider, out.Languages, out.Recording = c.DefaultProvider, c.Languages, c.Recording
	for id, p := range c.Providers {
		out.Providers[id] = SanitizedProvider{Type: p.Type, Enabled: p.Enabled, Models: p.Models}
	}
	return out
}

func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}
func isLoopback(host string) bool { return host == "localhost" || host == "127.0.0.1" || host == "::1" }

// validateSarvamProvider validates a Sarvam streaming provider's base URL.
func validateSarvamProvider(id string, p *Provider) error {
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "wss" && !(u.Scheme == "ws" && isLoopback(u.Hostname()))) {
		return fmt.Errorf("provider %q has unsafe base_url", id)
	}
	return nil
}

// validateOpenRouterSTTProvider validates an OpenRouter STT provider's base URL.
func validateOpenRouterSTTProvider(id string, p *Provider) error {
	if p.BaseURL == "" {
		p.BaseURL = defaultOpenRouterBaseURL
	}
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Host == "" || u.Scheme != "https" {
		return fmt.Errorf("provider %q has unsafe base_url (must be https)", id)
	}
	return nil
}

// OpenRouterLanguageCode converts a BCP-47 locale tag (e.g. "en-IN") to an
// ISO-639-1 code (e.g. "en") suitable for the OpenRouter STT API. Returns
// the original code if it is already a simple two-letter code.
func OpenRouterLanguageCode(locale string) string {
	if len(locale) >= 2 && unicode.IsLetter(rune(locale[0])) && unicode.IsLetter(rune(locale[1])) {
		if len(locale) == 2 {
			return locale
		}
		if locale[2] == '-' || locale[2] == '_' {
			return locale[:2]
		}
	}
	return locale
}

// IsOpenRouterSTT reports whether the provider uses the OpenRouter batch STT API.
func (p Provider) IsOpenRouterSTT() bool { return p.Type == providerTypeOpenRouterSTT }

// IsSarvam reports whether the provider uses the Sarvam WebSocket streaming API.
func (p Provider) IsSarvam() bool { return p.Type == providerTypeSarvam }

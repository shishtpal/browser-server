package videos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const modelsFile = "bs-ai-video-models.json"

// Load resolves and parses the video config next to the given base directory.
func Load(base string) (Config, error) {
	return LoadPath(filepath.Join(base, modelsFile))
}

// LoadPath loads an explicit video config. A missing file is a valid disabled
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
		return config, fmt.Errorf("parse video models: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return config, errors.New("parse video models: multiple JSON values")
		}
		return config, fmt.Errorf("parse video models: %w", err)
	}
	config.Path = path
	if config.Providers == nil {
		config.Providers = map[string]Provider{}
	}
	if !config.Enabled {
		return config, nil
	}
	if config.DefaultProvider == "" {
		return config, errors.New("video default_provider is required")
	}
	if _, ok := config.Providers[config.DefaultProvider]; !ok {
		return config, errors.New("video default_provider is not configured")
	}
	for name, provider := range config.Providers {
		switch provider.Type {
		case "agnes_video":
			if provider.BaseURL == "" {
				provider.BaseURL = "https://apihub.agnes-ai.com"
			}
			// Normalize to the origin (no path). A trailing "/v1" is stripped so
			// both styles of config produce the same endpoints: create posts to
			// {origin}/v1/videos and the result poll hits {origin}/agnesapi.
			provider.BaseURL = strings.TrimRight(provider.BaseURL, "/")
			if before, ok := strings.CutSuffix(provider.BaseURL, "/v1"); ok {
				provider.BaseURL = before
			}
		case "openrouter_video":
			if provider.BaseURL == "" {
				provider.BaseURL = "https://openrouter.ai"
			}
			// OpenRouter's video API lives under {origin}/api/v1/videos; keep
			// the origin so no path prefix is double-applied.
			provider.BaseURL = strings.TrimRight(provider.BaseURL, "/")
		default:
			return config, fmt.Errorf("video provider %q has unsupported type %q", name, provider.Type)
		}
		if provider.APIKey == "" {
			return config, fmt.Errorf("video provider %q api_key is required", name)
		}
		if env, ok := strings.CutPrefix(provider.APIKey, "env:"); ok {
			provider.APIKey = os.Getenv(env)
			if provider.APIKey == "" {
				return config, fmt.Errorf("video provider %q API key environment variable %q is empty", name, env)
			}
		}
		if provider.RequestTimeoutSeconds == 0 {
			// Video rendering can take many minutes; keep this independent from
			// the shorter chat completion timeout.
			provider.RequestTimeoutSeconds = 900
		}
		if len(provider.Models) == 0 {
			return config, fmt.Errorf("video provider %q needs models", name)
		}
		for _, model := range provider.Models {
			if err := validateModel(model); err != nil {
				return config, fmt.Errorf("video provider %q model %q: %w", name, model.ID, err)
			}
		}
		config.Providers[name] = provider
	}
	return config, nil
}

func validateModel(m Model) error {
	if m.ID == "" {
		return errors.New("id is required")
	}
	if len(m.Parameters) == 0 {
		return errors.New("parameters are required")
	}
	seen := map[string]bool{}
	for _, p := range m.Parameters {
		if p.Key == "" {
			return errors.New("parameter key is required")
		}
		if p.Label == "" {
			return errors.New("parameter label is required")
		}
		if seen[p.Key] {
			return fmt.Errorf("duplicate parameter key %q", p.Key)
		}
		seen[p.Key] = true
		switch p.Type {
		case "text", "textarea", "number", "select", "boolean", "image_urls":
		default:
			return fmt.Errorf("parameter %q has unsupported type %q", p.Key, p.Type)
		}
		if p.Type == "select" && len(p.Options) == 0 {
			return fmt.Errorf("parameter %q is a select with no options", p.Key)
		}
		if p.Type == "number" {
			if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
				return fmt.Errorf("parameter %q min exceeds max", p.Key)
			}
		}
	}
	return nil
}

// ParamSpec declares one tweakable generation option for a model. The frontend
// renders a field for every spec, so adding provider-specific options is a
// config change rather than a UI change.
type ParamSpec struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Group    string   `json:"group,omitempty"`
	Default  any      `json:"default,omitempty"`
	Required bool     `json:"required,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Step     *float64 `json:"step,omitempty"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
}

// Model describes a single video-generation model exposed by a provider.
type Model struct {
	ID         string      `json:"id"`
	Label      string      `json:"label"`
	Default    bool        `json:"default"`
	Parameters []ParamSpec `json:"parameters"`
}

// Provider groups the connection details and model catalog for one vendor.
type Provider struct {
	Type                  string  `json:"type"`
	BaseURL               string  `json:"base_url"`
	APIKey                string  `json:"api_key"`
	RequestTimeoutSeconds int     `json:"request_timeout_seconds"`
	Models                []Model `json:"models"`
	// OpenRouterSiteURL and OpenRouterAppName carry the attribution values from
	// bs-ai-config.json's openrouter section (injected by the service layer, not
	// part of this file). They are sent as HTTP-Referer/Referer and X-Title on
	// requests to OpenRouter so the app shows up in its rankings.
	OpenRouterSiteURL string `json:"-"`
	OpenRouterAppName string `json:"-"`
}

// Config is the parsed bs-ai-video-models.json document.
type Config struct {
	Enabled         bool                `json:"enabled"`
	DefaultProvider string              `json:"default_provider"`
	DBPath          string              `json:"db_path"`
	VideoDir        string              `json:"video_dir"`
	Providers       map[string]Provider `json:"providers"`
	Path            string              `json:"-"`
	// OpenRouterSiteURL and OpenRouterAppName are the attribution values from
	// bs-ai-config.json's openrouter section, injected by the bootstrap layer.
	OpenRouterSiteURL string `json:"-"`
	OpenRouterAppName string `json:"-"`
}

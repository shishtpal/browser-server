package images

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

const modelsFile = "bs-ai-image-models.json"

// Load resolves and parses the image config next to the given base directory.
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
		if provider.Type != "gemini_interactions" && provider.Type != "openrouter_images" && provider.Type != "agnes_images" {
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
		if provider.BaseURL == "" && provider.Type == "agnes_images" {
			provider.BaseURL = "https://apihub.agnes-ai.com/v1"
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

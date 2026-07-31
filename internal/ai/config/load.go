package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type modelsFile struct {
	Providers map[string]ProviderConfig `json:"providers"`
}

func Load() (*Config, error) {
	path := os.Getenv("BS_AI_CONFIG_PATH")
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(wd, defaultConfigFile)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{
				Enabled:     false,
				Path:        path,
				CORSEnabled: true,
				Providers:   map[string]ProviderConfig{},
			}, nil
		}
		return nil, fmt.Errorf("read AI config: %w", err)
	}

	cfg := &Config{Enabled: true, Path: path}
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse AI config: %w", err)
	}
	var mainRaw map[string]json.RawMessage
	_ = json.Unmarshal(content, &mainRaw)

	// Resolve models file path
	modelsPath := os.Getenv("BS_AI_MODELS_PATH")
	if modelsPath == "" {
		modelsPath = filepath.Join(filepath.Dir(path), defaultModelsFile)
	}
	cfg.ModelsPath = modelsPath

	modelsContent, err := os.ReadFile(modelsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.Enabled = false
			log.Printf("AI disabled: no models file at %s", modelsPath)
			return cfg, nil
		}
		return nil, fmt.Errorf("read AI models: %w", err)
	}

	var mf modelsFile
	if err := json.Unmarshal(modelsContent, &mf); err != nil {
		return nil, fmt.Errorf("parse AI models: %w", err)
	}
	var modelsRaw map[string]json.RawMessage
	_ = json.Unmarshal(modelsContent, &modelsRaw)

	cfg.Providers = mf.Providers
	applyDefaults(cfg, mainRaw, modelsRaw)
	if err := resolveSecrets(cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

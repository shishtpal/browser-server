package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(content, &raw)
	applyDefaults(cfg, raw)
	if err := resolveSecrets(cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

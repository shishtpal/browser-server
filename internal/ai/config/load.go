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
		exeDir, err := ExecutableDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(exeDir, defaultConfigFile)
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

	// Resolve models file path. The sibling default is already anchored to
	// the binary directory because `path` is, so no extra work is needed here.
	modelsPath := os.Getenv("BS_AI_MODELS_PATH")
	if modelsPath == "" {
		modelsPath = filepath.Join(filepath.Dir(path), defaultModelsFile)
	}
	cfg.ModelsPath = modelsPath
	if !cfg.Enabled {
		cfg.Providers = map[string]ProviderConfig{}
		applyDefaults(cfg, mainRaw, nil)
		return cfg, nil
	}

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
	if err := validateStorage(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ValidateBytes validates a candidate main config together with its sibling
// model catalog. It applies the exact startup defaults, secret resolution, and
// semantic checks without constructing providers, stores, tools, or workers.
func ValidateBytes(mainContent, modelsContent []byte, baseDir string) error {
	mainPath := filepath.Join(baseDir, defaultConfigFile)
	modelsPath := filepath.Join(baseDir, defaultModelsFile)
	cfg := &Config{Enabled: true, Path: mainPath, ModelsPath: modelsPath}
	if err := json.Unmarshal(mainContent, cfg); err != nil {
		return fmt.Errorf("parse AI config: %w", err)
	}
	var mainRaw map[string]json.RawMessage
	if err := json.Unmarshal(mainContent, &mainRaw); err != nil {
		return fmt.Errorf("parse AI config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	var models modelsFile
	if err := json.Unmarshal(modelsContent, &models); err != nil {
		return fmt.Errorf("parse AI models: %w", err)
	}
	var modelsRaw map[string]json.RawMessage
	if err := json.Unmarshal(modelsContent, &modelsRaw); err != nil {
		return fmt.Errorf("parse AI models: %w", err)
	}
	cfg.Providers = models.Providers
	applyDefaults(cfg, mainRaw, modelsRaw)
	if err := resolveSecrets(cfg); err != nil {
		return err
	}
	return validate(cfg)
}

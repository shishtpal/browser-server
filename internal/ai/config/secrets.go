package config

import (
	"fmt"
	"os"
	"strings"
)

func resolveSecrets(cfg *Config) error {
	for name, provider := range cfg.Providers {
		if strings.HasPrefix(provider.APIKey, "env:") {
			envName := strings.TrimSpace(strings.TrimPrefix(provider.APIKey, "env:"))
			if envName == "" {
				return fmt.Errorf("provider %q api_key env reference is empty", name)
			}
			value := os.Getenv(envName)
			if value == "" {
				return fmt.Errorf("provider %q api_key references unset environment variable %q", name, envName)
			}
			provider.APIKey = value
			cfg.Providers[name] = provider
		}
	}
	var err error
	if cfg.WebSearch.Providers.Brave.Enabled {
		cfg.WebSearch.Providers.Brave.APIKey, err = resolveOptionalEnv("web_search provider \"brave\" api_key", cfg.WebSearch.Providers.Brave.APIKey)
		if err != nil {
			return err
		}
	}
	if cfg.WebSearch.Providers.Tavily.Enabled {
		cfg.WebSearch.Providers.Tavily.APIKey, err = resolveOptionalEnv("web_search provider \"tavily\" api_key", cfg.WebSearch.Providers.Tavily.APIKey)
		if err != nil {
			return err
		}
	}
	if cfg.WebSearch.Providers.Google.Enabled {
		cfg.WebSearch.Providers.Google.APIKey, err = resolveOptionalEnv("web_search provider \"google\" api_key", cfg.WebSearch.Providers.Google.APIKey)
		if err != nil {
			return err
		}
		cfg.WebSearch.Providers.Google.SearchEngineID, err = resolveOptionalEnv("web_search provider \"google\" search_engine_id", cfg.WebSearch.Providers.Google.SearchEngineID)
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveOptionalEnv(field, value string) (string, error) {
	if !strings.HasPrefix(value, "env:") {
		return value, nil
	}
	envName := strings.TrimSpace(strings.TrimPrefix(value, "env:"))
	if envName == "" {
		return "", fmt.Errorf("%s env reference is empty", field)
	}
	resolved := os.Getenv(envName)
	if resolved == "" {
		return "", fmt.Errorf("%s references unset environment variable %q", field, envName)
	}
	return resolved, nil
}

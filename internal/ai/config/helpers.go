package config

import "path/filepath"

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (cfg *Config) DefaultModel(providerName string) (ModelConfig, bool) {
	provider, ok := cfg.Providers[providerName]
	if !ok || len(provider.Models) == 0 {
		return ModelConfig{}, false
	}
	for _, model := range provider.Models {
		if model.Default {
			return model, true
		}
	}
	return provider.Models[0], true
}

func (cfg *Config) FindModel(providerName, modelID string) (ProviderConfig, ModelConfig, bool) {
	provider, ok := cfg.Providers[providerName]
	if !ok {
		return ProviderConfig{}, ModelConfig{}, false
	}
	for _, model := range provider.Models {
		if model.ID == modelID {
			return provider, model, true
		}
	}
	return ProviderConfig{}, ModelConfig{}, false
}

func (cfg *Config) ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(filepath.Dir(cfg.Path), p)
}

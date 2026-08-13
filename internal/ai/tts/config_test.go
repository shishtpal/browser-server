package tts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `{
  "enabled": true,
  "default_provider": "openrouter",
  "providers": {
    "openrouter": {
      "type": "openrouter_speech",
      "base_url": "https://openrouter.ai/api/v1",
      "api_key": "env:TTS_TEST_KEY",
      "request_timeout_seconds": 120,
      "models": [
        {
          "id": "deepgram/flux-tts:free",
          "label": "Deepgram Flux TTS (free)",
          "default": true,
          "voices": [{"id": "flux-gemma-en", "label": "Gemma (EN)"}]
        }
      ]
    }
  }
}`

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingIsDisabled(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("missing file should return a disabled config")
	}
	if cfg.Providers == nil {
		t.Fatal("Providers should be a non-nil empty map")
	}
}

func TestLoadDisabledSkipsValidation(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, modelsFile, `{"enabled": false}`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("disabled config was treated as enabled")
	}
}

func TestLoadResolvesEnvKeyAndDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, modelsFile, `{
  "enabled": true,
  "default_provider": "openrouter",
  "providers": {
    "openrouter": {
      "type": "openrouter_speech",
      "api_key": "env:TTS_TEST_KEY",
      "models": [{"id": "deepgram/flux-tts:free", "label": "Flux", "default": true}]
    }
  }
}`)
	t.Setenv("TTS_TEST_KEY", "secret-key")
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Providers["openrouter"]
	if p.APIKey != "secret-key" {
		t.Fatalf("api_key = %q, want resolved secret", p.APIKey)
	}
	if p.BaseURL != defaultBaseURL {
		t.Fatalf("base_url = %q, want default", p.BaseURL)
	}
	if p.RequestTimeoutSeconds != defaultTimeoutSeconds {
		t.Fatalf("timeout = %d, want %d", p.RequestTimeoutSeconds, defaultTimeoutSeconds)
	}
}

func TestLoadRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"enabled without default_provider": `{
  "enabled": true,
  "providers": {
    "openrouter": {
      "type": "openrouter_speech",
      "api_key": "k",
      "models": [{"id": "m", "label": "M"}]
    }
  }
}`,
		"unknown default_provider": `{
  "enabled": true,
  "default_provider": "missing",
  "providers": {
    "openrouter": {
      "type": "openrouter_speech",
      "api_key": "k",
      "models": [{"id": "m", "label": "M"}]
    }
  }
}`,
		"unsupported type": `{
  "enabled": true,
  "default_provider": "openrouter",
  "providers": {
    "openrouter": {
      "type": "openai_compatible",
      "api_key": "k",
      "models": [{"id": "m", "label": "M"}]
    }
  }
}`,
		"empty models": `{
  "enabled": true,
  "default_provider": "openrouter",
  "providers": {
    "openrouter": {
      "type": "openrouter_speech",
      "api_key": "k",
      "models": []
    }
  }
}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, modelsFile, content)
			if _, err := Load(dir); err == nil {
				t.Fatal("invalid config was accepted")
			}
		})
	}
}

func TestLoadRejectsEmptyEnvKey(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, modelsFile, validConfig)
	t.Setenv("TTS_TEST_KEY", "")
	_, err := Load(dir)
	if err == nil {
		t.Fatal("empty env key was accepted")
	}
	if !strings.Contains(err.Error(), "TTS_TEST_KEY") {
		t.Fatalf("error %q should mention the env var", err)
	}
}

func TestNewDisabledReturnsNil(t *testing.T) {
	svc, err := New(Config{Enabled: false}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if svc != nil {
		t.Fatal("disabled config should return a nil service")
	}
}

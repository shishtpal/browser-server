package voice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `{
  "enabled": true,
  "languages": [{"code":"unknown","label":"Auto"}],
  "providers": {"sarvam":{"type":"sarvam_streaming","enabled":true,"base_url":"wss://example.com/stt","api_key":"env:VOICE_TEST_KEY","models":[{"id":"model","label":"Model","sample_rate":16000,"mode":"transcribe","input_audio_codec":"pcm_s16le"}]}}
}`

func TestLoadMissingIsDisabled(t *testing.T) {
	t.Setenv("BS_AI_VOICE_PATH", filepath.Join(t.TempDir(), "missing.json"))
	c, err := Load(t.TempDir())
	if err != nil || c.Enabled || c.Sanitized().Enabled {
		t.Fatalf("Load() = %#v, %v; want disabled config", c, err)
	}
}

func TestLoadDefaultsResolvesSecretAndSanitizes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "voice.json")
	if err := os.WriteFile(path, []byte(validConfig), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BS_AI_VOICE_PATH", path)
	t.Setenv("VOICE_TEST_KEY", "secret")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultProvider != "sarvam" || !c.Providers["sarvam"].Models[0].Default || c.Recording.MaxFrameBytes != 65536 || c.Providers["sarvam"].APIKey != "secret" {
		t.Fatalf("defaults or secret not applied: %#v", c)
	}
	b := c.Sanitized()
	if b.Providers["sarvam"].Models[0].SampleRate != 16000 {
		t.Fatalf("sanitized model missing capture details: %#v", b)
	}
	if strings.Contains(string(mustJSON(t, b)), "secret") || strings.Contains(string(mustJSON(t, b)), "example.com") {
		t.Fatal("sanitized config exposed a secret or upstream URL")
	}
	if _, err := c.Select("sarvam", "bad", "unknown"); err == nil {
		t.Fatal("unknown model accepted")
	}
	if _, err := c.Select("missing", "model", "unknown"); err == nil {
		t.Fatal("missing provider accepted")
	}
	if _, err := c.Select("sarvam", "model", "missing"); err == nil {
		t.Fatal("missing language accepted")
	}
}

func TestSelectDisabledProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "voice.json")
	content := `{
  "enabled": true,
  "languages": [{"code":"unknown","label":"Auto"}],
  "providers": {
    "disabled":{"type":"sarvam_streaming","enabled":false,"base_url":"wss://example.com/stt","api_key":"env:VOICE_TEST_KEY","models":[{"id":"disabled-model","label":"Disabled","sample_rate":16000,"mode":"transcribe","input_audio_codec":"pcm_s16le"}]},
    "sarvam":{"type":"sarvam_streaming","enabled":true,"base_url":"wss://example.com/stt","api_key":"env:VOICE_TEST_KEY","models":[{"id":"model","label":"Model","sample_rate":16000,"mode":"transcribe","input_audio_codec":"pcm_s16le"}]}
  }
}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BS_AI_VOICE_PATH", path)
	t.Setenv("VOICE_TEST_KEY", "secret")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultProvider != "sarvam" {
		t.Fatalf("expected default provider to fall back to enabled provider, got %q", c.DefaultProvider)
	}
	if _, err := c.Select("disabled", "disabled-model", "unknown"); err == nil {
		t.Fatal("disabled provider accepted")
	}
}

func TestLoadRejectsUnsafeAndMissingSecret(t *testing.T) {
	for name, content := range map[string]string{
		"missing secret": validConfig,
		"unsafe URL":     strings.Replace(validConfig, "wss://example.com/stt", "ws://example.com/stt", 1),
		"sample rate":    strings.Replace(validConfig, "16000", "44100", 1),
		"audio codec":    strings.Replace(validConfig, "pcm_s16le", "wav", 1),
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "voice.json")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("BS_AI_VOICE_PATH", path)
			t.Setenv("VOICE_TEST_KEY", "")
			if name != "missing secret" {
				t.Setenv("VOICE_TEST_KEY", "secret")
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("invalid config was accepted")
			}
		})
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

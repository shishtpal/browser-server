package config

import (
	"encoding/json"
	"testing"
)

func baseOCRConfig() *Config {
	return &Config{
		DefaultProvider: "openrouter.ai",
		Providers: map[string]ProviderConfig{
			"openrouter.ai": {
				BaseURL: "https://openrouter.ai/api/v1",
				Models: []ModelConfig{
					{ID: "openai/gpt-4o-mini", SupportsTools: true, SupportsVision: true, Default: true, MaxOutputTokens: 4096},
					{ID: "text-only", SupportsTools: true, Default: false, MaxOutputTokens: 2048},
				},
			},
		},
		Tools: ToolsConfig{Allowed: []string{"ocr_image"}},
		OCR: OCRConfig{
			Enabled:         true,
			DefaultProvider: "openrouter.ai",
			DefaultModel:    "openai/gpt-4o-mini",
			MaxInputBytes:   20 * 1024 * 1024,
			Poppler: OCRPopplerConfig{
				DPI:            150,
				Format:         "png",
				MaxPages:       32,
				TimeoutSeconds: 120,
			},
		},
	}
}

func TestValidateOCRDisabledSkipsChecks(t *testing.T) {
	cfg := baseOCRConfig()
	cfg.OCR = OCRConfig{Enabled: false}
	if err := validateOCR(cfg); err != nil {
		t.Fatalf("disabled OCR must pass validation: %v", err)
	}
}

func TestValidateOCRValidConfig(t *testing.T) {
	if err := validateOCR(baseOCRConfig()); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}

func TestValidateOCRRejectsNonVisionDefaultModel(t *testing.T) {
	cfg := baseOCRConfig()
	cfg.OCR.DefaultModel = "text-only"
	if err := validateOCR(cfg); err == nil {
		t.Fatal("expected error for non-vision default model")
	}
}

func TestValidateOCRRejectsUnknownProvider(t *testing.T) {
	cfg := baseOCRConfig()
	cfg.OCR.DefaultProvider = "nope"
	if err := validateOCR(cfg); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestValidateOCRPopplerBounds(t *testing.T) {
	cases := []func(*Config){
		func(c *Config) { c.OCR.MaxInputBytes = 1024 },
		func(c *Config) { c.OCR.OutputDir = "../escape" },
		func(c *Config) { c.OCR.Poppler.DPI = 10 },
		func(c *Config) { c.OCR.Poppler.Format = "bmp" },
		func(c *Config) { c.OCR.Poppler.MaxPages = 0 },
		func(c *Config) { c.OCR.Poppler.MaxPages = 500 },
		func(c *Config) { c.OCR.Poppler.TimeoutSeconds = 1 },
		func(c *Config) { c.OCR.Poppler.TimeoutSeconds = 3600 },
	}
	for i, mutate := range cases {
		cfg := baseOCRConfig()
		mutate(cfg)
		if err := validateOCR(cfg); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestValidateOCRWarnsWhenToolNotAllowed(t *testing.T) {
	cfg := baseOCRConfig()
	cfg.Tools.Allowed = []string{"read_file"}
	// Should not error — only warn — because tools.allowed is operator policy.
	if err := validateOCR(cfg); err != nil {
		t.Fatalf("missing ocr_image in tools.allowed must warn, not fail: %v", err)
	}
}

func TestApplyOCRDefaults(t *testing.T) {
	cfg := baseOCRConfig()
	cfg.OCR = OCRConfig{}
	raw := map[string]json.RawMessage{} // operator omitted everything
	cfg.OCR.Enabled = true
	applyOCRDefaults(cfg, raw)
	if cfg.OCR.DefaultProvider != "openrouter.ai" {
		t.Fatalf("default provider = %q", cfg.OCR.DefaultProvider)
	}
	if cfg.OCR.DefaultModel != "openai/gpt-4o-mini" {
		t.Fatalf("default model = %q", cfg.OCR.DefaultModel)
	}
	if cfg.OCR.MaxInputBytes != 20*1024*1024 {
		t.Fatalf("max input bytes = %d", cfg.OCR.MaxInputBytes)
	}
	if cfg.OCR.Poppler.DPI != 150 || cfg.OCR.Poppler.Format != "png" || cfg.OCR.Poppler.MaxPages != 32 || cfg.OCR.Poppler.TimeoutSeconds != 120 {
		t.Fatalf("poppler defaults = %+v", cfg.OCR.Poppler)
	}
}

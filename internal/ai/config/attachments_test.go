package config

import (
	"strings"
	"testing"
)

// attachmentModels is a small catalog that marks one vision-capable model and
// one text-only model so supports_vision exposure can be asserted.
func attachmentModels() string {
	return `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "sk-test",
				"models": [
					{"id": "vision-model", "label": "Vision", "supports_tools": true, "supports_vision": true, "default": true, "max_output_tokens": 4096},
					{"id": "text-model", "label": "Text", "supports_tools": true, "supports_vision": false, "max_output_tokens": 4096}
				]
			}
		}
	}`
}

func TestAttachmentDefaultsAppliedWhenOmitted(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, minimalProviderConfig(), attachmentModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	a := cfg.Chat.Attachments
	if !a.Enabled {
		t.Fatalf("expected attachments enabled by default")
	}
	if len(a.AllowedMIMETypes) != 4 || a.AllowedMIMETypes[0] != "image/png" {
		t.Fatalf("default MIME list = %+v", a.AllowedMIMETypes)
	}
	if a.MaxImages != 5 || a.MaxImageBytes != 5*1024*1024 || a.MaxTotalBytes != 20*1024*1024 || a.RetentionHours != 24 {
		t.Fatalf("attachment defaults not applied: %+v", a)
	}
}

func TestAttachmentDefaultsPreserveExplicitValues(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"chat": {
			"attachments": {
				"enabled": true,
				"allowed_mime_types": ["image/png"],
				"max_images": 3,
				"max_image_bytes": 131072,
				"max_total_bytes": 262144,
				"retention_hours": 12
			}
		}
	}`, attachmentModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	a := cfg.Chat.Attachments
	if a.MaxImages != 3 || a.MaxImageBytes != 131072 || a.MaxTotalBytes != 262144 || a.RetentionHours != 12 {
		t.Fatalf("explicit attachment values not preserved: %+v", a)
	}
	if len(a.AllowedMIMETypes) != 1 || a.AllowedMIMETypes[0] != "image/png" {
		t.Fatalf("explicit MIME list not preserved: %+v", a.AllowedMIMETypes)
	}
}

func TestAttachmentValidationRejectsBadLimits(t *testing.T) {
	cases := []struct {
		name string
		cfg  ChatAttachmentsConfig
		want string
	}{
		{"empty mime types", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: nil, MaxImages: 5, MaxImageBytes: 1 << 20, MaxTotalBytes: 2 << 20, RetentionHours: 24}, "allowed_mime_types must not be empty"},
		{"unsupported mime", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/bmp"}, MaxImages: 5, MaxImageBytes: 1 << 20, MaxTotalBytes: 2 << 20, RetentionHours: 24}, "unsupported type"},
		{"too few images", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/png"}, MaxImages: 0, MaxImageBytes: 1 << 20, MaxTotalBytes: 2 << 20, RetentionHours: 24}, "max_images must be between 1 and 20"},
		{"too many images", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/png"}, MaxImages: 21, MaxImageBytes: 1 << 20, MaxTotalBytes: 2 << 20, RetentionHours: 24}, "max_images must be between 1 and 20"},
		{"image bytes too small", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/png"}, MaxImages: 5, MaxImageBytes: 1024, MaxTotalBytes: 2 << 20, RetentionHours: 24}, "max_image_bytes must be between 65536 and 10485760"},
		{"total below image bytes", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/png"}, MaxImages: 5, MaxImageBytes: 1 << 20, MaxTotalBytes: 1<<20 - 1, RetentionHours: 24}, "max_total_bytes must be at least max_image_bytes"},
		{"retention too small", ChatAttachmentsConfig{Enabled: true, AllowedMIMETypes: []string{"image/png"}, MaxImages: 5, MaxImageBytes: 1 << 20, MaxTotalBytes: 2 << 20, RetentionHours: 0}, "retention_hours must be between 1 and 720"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAttachments(tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateAttachments err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestAttachmentValidationSkipsWhenDisabled(t *testing.T) {
	// A disabled attachments section must not reject an empty MIME list, so an
	// operator can disable the feature without also configuring limits.
	cfg := ChatAttachmentsConfig{Enabled: false}
	if err := validateAttachments(cfg); err != nil {
		t.Fatalf("disabled attachments should not validate, got %v", err)
	}
}

func TestSupportsVisionExposedThroughSanitizedConfig(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, minimalProviderConfig(), attachmentModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	out := cfg.Sanitized(nil)
	provider := out.Providers["openrouter"]
	var vision, text SanitizedModel
	for _, m := range provider.Models {
		if m.ID == "vision-model" {
			vision = m
		}
		if m.ID == "text-model" {
			text = m
		}
	}
	if !vision.SupportsVision {
		t.Fatalf("vision-model should expose supports_vision=true: %+v", vision)
	}
	if text.SupportsVision {
		t.Fatalf("text-model should expose supports_vision=false: %+v", text)
	}
	// Attachments limits must be surfaced to clients (no secrets here).
	if !out.Chat.Attachments.Enabled || out.Chat.Attachments.MaxImages != 5 {
		t.Fatalf("sanitized attachments not exposed: %+v", out.Chat.Attachments)
	}
}

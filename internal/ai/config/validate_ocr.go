package config

import (
	"fmt"
	"log"
	"strings"
)

// validateOCR checks the ocr section for internal consistency and references
// into the providers/models file (like the chat attachment vision check).
func validateOCR(cfg *Config) error {
	if !cfg.OCR.Enabled {
		return nil
	}
	if cfg.OCR.MaxInputBytes < 64*1024 || cfg.OCR.MaxInputBytes > 128*1024*1024 {
		return fmt.Errorf("ocr.max_input_bytes must be between 65536 and 134217728")
	}
	if strings.Contains(cfg.OCR.OutputDir, "..") {
		return fmt.Errorf("ocr.output_dir must not contain '..'")
	}
	_, model, ok := cfg.FindModel(cfg.OCR.DefaultProvider, cfg.OCR.DefaultModel)
	if !ok {
		return fmt.Errorf("ocr.default_provider/ocr.default_model (%s/%s) must reference a model configured in the models file", cfg.OCR.DefaultProvider, cfg.OCR.DefaultModel)
	}
	if !model.SupportsVision {
		return fmt.Errorf("ocr.default_model %q must have supports_vision: true in the models file", cfg.OCR.DefaultModel)
	}
	p := cfg.OCR.Poppler
	if p.DPI < 72 || p.DPI > 600 {
		return fmt.Errorf("ocr.poppler.dpi must be between 72 and 600")
	}
	switch p.Format {
	case "png", "ppm", "jpeg":
	default:
		return fmt.Errorf("ocr.poppler.format must be png, ppm, or jpeg")
	}
	if p.MaxPages < 1 || p.MaxPages > 256 {
		return fmt.Errorf("ocr.poppler.max_pages must be between 1 and 256")
	}
	if p.TimeoutSeconds < 5 || p.TimeoutSeconds > 600 {
		return fmt.Errorf("ocr.poppler.timeout_seconds must be between 5 and 600")
	}
	allowed := false
	for _, name := range cfg.Tools.Allowed {
		if name == "ocr_image" {
			allowed = true
			break
		}
	}
	if !allowed {
		// tools.allowed is explicit operator policy — never auto-append, only warn.
		log.Printf("WARN: ocr.enabled is true but %q is absent from tools.allowed; the ocr_image tool will not be exposed", "ocr_image")
	}
	return nil
}

package config

import "encoding/json"

// OCRConfig is loaded from the "ocr" section of bs-ai-config.json. It gates
// the built-in ocr_image tool (image OCR via a vision model, and PDF
// rasterization via the Poppler CLI).
type OCRConfig struct {
	Enabled         bool             `json:"enabled"`
	DefaultProvider string           `json:"default_provider"`
	DefaultModel    string           `json:"default_model"`
	MaxInputBytes   int              `json:"max_input_bytes"`
	OutputDir       string           `json:"output_dir"`
	Poppler         OCRPopplerConfig `json:"poppler"`
}

// OCRPopplerConfig configures the pdftoppm binary used by the
// pdf_to_images action.
type OCRPopplerConfig struct {
	Dir            string `json:"dir"`
	DPI            int    `json:"dpi"`
	Format         string `json:"format"`
	MaxPages       int    `json:"max_pages"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// applyOCRDefaults fills the ocr section with safe defaults when the operator
// omitted the whole object or individual fields.
func applyOCRDefaults(cfg *Config, mainRaw map[string]json.RawMessage) {
	if !nestedPresent(mainRaw, "ocr", "default_provider") {
		cfg.OCR.DefaultProvider = cfg.DefaultProvider
	}
	if !nestedPresent(mainRaw, "ocr", "default_model") {
		if m, ok := cfg.DefaultModel(cfg.OCR.DefaultProvider); ok {
			cfg.OCR.DefaultModel = m.ID
		}
	}
	if !nestedPresent(mainRaw, "ocr", "max_input_bytes") {
		cfg.OCR.MaxInputBytes = 20 * 1024 * 1024
	}
	if cfg.OCR.Poppler.DPI == 0 {
		cfg.OCR.Poppler.DPI = 150
	}
	if cfg.OCR.Poppler.Format == "" {
		cfg.OCR.Poppler.Format = "png"
	}
	if cfg.OCR.Poppler.MaxPages == 0 {
		cfg.OCR.Poppler.MaxPages = 32
	}
	if cfg.OCR.Poppler.TimeoutSeconds == 0 {
		cfg.OCR.Poppler.TimeoutSeconds = 120
	}
}

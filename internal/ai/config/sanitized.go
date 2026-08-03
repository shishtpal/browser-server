package config

type SanitizedConfig struct {
	Enabled         bool                         `json:"enabled"`
	DefaultProvider string                       `json:"default_provider,omitempty"`
	Providers       map[string]SanitizedProvider `json:"providers"`
	Tools           SanitizedTools               `json:"tools"`
	Chat            SanitizedChat                `json:"chat"`
}

type SanitizedProvider struct {
	Type    string           `json:"type"`
	Models  []SanitizedModel `json:"models"`
	Default string           `json:"default_model"`
}

type SanitizedModel struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	SupportsTools   bool   `json:"supports_tools"`
	SupportsVision  bool   `json:"supports_vision"`
	Default         bool   `json:"default"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

type SanitizedTools struct {
	Enabled       bool              `json:"enabled"`
	Allowed       []string          `json:"allowed"`
	Categories    map[string]string `json:"categories"`
	MaxIterations int               `json:"max_iterations"`
}

type SanitizedChat struct {
	MaxHistoryMessages int                    `json:"max_history_messages"`
	Stream             bool                   `json:"stream"`
	Temperature        float64                `json:"temperature"`
	Attachments        SanitizedChatAttachments `json:"attachments"`
}

// SanitizedChatAttachments exposes the upload limits (without any operator
// secrets) so browser clients can pre-validate selections against server rules.
type SanitizedChatAttachments struct {
	Enabled          bool     `json:"enabled"`
	AllowedMIMETypes []string `json:"allowed_mime_types"`
	MaxImages        int      `json:"max_images"`
	MaxImageBytes    int      `json:"max_image_bytes"`
	MaxTotalBytes    int      `json:"max_total_bytes"`
}

func (cfg *Config) Sanitized(categories map[string]string) SanitizedConfig {
	out := SanitizedConfig{
		Enabled:         cfg.Enabled,
		DefaultProvider: cfg.DefaultProvider,
		Providers:       map[string]SanitizedProvider{},
		Tools: SanitizedTools{
			Enabled:       cfg.Tools.Enabled,
			Allowed:       append([]string{}, cfg.Tools.Allowed...),
			Categories:    categories,
			MaxIterations: cfg.Tools.MaxIterations,
		},
		Chat: SanitizedChat{
			MaxHistoryMessages: cfg.Chat.MaxHistoryMessages,
			Stream:             cfg.Chat.Stream,
			Temperature:        cfg.Chat.Temperature,
			Attachments: SanitizedChatAttachments{
				Enabled:          cfg.Chat.Attachments.Enabled,
				AllowedMIMETypes: append([]string{}, cfg.Chat.Attachments.AllowedMIMETypes...),
				MaxImages:        cfg.Chat.Attachments.MaxImages,
				MaxImageBytes:    cfg.Chat.Attachments.MaxImageBytes,
				MaxTotalBytes:    cfg.Chat.Attachments.MaxTotalBytes,
			},
		},
	}
	if out.Tools.Categories == nil {
		out.Tools.Categories = map[string]string{}
	}
	for name, provider := range cfg.Providers {
		sanitized := SanitizedProvider{Type: provider.Type}
		for _, model := range provider.Models {
			label := model.Label
			if label == "" {
				label = model.ID
			}
			if model.Default {
				sanitized.Default = model.ID
			}
			sanitized.Models = append(sanitized.Models, SanitizedModel{
				ID:              model.ID,
				Label:           label,
				SupportsTools:   model.SupportsTools,
				SupportsVision:  model.SupportsVision,
				Default:         model.Default,
				MaxOutputTokens: model.MaxOutputTokens,
			})
		}
		if sanitized.Default == "" && len(sanitized.Models) > 0 {
			sanitized.Default = sanitized.Models[0].ID
		}
		out.Providers[name] = sanitized
	}
	return out
}

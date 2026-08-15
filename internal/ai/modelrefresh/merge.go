package modelrefresh

import "browser-server/internal/ai/config"

// Merge appends fetched models to the existing list. Existing entries keep
// their order and every field; only models whose ID is not already present are
// appended (deduplicated by ID). When the merged list has no default model and
// is non-empty, the first entry is marked as the default so the result
// satisfies the server's validation (exactly one default model). Existing
// default flags are never changed.
func Merge(existing []config.ModelConfig, fetched []ModelInfo) []config.ModelConfig {
	merged := make([]config.ModelConfig, 0, len(existing)+len(fetched))
	merged = append(merged, existing...)

	seen := make(map[string]bool, len(merged))
	for _, model := range merged {
		seen[model.ID] = true
	}

	for _, info := range fetched {
		if info.ID == "" || seen[info.ID] {
			continue
		}
		seen[info.ID] = true
		merged = append(merged, config.ModelConfig{
			ID:              info.ID,
			Label:           info.Label,
			SupportsTools:   info.SupportsTools,
			SupportsVision:  info.SupportsVision,
			MaxOutputTokens: info.MaxOutputTokens,
			Default:         false,
		})
	}

	if len(merged) > 0 && !hasDefault(merged) {
		merged[0].Default = true
	}
	return merged
}

func hasDefault(models []config.ModelConfig) bool {
	for _, model := range models {
		if model.Default {
			return true
		}
	}
	return false
}

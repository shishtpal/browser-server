package config

import "encoding/json"

func applyDefaults(cfg *Config, mainRaw, modelsRaw map[string]json.RawMessage) {
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	if _, present := mainRaw["cors_enabled"]; !present {
		cfg.CORSEnabled = true
	}
	if !nestedPresent(mainRaw, "tools", "max_iterations") {
		cfg.Tools.MaxIterations = 5
	}
	if !nestedPresent(mainRaw, "tools", "max_output") {
		cfg.Tools.MaxOutput = 32 * 1024
	}
	if !nestedPresent(mainRaw, "tools", "max_diff_output") {
		cfg.Tools.MaxDiffOutput = 0
	}
	if !nestedPresent(mainRaw, "tools", "git_timeout_seconds") {
		cfg.Tools.GitTimeoutSecs = 30
	}
	if cfg.WebSearch.DefaultProvider == "" {
		cfg.WebSearch.DefaultProvider = "auto"
	}
	if !nestedPresent(mainRaw, "web_search", "timeout_seconds") {
		cfg.WebSearch.TimeoutSeconds = 30
	}
	if !nestedPresent(mainRaw, "web_search", "max_results") {
		cfg.WebSearch.MaxResults = 10
	}
	if !nestedPresent(mainRaw, "web_search", "fallback") {
		cfg.WebSearch.Fallback = true
	}
	if !nestedPresent(mainRaw, "web_search", "cache_ttl_minutes") {
		cfg.WebSearch.CacheTTLMinutes = 5
	}
	if !nestedPresent(mainRaw, "web_search", "cache_max_entries") {
		cfg.WebSearch.CacheMaxEntries = 100
	}
	if cfg.Memory.Directory == "" {
		cfg.Memory.Directory = ".memory"
	}
	if cfg.Skills.Directory == "" {
		cfg.Skills.Directory = ".skills"
	}
	if cfg.Memory.PrimaryDir == "" {
		cfg.Memory.PrimaryDir = "memories"
	}
	if cfg.Memory.RefsDir == "" {
		cfg.Memory.RefsDir = "refs"
	}
	if cfg.Memory.CacheDir == "" {
		cfg.Memory.CacheDir = "cache"
	}
	if !nestedPresent(mainRaw, "memory", "max_file_size_kb") {
		cfg.Memory.MaxFileSizeKB = 1024
	}
	if !nestedPresent(mainRaw, "memory", "retention_days") {
		cfg.Memory.RetentionDays = 365
	}
	if !nestedPresent(mainRaw, "memory", "max_reference_depth") {
		cfg.Memory.MaxReferenceDepth = 5
	}
	if !nestedPresent(mainRaw, "memory", "cache_size_limit_mb") {
		cfg.Memory.CacheSizeLimitMB = 100
	}
	if cfg.Logging.DBPath == "" {
		cfg.Logging.DBPath = ".data/bs-ai.db"
	}
	if !nestedPresent(mainRaw, "logging", "retention_days") {
		cfg.Logging.RetentionDays = 60
	}
	if !nestedPresent(mainRaw, "logging", "max_payload_bytes") {
		cfg.Logging.MaxPayloadBytes = 1048576
	}
	if cfg.Chat.SystemPrompt == "" {
		cfg.Chat.SystemPrompt = "You are a helpful assistant integrated into browser-server."
	}
	if !nestedPresent(mainRaw, "chat", "max_history_messages") {
		cfg.Chat.MaxHistoryMessages = 30
	}
	if !nestedPresent(mainRaw, "chat", "temperature") {
		cfg.Chat.Temperature = 0.7
	}
	if !nestedPresent(mainRaw, "chat", "stream") {
		cfg.Chat.Stream = true
	}
	if !nestedPresent(mainRaw, "chat", "tool_retry_attempts") {
		cfg.Chat.ToolRetryAttempts = 5
	}
	if !nestedPresent(mainRaw, "chat", "tool_retry_delay_seconds") {
		cfg.Chat.ToolRetryDelaySeconds = 5
	}
	for name, provider := range cfg.Providers {
		if !providerFieldPresent(modelsRaw, name, "request_timeout_seconds") {
			provider.RequestTimeoutSeconds = 120
		}
		if !providerFieldPresent(modelsRaw, name, "retry_attempts") {
			provider.RetryAttempts = 10
		}
		if !providerFieldPresent(modelsRaw, name, "retry_delay_seconds") {
			provider.RetryDelaySeconds = 5
		}
		cfg.Providers[name] = provider
	}
	if !nestedPresent(mainRaw, "file_tools", "max_read_bytes") {
		cfg.FileTools.MaxReadBytes = 32768
	}
	if !nestedPresent(mainRaw, "file_tools", "max_line_read_bytes") {
		cfg.FileTools.MaxLineReadBytes = 65536
	}
	if !nestedPresent(mainRaw, "file_tools", "max_line_count") {
		cfg.FileTools.MaxLineCount = 5000
	}
	if !nestedPresent(mainRaw, "file_tools", "max_file_size_warn_mb") {
		cfg.FileTools.MaxFileSizeWarnMB = 100
	}
}

func nestedPresent(raw map[string]json.RawMessage, section, field string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw[section], &m) != nil {
		return false
	}
	_, ok := m[field]
	return ok
}

func providerFieldPresent(raw map[string]json.RawMessage, name, field string) bool {
	var p map[string]json.RawMessage
	if json.Unmarshal(raw["providers"], &p) != nil {
		return false
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(p[name], &m) != nil {
		return false
	}
	_, ok := m[field]
	return ok
}

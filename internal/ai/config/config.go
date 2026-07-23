package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NOTE: Must be relative to go compiled binary for portable app
const defaultConfigFile = "bs-ai-config.json"

type Config struct {
	Enabled         bool                      `json:"-"`
	Path            string                    `json:"-"`
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
	Tools           ToolsConfig               `json:"tools"`
	WebSearch       WebSearchConfig           `json:"web_search"`
	Memory          MemoryConfig              `json:"memory"`
	Skills          SkillsConfig              `json:"skills"`
	Logging         LoggingConfig             `json:"logging"`
	Chat            ChatConfig                `json:"chat"`
}

type SkillsConfig struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
}

type ProviderConfig struct {
	Type                  string        `json:"type"`
	BaseURL               string        `json:"base_url"`
	APIKey                string        `json:"api_key"`
	RequestTimeoutSeconds int           `json:"request_timeout_seconds"`
	RetryAttempts         int           `json:"retry_attempts"`
	RetryDelaySeconds     int           `json:"retry_delay_seconds"`
	Models                []ModelConfig `json:"models"`
}

type ModelConfig struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	SupportsTools   bool   `json:"supports_tools"`
	Default         bool   `json:"default"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

type ToolsConfig struct {
	Enabled       bool     `json:"enabled"`
	Allowed       []string `json:"allowed"`
	MaxIterations int      `json:"max_iterations"`
}

type WebSearchConfig struct {
	Enabled         bool                     `json:"enabled"`
	DefaultProvider string                   `json:"default_provider"`
	TimeoutSeconds  int                      `json:"timeout_seconds"`
	MaxResults      int                      `json:"max_results"`
	Fallback        bool                     `json:"fallback"`
	CacheTTLMinutes int                      `json:"cache_ttl_minutes"`
	CacheMaxEntries int                      `json:"cache_max_entries"`
	Providers       WebSearchProvidersConfig `json:"providers"`
}

type WebSearchProvidersConfig struct {
	Brave      WebSearchAPIProviderConfig `json:"brave"`
	Tavily     WebSearchAPIProviderConfig `json:"tavily"`
	Google     WebSearchGoogleConfig      `json:"google"`
	SearxNG    WebSearchSearxNGConfig     `json:"searxng"`
	DuckDuckGo WebSearchProviderConfig    `json:"duckduckgo"`
}

type WebSearchProviderConfig struct {
	Enabled bool `json:"enabled"`
}

type WebSearchAPIProviderConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key"`
}

type WebSearchGoogleConfig struct {
	Enabled        bool   `json:"enabled"`
	APIKey         string `json:"api_key"`
	SearchEngineID string `json:"search_engine_id"`
}

type WebSearchSearxNGConfig struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url"`
}

type MemoryConfig struct {
	Directory         string `json:"directory"`
	PrimaryDir        string `json:"primary_dir"`
	RefsDir           string `json:"refs_dir"`
	CacheDir          string `json:"cache_dir"`
	MaxFileSizeKB     int    `json:"max_file_size_kb"`
	RetentionDays     int    `json:"retention_days"`
	AutoCleanup       bool   `json:"auto_cleanup"`
	MaxReferenceDepth int    `json:"max_reference_depth"`
	LazyLoading       bool   `json:"lazy_loading"`
	CacheSizeLimitMB  int    `json:"cache_size_limit_mb"`
}

type LoggingConfig struct {
	Enabled         bool   `json:"enabled"`
	DBPath          string `json:"db_path"`
	RetentionDays   int    `json:"retention_days"`
	LogFullPayload  bool   `json:"log_full_payload"`
	MaxPayloadBytes int    `json:"max_payload_bytes"`
}

type ChatConfig struct {
	SystemPrompt       string  `json:"system_prompt"`
	MaxHistoryMessages int     `json:"max_history_messages"`
	Stream             bool    `json:"stream"`
	Temperature        float64 `json:"temperature"`
}

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
	MaxHistoryMessages int     `json:"max_history_messages"`
	Stream             bool    `json:"stream"`
	Temperature        float64 `json:"temperature"`
}

func Load() (*Config, error) {
	path := os.Getenv("BS_AI_CONFIG_PATH")
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(wd, defaultConfigFile)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{
				Enabled:   false,
				Path:      path,
				Providers: map[string]ProviderConfig{},
			}, nil
		}
		return nil, fmt.Errorf("read AI config: %w", err)
	}

	cfg := &Config{Enabled: true, Path: path}
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse AI config: %w", err)
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(content, &raw)
	applyDefaults(cfg, raw)
	if err := resolveSecrets(cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config, raw map[string]json.RawMessage) {
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	if !nestedPresent(raw, "tools", "max_iterations") {
		cfg.Tools.MaxIterations = 5
	}
	if cfg.WebSearch.DefaultProvider == "" {
		cfg.WebSearch.DefaultProvider = "auto"
	}
	if !nestedPresent(raw, "web_search", "timeout_seconds") {
		cfg.WebSearch.TimeoutSeconds = 30
	}
	if !nestedPresent(raw, "web_search", "max_results") {
		cfg.WebSearch.MaxResults = 10
	}
	if !nestedPresent(raw, "web_search", "fallback") {
		cfg.WebSearch.Fallback = true
	}
	if !nestedPresent(raw, "web_search", "cache_ttl_minutes") {
		cfg.WebSearch.CacheTTLMinutes = 5
	}
	if !nestedPresent(raw, "web_search", "cache_max_entries") {
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
	if !nestedPresent(raw, "memory", "max_file_size_kb") {
		cfg.Memory.MaxFileSizeKB = 1024
	}
	if !nestedPresent(raw, "memory", "retention_days") {
		cfg.Memory.RetentionDays = 365
	}
	if !nestedPresent(raw, "memory", "max_reference_depth") {
		cfg.Memory.MaxReferenceDepth = 5
	}
	if !nestedPresent(raw, "memory", "cache_size_limit_mb") {
		cfg.Memory.CacheSizeLimitMB = 100
	}
	if cfg.Logging.DBPath == "" {
		cfg.Logging.DBPath = ".data/bs-ai.db"
	}
	if !nestedPresent(raw, "logging", "retention_days") {
		cfg.Logging.RetentionDays = 60
	}
	if !nestedPresent(raw, "logging", "max_payload_bytes") {
		cfg.Logging.MaxPayloadBytes = 1048576
	}
	if cfg.Chat.SystemPrompt == "" {
		cfg.Chat.SystemPrompt = "You are a helpful assistant integrated into browser-server."
	}
	if !nestedPresent(raw, "chat", "max_history_messages") {
		cfg.Chat.MaxHistoryMessages = 30
	}
	if !nestedPresent(raw, "chat", "temperature") {
		cfg.Chat.Temperature = 0.7
	}
	if !nestedPresent(raw, "chat", "stream") {
		cfg.Chat.Stream = true
	}
	for name, provider := range cfg.Providers {
		if !providerFieldPresent(raw, name, "request_timeout_seconds") {
			provider.RequestTimeoutSeconds = 120
		}
		if !providerFieldPresent(raw, name, "retry_attempts") {
			provider.RetryAttempts = 10
		}
		if !providerFieldPresent(raw, name, "retry_delay_seconds") {
			provider.RetryDelaySeconds = 5
		}
		cfg.Providers[name] = provider
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

func resolveSecrets(cfg *Config) error {
	for name, provider := range cfg.Providers {
		if strings.HasPrefix(provider.APIKey, "env:") {
			envName := strings.TrimSpace(strings.TrimPrefix(provider.APIKey, "env:"))
			if envName == "" {
				return fmt.Errorf("provider %q api_key env reference is empty", name)
			}
			value := os.Getenv(envName)
			if value == "" {
				return fmt.Errorf("provider %q api_key references unset environment variable %q", name, envName)
			}
			provider.APIKey = value
			cfg.Providers[name] = provider
		}
	}
	var err error
	if cfg.WebSearch.Providers.Brave.Enabled {
		cfg.WebSearch.Providers.Brave.APIKey, err = resolveOptionalEnv("web_search provider \"brave\" api_key", cfg.WebSearch.Providers.Brave.APIKey)
		if err != nil {
			return err
		}
	}
	if cfg.WebSearch.Providers.Tavily.Enabled {
		cfg.WebSearch.Providers.Tavily.APIKey, err = resolveOptionalEnv("web_search provider \"tavily\" api_key", cfg.WebSearch.Providers.Tavily.APIKey)
		if err != nil {
			return err
		}
	}
	if cfg.WebSearch.Providers.Google.Enabled {
		cfg.WebSearch.Providers.Google.APIKey, err = resolveOptionalEnv("web_search provider \"google\" api_key", cfg.WebSearch.Providers.Google.APIKey)
		if err != nil {
			return err
		}
		cfg.WebSearch.Providers.Google.SearchEngineID, err = resolveOptionalEnv("web_search provider \"google\" search_engine_id", cfg.WebSearch.Providers.Google.SearchEngineID)
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveOptionalEnv(field, value string) (string, error) {
	if !strings.HasPrefix(value, "env:") {
		return value, nil
	}
	envName := strings.TrimSpace(strings.TrimPrefix(value, "env:"))
	if envName == "" {
		return "", fmt.Errorf("%s env reference is empty", field)
	}
	resolved := os.Getenv(envName)
	if resolved == "" {
		return "", fmt.Errorf("%s references unset environment variable %q", field, envName)
	}
	return resolved, nil
}

func validate(cfg *Config) error {
	if cfg.DefaultProvider == "" {
		return fmt.Errorf("default_provider is required")
	}
	if _, ok := cfg.Providers[cfg.DefaultProvider]; !ok {
		return fmt.Errorf("default_provider %q is not configured", cfg.DefaultProvider)
	}
	for name, provider := range cfg.Providers {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("provider name cannot be empty")
		}
		if provider.Type != "openai_compatible" {
			return fmt.Errorf("provider %q has unsupported type %q", name, provider.Type)
		}
		parsed, err := url.Parse(provider.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("provider %q base_url is invalid", name)
		}
		if parsed.Scheme != "https" && !isLocalHost(parsed.Hostname()) {
			return fmt.Errorf("provider %q base_url must use https unless it is local", name)
		}
		if strings.TrimSpace(provider.APIKey) == "" {
			return fmt.Errorf("provider %q api_key is required", name)
		}
		if provider.RequestTimeoutSeconds <= 0 || provider.RequestTimeoutSeconds > int((10*time.Minute).Seconds()) {
			return fmt.Errorf("provider %q request_timeout_seconds must be between 1 and 600", name)
		}
		if provider.RetryAttempts < 0 || provider.RetryAttempts > 20 {
			return fmt.Errorf("provider %q retry_attempts must be between 0 and 20", name)
		}
		if provider.RetryDelaySeconds < 1 || provider.RetryDelaySeconds > 300 {
			return fmt.Errorf("provider %q retry_delay_seconds must be between 1 and 300", name)
		}
		if len(provider.Models) == 0 {
			return fmt.Errorf("provider %q must configure at least one model", name)
		}
		defaults := 0
		modelIDs := map[string]bool{}
		for _, model := range provider.Models {
			if strings.TrimSpace(model.ID) == "" {
				return fmt.Errorf("provider %q model id cannot be empty", name)
			}
			if modelIDs[model.ID] {
				return fmt.Errorf("provider %q has duplicate model %q", name, model.ID)
			}
			modelIDs[model.ID] = true
			if model.Default {
				defaults++
			}
			if model.MaxOutputTokens <= 0 {
				return fmt.Errorf("provider %q model %q max_output_tokens must be positive", name, model.ID)
			}
		}
		if defaults != 1 {
			return fmt.Errorf("provider %q must have exactly one default model", name)
		}
	}
	if cfg.Tools.MaxIterations <= 0 || cfg.Tools.MaxIterations > 500 {
		return fmt.Errorf("tools.max_iterations must be between 1 and 500")
	}
	known := map[string]bool{
		"get_current_time": true, "search_bookmarks": true, "execute_command": true,
		"web_search": true, "web_fetch": true,
		"read_file": true, "write_file": true, "edit_file": true, "list_directory": true,
		"delete_file": true, "move_file": true, "copy_file": true,
		"directory_tree": true,
		"search_code":    true, "analyze_code": true, "get_diagnostics": true,
		"git_status": true, "git_diff": true, "git_log": true,
		"git_branch": true, "git_checkout": true, "git_commit": true,
		"git_push": true, "git_pull": true, "git_merge": true,
		"ai_remember": true, "ai_recall": true, "ai_search_memory": true,
		"ai_list_memories": true, "ai_forget": true, "ai_update_memory": true,
		"ai_resolve_references": true, "ai_lazy_memory": true, "ai_manage_cache": true,
		"list_skills": true, "activate_skill": true, "deactivate_skill": true, "get_active_skills": true,
	}
	for _, name := range cfg.Tools.Allowed {
		if !known[name] {
			return fmt.Errorf("tools.allowed contains unknown tool %q", name)
		}
	}
	if err := validateWebSearch(cfg.WebSearch); err != nil {
		return err
	}
	if filepath.IsAbs(cfg.Memory.Directory) || strings.Contains(cfg.Memory.Directory, "..") {
		return fmt.Errorf("memory.directory must be a safe relative path")
	}
	if filepath.IsAbs(cfg.Skills.Directory) || strings.Contains(cfg.Skills.Directory, "..") {
		return fmt.Errorf("skills.directory must be a safe relative path")
	}
	for _, dir := range []string{cfg.Memory.PrimaryDir, cfg.Memory.RefsDir, cfg.Memory.CacheDir} {
		if dir == "" || filepath.IsAbs(dir) || strings.Contains(dir, "..") || filepath.Base(dir) != dir {
			return fmt.Errorf("memory subdirectories must be safe names")
		}
	}
	if cfg.Memory.MaxFileSizeKB < 1 || cfg.Memory.MaxFileSizeKB > 10240 {
		return fmt.Errorf("memory.max_file_size_kb must be between 1 and 10240")
	}
	if cfg.Memory.MaxReferenceDepth < 1 || cfg.Memory.MaxReferenceDepth > 20 {
		return fmt.Errorf("memory.max_reference_depth must be between 1 and 20")
	}
	if cfg.Memory.RetentionDays < 1 || cfg.Memory.RetentionDays > 3650 {
		return fmt.Errorf("memory.retention_days must be between 1 and 3650")
	}
	if cfg.Memory.CacheSizeLimitMB < 1 || cfg.Memory.CacheSizeLimitMB > 10240 {
		return fmt.Errorf("memory.cache_size_limit_mb must be between 1 and 10240")
	}
	if cfg.Logging.RetentionDays < 1 || cfg.Logging.RetentionDays > 3650 {
		return fmt.Errorf("logging.retention_days must be between 1 and 3650")
	}
	if cfg.Logging.MaxPayloadBytes < 1024 || cfg.Logging.MaxPayloadBytes > 10*1024*1024 {
		return fmt.Errorf("logging.max_payload_bytes must be between 1024 and 10485760")
	}
	if cfg.Chat.MaxHistoryMessages < 1 || cfg.Chat.MaxHistoryMessages > 200 {
		return fmt.Errorf("chat.max_history_messages must be between 1 and 200")
	}
	if cfg.Chat.Temperature < 0 || cfg.Chat.Temperature > 2 {
		return fmt.Errorf("chat.temperature must be between 0 and 2")
	}
	parent := filepath.Dir(cfg.ResolvePath(cfg.Logging.DBPath))
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("logging database parent: %w", err)
	}
	probe, err := os.CreateTemp(parent, ".ai-write-test-")
	if err != nil {
		return fmt.Errorf("logging database parent is not writable: %w", err)
	}
	probeName := probe.Name()
	probe.Close()
	os.Remove(probeName)
	return nil
}

func validateWebSearch(cfg WebSearchConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.TimeoutSeconds < 1 || cfg.TimeoutSeconds > 120 {
		return fmt.Errorf("web_search.timeout_seconds must be between 1 and 120")
	}
	if cfg.MaxResults < 1 || cfg.MaxResults > 20 {
		return fmt.Errorf("web_search.max_results must be between 1 and 20")
	}
	if cfg.CacheTTLMinutes < 1 || cfg.CacheTTLMinutes > 1440 {
		return fmt.Errorf("web_search.cache_ttl_minutes must be between 1 and 1440")
	}
	if cfg.CacheMaxEntries < 1 || cfg.CacheMaxEntries > 10000 {
		return fmt.Errorf("web_search.cache_max_entries must be between 1 and 10000")
	}
	available := map[string]bool{
		"brave":      cfg.Providers.Brave.Enabled,
		"tavily":     cfg.Providers.Tavily.Enabled,
		"google":     cfg.Providers.Google.Enabled,
		"searxng":    cfg.Providers.SearxNG.Enabled,
		"duckduckgo": cfg.Providers.DuckDuckGo.Enabled,
	}
	if cfg.DefaultProvider != "auto" && !available[cfg.DefaultProvider] {
		return fmt.Errorf("web_search.default_provider %q is not enabled", cfg.DefaultProvider)
	}
	if cfg.Providers.Brave.Enabled && strings.TrimSpace(cfg.Providers.Brave.APIKey) == "" {
		return fmt.Errorf("web_search provider \"brave\" api_key is required")
	}
	if cfg.Providers.Tavily.Enabled && strings.TrimSpace(cfg.Providers.Tavily.APIKey) == "" {
		return fmt.Errorf("web_search provider \"tavily\" api_key is required")
	}
	if cfg.Providers.Google.Enabled && (strings.TrimSpace(cfg.Providers.Google.APIKey) == "" || strings.TrimSpace(cfg.Providers.Google.SearchEngineID) == "") {
		return fmt.Errorf("web_search provider \"google\" api_key and search_engine_id are required")
	}
	if cfg.Providers.SearxNG.Enabled {
		u, err := url.Parse(cfg.Providers.SearxNG.BaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "https" && !isLocalHost(u.Hostname())) {
			return fmt.Errorf("web_search provider \"searxng\" base_url must be a valid HTTPS or local URL")
		}
	}
	for _, enabled := range available {
		if enabled {
			return nil
		}
	}
	return fmt.Errorf("web_search must enable at least one provider")
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
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

func (cfg *Config) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(cfg.Path), path)
}

package config

// NOTE: Must be relative to go compiled binary for portable app
const defaultConfigFile = "bs-ai-config.json"

type Config struct {
	Enabled         bool                      `json:"-"`
	Path            string                    `json:"-"`
	CORSEnabled     bool                      `json:"cors_enabled"`
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
	Tools           ToolsConfig               `json:"tools"`
	FileTools       FileToolsConfig           `json:"file_tools"`
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

type FileToolsConfig struct {
	MaxReadBytes        int      `json:"max_read_bytes"`
	MaxLineReadBytes    int      `json:"max_line_read_bytes"`
	MaxLineCount        int      `json:"max_line_count"`
	MaxFileSizeWarnMB   int      `json:"max_file_size_warn_mb"`
	AllowedExtensions   []string `json:"allowed_extensions"`
	BlockedPathPatterns []string `json:"blocked_path_patterns"`
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
	SystemPrompt          string  `json:"system_prompt"`
	MaxHistoryMessages    int     `json:"max_history_messages"`
	Stream                bool    `json:"stream"`
	Temperature           float64 `json:"temperature"`
	ToolRetryAttempts     int     `json:"tool_retry_attempts"`
	ToolRetryDelaySeconds int     `json:"tool_retry_delay_seconds"`
}

// maxRequestTimeoutSeconds mirrors the upper bound enforced in validate()
// (10 minutes).
const maxRequestTimeoutSeconds = 600

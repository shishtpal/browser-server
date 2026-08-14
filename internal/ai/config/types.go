package config

import "time"

// NOTE: Must be relative to go compiled binary for portable app
const defaultConfigFile = "bs-ai-config.json"
const defaultModelsFile = "bs-ai-models.json"

type Config struct {
	Enabled         bool                      `json:"enabled"`
	Path            string                    `json:"-"`
	ModelsPath      string                    `json:"-"`
	CORSEnabled     bool                      `json:"cors_enabled"`
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"-"`
	Tools           ToolsConfig               `json:"tools"`
	FileTools       FileToolsConfig           `json:"file_tools"`
	WebSearch       WebSearchConfig           `json:"web_search"`
	Memory          MemoryConfig              `json:"memory"`
	Skills          SkillsConfig              `json:"skills"`
	Logging         LoggingConfig             `json:"logging"`
	Chat            ChatConfig                `json:"chat"`
	Tasks           TasksConfig               `json:"tasks"`
	Paths           PathsConfig               `json:"paths"`
	OCR             OCRConfig                 `json:"ocr"`
}

// TasksConfig controls the durable background task runner. The runner survives
// crashes by leasing tasks and checkpointing progress, so the timing fields
// below are load-bearing rather than cosmetic: see the invariants enforced in
// validateTasks.
//
// It is declared here rather than in internal/ai/tasks because that package
// imports chat, which imports this one — the timing knobs have to live on the
// config side of that edge.
type TasksConfig struct {
	// Enabled gates the whole subsystem. When false no workers or watchdog run.
	Enabled bool `json:"enabled"`
	// MaxConcurrent is the number of workers polling for tasks.
	MaxConcurrent int `json:"max_concurrent"`
	// LeaseSeconds is how long a claim stays valid without a heartbeat.
	LeaseSeconds int `json:"lease_seconds"`
	// HeartbeatSeconds is how often a worker renews its lease.
	HeartbeatSeconds int `json:"heartbeat_seconds"`
	// StepTimeoutSeconds bounds one agent step (one model call plus its tools).
	StepTimeoutSeconds int `json:"step_timeout_seconds"`
	// NoProgressSeconds is how long a task may hold a healthy lease without
	// producing a checkpoint before it is treated as wedged.
	NoProgressSeconds int `json:"no_progress_seconds"`
	// WatchdogSeconds is how often expired leases are swept.
	WatchdogSeconds int `json:"watchdog_seconds"`
	// IdleDelaySeconds is how long a worker waits after finding no work.
	IdleDelaySeconds int `json:"idle_delay_seconds"`
	// MaxSteps is the hard ceiling on steps per task, independent of whatever
	// the model claims about its own progress.
	MaxSteps int `json:"max_steps"`
	// MaxAttempts is the default retry budget for newly enqueued tasks.
	MaxAttempts int `json:"max_attempts"`
	// RetentionDays is how long terminal tasks are kept before cleanup.
	RetentionDays int `json:"retention_days"`
	// Provider and Model override the default selection for task work.
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// ToolsEnabled lets task agents call tools. Task execution is unattended,
	// so those calls always run without approval — there is no operator to ask.
	ToolsEnabled bool `json:"tools_enabled"`
}

// PathsConfig lets the operator configure additional PATH directories and
// explicit binary overrides so AI tools can find executables that are not on
// the server process's inherited PATH.
//
// Binaries behave differently for the two kinds of tools:
//   - Tools that launch a known binary directly (get_diagnostics, git tools,
//     execute_python) use the exact mapped path, bypassing PATH.
//   - Child shell processes (execute_command) receive the mapped binaries'
//     parent directories in PATH, so a command that calls the binary by its
//     basename (e.g. `go version`) resolves to the configured file.
type PathsConfig struct {
	// AdditionalDirs are prepended to the PATH of child processes. They are
	// also searched (in order) before the system PATH when resolving a binary.
	AdditionalDirs []string `json:"additional_dirs"`
	// Binaries maps a binary name to an explicit full path. Direct known-binary
	// tools use the exact path; shell commands see the mapped binary's parent
	// directory prepended to their PATH, so the key should match the executable
	// basename (without a platform extension) for shell name resolution.
	Binaries map[string]string `json:"binaries"`
}

type SkillsConfig struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
}

type ProviderConfig struct {
	Type                  string        `json:"type"`
	BaseURL               string        `json:"base_url"`
	URL                   string        `json:"url,omitempty"`
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
	SupportsVision  bool   `json:"supports_vision"`
	Default         bool   `json:"default"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

type ToolsConfig struct {
	Enabled        bool     `json:"enabled"`
	Allowed        []string `json:"allowed"`
	RawOutput      []string `json:"raw_output"`
	MaxIterations  int      `json:"max_iterations"`
	MaxOutput      int      `json:"max_output"`
	MaxDiffOutput  int      `json:"max_diff_output"`
	GitTimeoutSecs int      `json:"git_timeout_seconds"`
}

// MaxOutputBytes returns the configured max output limit clamped to a sane
// minimum, so callers can safely use it without checking zero.
func (t ToolsConfig) MaxOutputBytes() int {
	if t.MaxOutput <= 0 {
		return 32 * 1024
	}
	return t.MaxOutput
}

// MaxDiffOutputBytes returns the configured diff output limit. It falls back to
// the general max output when a diff-specific value is not configured.
func (t ToolsConfig) MaxDiffOutputBytes() int {
	if t.MaxDiffOutput <= 0 {
		return t.MaxOutputBytes()
	}
	return t.MaxDiffOutput
}

// GitTimeout returns the configured git timeout clamped to a positive value.
func (t ToolsConfig) GitTimeout() time.Duration {
	if t.GitTimeoutSecs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(t.GitTimeoutSecs) * time.Second
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

// MemoryConfig configures the v2 graph fragment memory system
// (internal/ai/memory). It supersedes the v1 flat-file store: the only unit is
// a fragment, stored under fragments_dir, connected by typed edges into a tree
// rooted at mem_root plus an associative web. Every field below is read by the
// memory package (or validated in this file); the reflection test
// TestAllMemoryFieldsUsed guards against dead configuration.
type MemoryConfig struct {
	// Enabled gates the whole subsystem. When false no fragments are created,
	// no persona block is injected, and the memory tools return "disabled".
	Enabled bool `json:"enabled"`

	// Directory is the memory root, resolved relative to the executable dir
	// (or absolute). Must be a safe relative path.
	Directory    string `json:"directory"`
	FragmentsDir string `json:"fragments_dir"`
	ArchiveDir   string `json:"archive_dir"`

	// MaxBodyKB bounds a fragment body; MaxLinksPerFragment bounds its edges;
	// MaxOpsPerCall bounds a write_memory batch; MaxResultBytes caps a recall
	// payload before the JSON envelope pushes it over the tool output limit.
	MaxBodyKB           int `json:"max_body_kb"`
	MaxLinksPerFragment int `json:"max_links_per_fragment"`
	MaxOpsPerCall       int `json:"max_ops_per_call"`
	MaxResultBytes      int `json:"max_result_bytes"`

	// DefaultDepth is used when recall_memory omits depth; MaxDepth clamps the
	// tool's depth argument and the spreading-activation hops.
	DefaultDepth int     `json:"default_depth"`
	MaxDepth     int     `json:"max_depth"`
	SpreadFactor float64 `json:"spread_factor"`

	// PersonaTokenBudget bounds the auto-injected <memory:persona> block.
	// InjectPersona / InjectUsageGuide toggle that block and the usage guide.
	PersonaTokenBudget int  `json:"persona_token_budget"`
	InjectPersona      bool `json:"inject_persona"`
	InjectUsageGuide   bool `json:"inject_usage_guide"`

	// Maintenance: RetentionDays is how long soft-deleted fragments stay in
	// .archive before purge; AutoCleanup gates the periodic Maintain job;
	// MaintenanceInterval is its period; SalienceDecayPerWeek and
	// ArchiveThreshold drive salience decay and archival.
	RetentionDays        int     `json:"retention_days"`
	AutoCleanup          bool    `json:"auto_cleanup"`
	MaintenanceInterval  string  `json:"maintenance_interval"`
	SalienceDecayPerWeek float64 `json:"salience_decay_per_week"`
	ArchiveThreshold     float64 `json:"archive_threshold"`

	// SecretScan rejects writes whose body contains credentials; writes with
	// a similarity >= NearDuplicateThreshold to an existing fragment are
	// merged (as duplicates) instead of creating a near-twin.
	SecretScan             bool    `json:"secret_scan"`
	NearDuplicateThreshold float64 `json:"near_duplicate_threshold"`

	// Synthesizer is the optional "librarian" sub-agent: a cheap model that
	// reads matched fragments and returns a concise answer. Provider and Model
	// are names resolved from bs-ai-models.json (the file that actually holds
	// provider base URLs, API keys, etc.). Set Synthesizer.Enabled false to
	// return raw graph JSON instead.
	Synthesizer SynthesizerConfig `json:"synthesizer"`

	// Embeddings is an optional, drop-in semantic layer. When disabled the
	// system is pure lexical BM25-lite; nothing else changes.
	Embeddings EmbeddingsConfig `json:"embeddings"`
}

// SynthesizerConfig configures the recall_memory synthesize=true mode.
type SynthesizerConfig struct {
	Enabled bool `json:"enabled"`
	// Provider and Model reference an entry in bs-ai-models.json. The actual
	// base_url/api_key live there, so the operator can reuse an existing
	// provider (e.g. "openrouter") with a cheap model.
	Provider        string  `json:"provider"`
	Model           string  `json:"model"`
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"max_output_tokens"`
	TimeoutMS       int     `json:"timeout_ms"`
	// FallbackOnError returns the raw graph JSON if the cheap model call fails.
	FallbackOnError bool `json:"fallback_on_error"`
}

// EmbeddingsConfig configures the optional semantic retrieval layer.
type EmbeddingsConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Dims     int    `json:"dims"`
}

type LoggingConfig struct {
	Enabled         bool   `json:"enabled"`
	DBPath          string `json:"db_path"`
	RetentionDays   int    `json:"retention_days"`
	LogFullPayload  bool   `json:"log_full_payload"`
	MaxPayloadBytes int    `json:"max_payload_bytes"`
}

type ChatConfig struct {
	SystemPrompt          string                `json:"system_prompt"`
	MaxHistoryMessages    int                   `json:"max_history_messages"`
	Stream                bool                  `json:"stream"`
	ShowThinking          bool                  `json:"show_thinking"`
	Temperature           float64               `json:"temperature"`
	ToolRetryAttempts     int                   `json:"tool_retry_attempts"`
	ToolRetryDelaySeconds int                   `json:"tool_retry_delay_seconds"`
	Attachments           ChatAttachmentsConfig `json:"attachments"`
}

// ChatAttachmentsConfig bounds image attachments on user chat messages. The
// operator can tighten any limit in bs-ai-config.json; zero values are filled
// with safe defaults during Load.
type ChatAttachmentsConfig struct {
	// Enabled gates the whole feature: upload endpoints and the vision
	// enforcement on submit are disabled when false.
	Enabled bool `json:"enabled"`
	// AllowedMIMETypes is the set of image MIME types accepted for upload.
	AllowedMIMETypes []string `json:"allowed_mime_types"`
	// MaxImages is the maximum number of attachments per user message.
	MaxImages int `json:"max_images"`
	// MaxImageBytes is the maximum size of a single image file.
	MaxImageBytes int `json:"max_image_bytes"`
	// MaxTotalBytes is the aggregate byte limit for all images in one message.
	MaxTotalBytes int `json:"max_total_bytes"`
	// RetentionHours is how long staged (unattached) uploads are kept before
	// the cleanup job removes them and reclaims disk space.
	RetentionHours int `json:"retention_hours"`
}

// maxRequestTimeoutSeconds mirrors the upper bound enforced in validate()
// (10 minutes).
const maxRequestTimeoutSeconds = 600

package config

import "encoding/json"

// ExploreProjectConfig is loaded from the "explore_project" section of
// bs-ai-config.json. It gates the built-in explore_project tool: an agentic
// codebase-exploration tool that runs an internal read-only "explorer" LLM
// (defaulting to the .skills/explorer.md prompt) which iteratively calls
// read-only tools against a project, then returns the synthesized answer.
type ExploreProjectConfig struct {
	// Enabled gates the whole tool. When false the explore_project tool is not
	// registered and cannot be invoked.
	Enabled bool `json:"enabled"`
	// DefaultProvider is the provider used when the caller omits provider.
	// Falls back to the top-level default_provider when omitted.
	DefaultProvider string `json:"default_provider"`
	// DefaultModel is the model used when the caller omits model. Resolved from
	// the configured provider's default model when omitted. The model MUST
	// support tool calling (supports_tools: true) because the explorer agent
	// drives read-only tools via function calls.
	DefaultModel string `json:"default_model"`
	// Temperature for the explorer LLM. Low values keep answers factual.
	Temperature float64 `json:"temperature"`
	// MaxOutputTokens bounds the explorer's final synthesized answer.
	MaxOutputTokens int `json:"max_output_tokens"`
	// TimeoutMS bounds a single explorer LLM call.
	TimeoutMS int `json:"timeout_ms"`
	// MaxIterations caps the explorer agent's tool-calling loop.
	MaxIterations int `json:"max_iterations"`
	// ExcludedDirs are injected into search_code exclude globs so the explorer
	// skips generated/dependency/vendor noise by default.
	ExcludedDirs []string `json:"excluded_dirs"`
	// SkillName is the .skills/*.md skill whose body is used as the default
	// system prompt for the explorer LLM. Defaults to "explorer".
	SkillName string `json:"skill_name"`
}

// applyExploreProjectDefaults fills the explore_project section with safe
// defaults when the operator omitted the whole object or individual fields.
func applyExploreProjectDefaults(cfg *Config, mainRaw map[string]json.RawMessage) {
	if !nestedPresent(mainRaw, "explore_project", "default_provider") {
		cfg.ExploreProject.DefaultProvider = cfg.DefaultProvider
	}
	if !nestedPresent(mainRaw, "explore_project", "default_model") {
		if m, ok := cfg.DefaultModel(cfg.ExploreProject.DefaultProvider); ok {
			cfg.ExploreProject.DefaultModel = m.ID
		}
	}
	if !nestedPresent(mainRaw, "explore_project", "temperature") {
		cfg.ExploreProject.Temperature = 0.2
	}
	if !nestedPresent(mainRaw, "explore_project", "max_output_tokens") {
		cfg.ExploreProject.MaxOutputTokens = 4096
	}
	if !nestedPresent(mainRaw, "explore_project", "timeout_ms") {
		cfg.ExploreProject.TimeoutMS = 120000
	}
	if !nestedPresent(mainRaw, "explore_project", "max_iterations") {
		cfg.ExploreProject.MaxIterations = 20
	}
	if cfg.ExploreProject.ExcludedDirs == nil {
		cfg.ExploreProject.ExcludedDirs = []string{
			"node_modules", "dist", "bin", ".git", ".data", ".memory",
			"target", "vendor", "build",
		}
	}
	if cfg.ExploreProject.SkillName == "" {
		cfg.ExploreProject.SkillName = "explorer"
	}
}

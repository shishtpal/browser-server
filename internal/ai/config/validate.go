package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// knownToolNames is the canonical list of tools that may appear in
// tools.allowed. Keep this in one place so adding a new tool does not require
// touching multiple call sites.
var knownToolNames = map[string]bool{
	"search_tool": true, "get_current_time": true, "ask_questions": true,
	"search_bookmarks": true, "search_todos": true,
	"add_todo_record":    true,
	"update_todo_record": true,
	"search_prompts":     true,
	"manage_prompt":      true,
	"search_questions":   true,
	"manage_question":    true,
	"search_history":     true, "search_calendar": true, "manage_calendar": true,
	"execute_command": true,
	"execute_python":  true,
	"web_search":      true, "web_fetch": true,
	"read_file": true, "read_files": true, "write_file": true, "edit_file": true, "multi_edit": true, "list_directory": true,
	"delete_file": true, "move_file": true, "copy_file": true,
	"directory_tree": true,
	"search_code":    true, "analyze_code": true, "get_diagnostics": true,
	"git_status": true, "git_diff": true, "git_log": true,
	"git_branch": true, "git_checkout": true, "git_commit": true,
	"git_push": true, "git_pull": true, "git_merge": true,
	"recall_memory": true, "write_memory": true,
	"list_skills": true, "activate_skill": true, "deactivate_skill": true, "get_active_skills": true,
	"generate_image":  true,
	"generate_video":  true,
	"text_to_speech":  true,
	"speech_to_text":  true,
	"ocr_image":       true,
	"explore_project": true,
}

// supportedAttachmentTypes is the closed set of image MIME types the feature
// accepts. It mirrors the shared AIImageAttachment content_type union and is
// enforced both here (config validation) and at upload time (file signatures).
var supportedAttachmentTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

func validateAttachments(cfg ChatAttachmentsConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if len(cfg.AllowedMIMETypes) == 0 {
		return fmt.Errorf("chat.attachments.allowed_mime_types must not be empty")
	}
	for _, mime := range cfg.AllowedMIMETypes {
		if !supportedAttachmentTypes[mime] {
			return fmt.Errorf("chat.attachments.allowed_mime_types contains unsupported type %q", mime)
		}
	}
	if cfg.MaxImages < 1 || cfg.MaxImages > 20 {
		return fmt.Errorf("chat.attachments.max_images must be between 1 and 20")
	}
	if cfg.MaxImageBytes < 64*1024 || cfg.MaxImageBytes > 10*1024*1024 {
		return fmt.Errorf("chat.attachments.max_image_bytes must be between 65536 and 10485760")
	}
	if cfg.MaxTotalBytes < cfg.MaxImageBytes || cfg.MaxTotalBytes > 100*1024*1024 {
		return fmt.Errorf("chat.attachments.max_total_bytes must be at least max_image_bytes and at most 104857600")
	}
	if cfg.RetentionHours < 1 || cfg.RetentionHours > 720 {
		return fmt.Errorf("chat.attachments.retention_hours must be between 1 and 720")
	}
	return nil
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
		switch provider.Type {
		case "openai_compatible", "gemini_interactions":
		default:
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
		if raw := strings.TrimSpace(provider.URL); raw != "" {
			consoleURL, err := url.Parse(raw)
			if err != nil || consoleURL.Scheme == "" || consoleURL.Host == "" {
				return fmt.Errorf("provider %q url is invalid", name)
			}
			if consoleURL.Scheme != "https" && !isLocalHost(consoleURL.Hostname()) {
				return fmt.Errorf("provider %q url must use https unless it is local", name)
			}
		}
		if provider.RequestTimeoutSeconds <= 0 || provider.RequestTimeoutSeconds > maxRequestTimeoutSeconds {
			return fmt.Errorf("provider %q request_timeout_seconds must be between 1 and %d", name, maxRequestTimeoutSeconds)
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
	if cfg.Tools.MaxIterations < 0 || cfg.Tools.MaxIterations > 500 {
		return fmt.Errorf("tools.max_iterations must be between 1 and 500")
	}
	// max_output must leave room for the tools' JSON response envelope (the
	// tools package reserves resultHeadroom, 2048 bytes, per result) plus a
	// usable payload, so the minimum is 4 KiB rather than 1 KiB.
	if cfg.Tools.MaxOutput < 4*1024 || cfg.Tools.MaxOutput > 512*1024 {
		return fmt.Errorf("tools.max_output must be between 4096 and 524288")
	}
	if cfg.Tools.MaxDiffOutput != 0 && (cfg.Tools.MaxDiffOutput < 1*1024 || cfg.Tools.MaxDiffOutput > 512*1024) {
		return fmt.Errorf("tools.max_diff_output must be between 1024 and 524288 or 0 to use max_output")
	}
	if cfg.Tools.GitTimeoutSecs < 1 || cfg.Tools.GitTimeoutSecs > int(10*time.Minute/time.Second) {
		return fmt.Errorf("tools.git_timeout_seconds must be between 1 and %d", int(10*time.Minute/time.Second))
	}
	for _, name := range cfg.Tools.Allowed {
		if !knownToolNames[name] {
			return fmt.Errorf("tools.allowed contains unknown tool %q", name)
		}
	}
	for _, name := range cfg.Tools.RawOutput {
		if !knownToolNames[name] {
			return fmt.Errorf("tools.raw_output contains unknown tool %q", name)
		}
	}
	if err := validateWebSearch(cfg.WebSearch); err != nil {
		return err
	}
	if err := validateFileTools(cfg.FileTools); err != nil {
		return err
	}
	if err := validatePaths(cfg.Paths, cfg); err != nil {
		return err
	}
	if filepath.IsAbs(cfg.Memory.Directory) || strings.Contains(cfg.Memory.Directory, "..") {
		return fmt.Errorf("memory.directory must be a safe relative path")
	}
	if filepath.IsAbs(cfg.Skills.Directory) || strings.Contains(cfg.Skills.Directory, "..") {
		return fmt.Errorf("skills.directory must be a safe relative path")
	}
	for _, dir := range []string{cfg.Memory.FragmentsDir, cfg.Memory.ArchiveDir} {
		if dir == "" || filepath.IsAbs(dir) || strings.Contains(dir, "..") || filepath.Base(dir) != dir {
			return fmt.Errorf("memory subdirectories must be safe names")
		}
	}
	if cfg.Memory.MaxBodyKB < 1 || cfg.Memory.MaxBodyKB > 10240 {
		return fmt.Errorf("memory.max_body_kb must be between 1 and 10240")
	}
	if cfg.Memory.MaxLinksPerFragment < 1 || cfg.Memory.MaxLinksPerFragment > 1024 {
		return fmt.Errorf("memory.max_links_per_fragment must be between 1 and 1024")
	}
	if cfg.Memory.MaxOpsPerCall < 1 || cfg.Memory.MaxOpsPerCall > 100 {
		return fmt.Errorf("memory.max_ops_per_call must be between 1 and 100")
	}
	if cfg.Memory.MaxResultBytes < 512 || cfg.Memory.MaxResultBytes > 1<<20 {
		return fmt.Errorf("memory.max_result_bytes must be between 512 and 1048576")
	}
	if cfg.Memory.MaxDepth < 1 || cfg.Memory.MaxDepth > 10 {
		return fmt.Errorf("memory.max_depth must be between 1 and 10")
	}
	if cfg.Memory.DefaultDepth < 0 || cfg.Memory.DefaultDepth > cfg.Memory.MaxDepth {
		return fmt.Errorf("memory.default_depth must be between 0 and max_depth")
	}
	if cfg.Memory.SpreadFactor < 0 || cfg.Memory.SpreadFactor > 1 {
		return fmt.Errorf("memory.spread_factor must be between 0 and 1")
	}
	if cfg.Memory.PersonaTokenBudget < 100 || cfg.Memory.PersonaTokenBudget > 20000 {
		return fmt.Errorf("memory.persona_token_budget must be between 100 and 20000")
	}
	if cfg.Memory.RetentionDays < 1 || cfg.Memory.RetentionDays > 3650 {
		return fmt.Errorf("memory.retention_days must be between 1 and 3650")
	}
	if cfg.Memory.SalienceDecayPerWeek <= 0 || cfg.Memory.SalienceDecayPerWeek > 1 {
		return fmt.Errorf("memory.salience_decay_per_week must be between 0 and 1")
	}
	if cfg.Memory.ArchiveThreshold < 0 || cfg.Memory.ArchiveThreshold > 1 {
		return fmt.Errorf("memory.archive_threshold must be between 0 and 1")
	}
	if cfg.Memory.NearDuplicateThreshold <= 0 || cfg.Memory.NearDuplicateThreshold > 1 {
		return fmt.Errorf("memory.near_duplicate_threshold must be between 0 and 1")
	}
	if _, err := time.ParseDuration(cfg.Memory.MaintenanceInterval); err != nil || cfg.Memory.MaintenanceInterval == "" {
		return fmt.Errorf("memory.maintenance_interval must be a valid duration")
	}
	if cfg.Memory.Synthesizer.Enabled && (cfg.Memory.Synthesizer.Provider == "" || cfg.Memory.Synthesizer.Model == "") {
		return fmt.Errorf("memory.synthesizer.provider and memory.synthesizer.model are required when the synthesizer is enabled")
	}
	if cfg.Memory.Embeddings.Enabled && (cfg.Memory.Embeddings.Provider == "" || cfg.Memory.Embeddings.Model == "") {
		return fmt.Errorf("memory.embeddings.provider and memory.embeddings.model are required when embeddings are enabled")
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
	if cfg.Chat.ToolRetryAttempts < 1 || cfg.Chat.ToolRetryAttempts > 20 {
		return fmt.Errorf("chat.tool_retry_attempts must be between 1 and 20")
	}
	if cfg.Chat.ToolRetryDelaySeconds < 1 || cfg.Chat.ToolRetryDelaySeconds > 300 {
		return fmt.Errorf("chat.tool_retry_delay_seconds must be between 1 and 300")
	}
	if err := validateAttachments(cfg.Chat.Attachments); err != nil {
		return err
	}
	if err := validateTasks(cfg.Tasks); err != nil {
		return err
	}
	if err := validateOCR(cfg); err != nil {
		return err
	}
	if err := validateExploreProject(cfg); err != nil {
		return err
	}
	if err := validateOpenRouter(cfg); err != nil {
		return err
	}
	return nil
}

// validateOpenRouter enforces that the openrouter attribution section, when
// set, carries an absolute http(s) URL and header-safe values. The values
// travel verbatim as HTTP headers, so line breaks (which Go's HTTP writer
// rejects) and oversized strings are refused here at config time.
func validateOpenRouter(cfg *Config) error {
	if cfg.OpenRouter.SiteURL == "" && cfg.OpenRouter.AppName == "" {
		return nil
	}
	if len(cfg.OpenRouter.SiteURL) > 2048 || len(cfg.OpenRouter.AppName) > 128 {
		return fmt.Errorf("openrouter.site_url/app_name exceeds the length limit")
	}
	if strings.ContainsAny(cfg.OpenRouter.SiteURL, "\r\n") || strings.ContainsAny(cfg.OpenRouter.AppName, "\r\n") {
		return fmt.Errorf("openrouter.site_url/app_name must not contain line breaks")
	}
	if cfg.OpenRouter.SiteURL != "" {
		parsed, err := url.Parse(cfg.OpenRouter.SiteURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("openrouter.site_url must be an absolute http(s) URL")
		}
	}
	return nil
}

// validateStorage performs the startup-only writable-path probe. It is kept
// out of validate so administrator dry runs remain side-effect free.
func validateStorage(cfg *Config) error {
	parent := filepath.Dir(cfg.ResolvePath(cfg.Logging.DBPath))
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("logging database parent: %w", err)
	}
	probe, err := os.CreateTemp(parent, ".ai-write-test-")
	if err != nil {
		return fmt.Errorf("logging database parent is not writable: %w", err)
	}
	probeName := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(probeName)
		return fmt.Errorf("close logging database probe: %w", err)
	}
	if err := os.Remove(probeName); err != nil {
		return fmt.Errorf("remove logging database probe: %w", err)
	}
	return nil
}

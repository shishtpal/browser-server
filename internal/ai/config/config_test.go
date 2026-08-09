package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// withConfigPath sets BS_AI_CONFIG_PATH for the duration of a test, restoring
// the previous value on cleanup. Callers must run serially because the env
// variable is process-global. The returned path is the file the tests must
// populate (use writeConfig).
func withConfigPath(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "bs-ai-config.json")
	prev, had := os.LookupEnv("BS_AI_CONFIG_PATH")
	if err := os.Setenv("BS_AI_CONFIG_PATH", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("BS_AI_CONFIG_PATH", prev)
		} else {
			_ = os.Unsetenv("BS_AI_CONFIG_PATH")
		}
	})
	return path
}

// withModelsPath sets BS_AI_MODELS_PATH to the given path for the duration of a
// test, restoring the previous value on cleanup. The returned path is the file
// tests must populate.
func withModelsPath(t *testing.T, path string) string {
	t.Helper()
	prev, had := os.LookupEnv("BS_AI_MODELS_PATH")
	if err := os.Setenv("BS_AI_MODELS_PATH", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("BS_AI_MODELS_PATH", prev)
		} else {
			_ = os.Unsetenv("BS_AI_MODELS_PATH")
		}
	})
	return path
}

func writeConfig(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func writeModelsFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func minimalProviderConfig() string {
	return `{
		"default_provider": "openrouter"
	}`
}

func minimalProviderModels() string {
	return `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "sk-test",
				"models": [
					{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}
				]
			}
		}
	}`
}

func minimalProvider() string {
	return minimalProviderConfig() + "\n" + minimalProviderModels()
}

func setupBothConfigAndModels(t *testing.T, configBody, modelsBody string) (configPath, modelsPath string) {
	t.Helper()
	dir := t.TempDir()
	configPath = withConfigPath(t, dir)
	modelsPath = withModelsPath(t, filepath.Join(dir, "bs-ai-models.json"))
	writeConfig(t, configPath, configBody)
	writeModelsFile(t, modelsPath, modelsBody)
	return configPath, modelsPath
}

func TestLoadMissingConfigReturnsDisabled(t *testing.T) {
	path := withConfigPath(t, t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected disabled config, got enabled")
	}
	if cfg.Path != path {
		t.Fatalf("path = %q, want %q", cfg.Path, path)
	}
	if !cfg.CORSEnabled || cfg.Providers == nil {
		t.Fatalf("expected defaults: CORSEnabled=true, Providers non-nil")
	}
}

func TestLoadMissingModelsFileReturnsDisabled(t *testing.T) {
	path := withConfigPath(t, t.TempDir())
	writeConfig(t, path, minimalProviderConfig())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected disabled config when models file is missing")
	}
	if cfg.ModelsPath == "" {
		t.Fatalf("expected ModelsPath to be set")
	}
	if cfg.Providers != nil && len(cfg.Providers) > 0 {
		t.Fatalf("expected no providers loaded")
	}
}

func TestLoadPrefersModelsFile(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, minimalProviderModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("expected enabled config")
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(cfg.Providers))
	}
	if _, ok := cfg.Providers["openrouter"]; !ok {
		t.Fatalf("expected openrouter provider")
	}
}

func TestLoadModelsFileParseError(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, "{not json")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "parse AI models") {
		t.Fatalf("expected parse AI models error, got %v", err)
	}
}

func TestLoadRespectsBS_AI_MODELS_PATH(t *testing.T) {
	dir := t.TempDir()
	customModels := filepath.Join(dir, "custom-models.json")
	configPath := withConfigPath(t, dir)
	modelsPath := withModelsPath(t, customModels)
	writeConfig(t, configPath, `{"default_provider": "openrouter"}`)
	writeModelsFile(t, modelsPath, minimalProviderModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsPath != customModels {
		t.Fatalf("models path = %q, want %q", cfg.ModelsPath, customModels)
	}
	if !cfg.Enabled {
		t.Fatalf("expected enabled config")
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(cfg.Providers))
	}
}

func TestLoadIgnoresProvidersInMainConfig(t *testing.T) {
	path := withConfigPath(t, t.TempDir())
	writeConfig(t, path, `{
		"default_provider": "openrouter",
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "sk-test",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected disabled config when models file is missing and main config has providers")
	}
	if cfg.Providers != nil && len(cfg.Providers) > 0 {
		t.Fatalf("expected providers to be empty when sourced from missing models file")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, minimalProviderConfig(), minimalProviderModels())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("expected enabled config when file exists")
	}
	if cfg.Tools.MaxIterations != 5 {
		t.Fatalf("tools.max_iterations = %d, want 5", cfg.Tools.MaxIterations)
	}
	if cfg.Chat.Temperature != 0.7 || !cfg.Chat.Stream {
		t.Fatalf("chat defaults not applied: %+v", cfg.Chat)
	}
	if cfg.Chat.MaxHistoryMessages != 30 {
		t.Fatalf("chat.max_history_messages = %d, want 30", cfg.Chat.MaxHistoryMessages)
	}
	if cfg.Logging.DBPath != ".data/bs-ai.db" {
		t.Fatalf("logging.db_path = %q", cfg.Logging.DBPath)
	}
}

func TestDefaultsPreserveExplicitValues(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"tools": {"max_iterations": 10},
		"chat": {"temperature": 0.1, "stream": false, "max_history_messages": 5},
		"logging": {"retention_days": 1, "max_payload_bytes": 1024},
		"web_search": {"timeout_seconds": 1, "max_results": 2, "fallback": false, "cache_ttl_minutes": 1, "cache_max_entries": 1},
		"memory": {"max_body_kb": 64, "retention_days": 1, "max_depth": 3, "default_depth": 1},
		"file_tools": {"max_read_bytes": 4096, "max_line_read_bytes": 4096, "max_line_count": 100, "max_file_size_warn_mb": 1}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "sk-test",
				"request_timeout_seconds": 60,
				"retry_attempts": 0,
				"retry_delay_seconds": 1,
				"models": [
					{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 1}
				]
			}
		}
	}`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Chat.Temperature != 0.1 || cfg.Chat.Stream {
		t.Fatalf("explicit chat values lost: %+v", cfg.Chat)
	}
	if cfg.Providers["openrouter"].RequestTimeoutSeconds != 60 {
		t.Fatalf("explicit request_timeout_seconds lost: %d", cfg.Providers["openrouter"].RequestTimeoutSeconds)
	}
	if cfg.FileTools.MaxReadBytes != 4096 || cfg.FileTools.MaxLineCount != 100 {
		t.Fatalf("explicit file_tools values lost: %+v", cfg.FileTools)
	}
}

func TestResolveProviderSecret(t *testing.T) {
	t.Setenv("BS_AI_TEST_KEY", "secret-value")
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "env:BS_AI_TEST_KEY",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Providers["openrouter"].APIKey; got != "secret-value" {
		t.Fatalf("api_key = %q, want %q", got, "secret-value")
	}
}

func TestResolveProviderSecretMissingEnv(t *testing.T) {
	os.Unsetenv("BS_AI_TEST_KEY")
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "env:BS_AI_TEST_KEY",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "BS_AI_TEST_KEY") {
		t.Fatalf("expected unset-env error, got %v", err)
	}
}

func TestResolveEmptyEnvReference(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "env:",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "env reference is empty") {
		t.Fatalf("expected empty env reference error, got %v", err)
	}
}

func TestValidateRequiresDefaultProvider(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{}`, `{"providers": {}}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "default_provider is required") {
		t.Fatalf("expected default_provider required, got %v", err)
	}
}

func TestValidateRequiresKnownProviderType(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "unknown",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestValidateAllowsLocalHost(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "local"}`, `{
		"providers": {
			"local": {
				"type": "openai_compatible",
				"base_url": "http://localhost:11434/v1",
				"api_key": "k",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	if _, err := Load(); err != nil {
		t.Fatalf("local http host should validate, got %v", err)
	}
}

func TestValidateRejectsNonHttpsRemote(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "http://example.com/v1",
				"api_key": "k",
				"models": [{"id": "model-a", "label": "Model A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "must use https") {
		t.Fatalf("expected https requirement, got %v", err)
	}
}

func TestValidateRequiresExactlyOneDefaultModel(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{"default_provider": "openrouter"}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [
					{"id": "a", "label": "A", "supports_tools": true, "max_output_tokens": 4096},
					{"id": "b", "label": "B", "supports_tools": true, "max_output_tokens": 4096}
				]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "exactly one default model") {
		t.Fatalf("expected default model error, got %v", err)
	}
}

func TestValidateRejectsUnknownToolName(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"tools": {"allowed": ["does_not_exist"]}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "a", "label": "A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected unknown tool error, got %v", err)
	}
}

func TestValidateMemoryDirectoryMustBeSafe(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"memory": {"directory": "../escape"}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "a", "label": "A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "memory.directory") {
		t.Fatalf("expected safe path error, got %v", err)
	}
}

func TestValidateLoggingParentMustBeWritable(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"logging": {"db_path": "subdir/bs-ai.db"}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "a", "label": "A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	if _, err := Load(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateChatRanges(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"chat": {"temperature": 5}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "a", "label": "A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "chat.temperature") {
		t.Fatalf("expected chat.temperature error, got %v", err)
	}
}

func TestSanitizedStripsSecretsAndCopiesAllowed(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		DefaultProvider: "openrouter",
		Providers: map[string]ProviderConfig{
			"openrouter": {
				Type:    "openai_compatible",
				BaseURL: "https://openrouter.ai/api/v1",
				APIKey:  "should-never-leak",
				Models: []ModelConfig{
					{ID: "model-a", Label: "", SupportsTools: true, Default: true, MaxOutputTokens: 4096},
					{ID: "model-b", Label: "Model B", SupportsTools: true, MaxOutputTokens: 8192},
				},
			},
		},
		Tools: ToolsConfig{Enabled: true, Allowed: []string{"read_file"}, MaxIterations: 5},
		Chat:  ChatConfig{MaxHistoryMessages: 30, Stream: true, Temperature: 0.7},
	}
	out := cfg.Sanitized(nil)
	if out.Tools.Categories == nil {
		t.Fatalf("Categories should be normalized to empty map")
	}
	if got := out.Providers["openrouter"].Default; got != "model-a" {
		t.Fatalf("default model = %q, want model-a", got)
	}
	if got := out.Providers["openrouter"].Models[0].Label; got != "model-a" {
		t.Fatalf("fallback label = %q, want model-a", got)
	}
	if out.Tools.Allowed[0] != "read_file" {
		t.Fatalf("Allowed not copied")
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "should-never-leak") {
		t.Fatalf("sanitized output leaked api key: %s", raw)
	}
}

func TestDefaultModelAndFindModel(t *testing.T) {
	cfg := &Config{Providers: map[string]ProviderConfig{
		"p": {
			Models: []ModelConfig{
				{ID: "a", Label: "A", Default: false, MaxOutputTokens: 4096},
				{ID: "b", Label: "B", Default: true, MaxOutputTokens: 4096},
			},
		},
	}}
	if m, ok := cfg.DefaultModel("p"); !ok || m.ID != "b" {
		t.Fatalf("DefaultModel = %+v ok=%v", m, ok)
	}
	if _, _, ok := cfg.FindModel("p", "a"); !ok {
		t.Fatalf("FindModel should locate a")
	}
	if _, _, ok := cfg.FindModel("p", "missing"); ok {
		t.Fatalf("FindModel should not locate missing")
	}
	if _, ok := cfg.DefaultModel("missing"); ok {
		t.Fatalf("DefaultModel should not locate missing provider")
	}
}

func TestResolvePathAbsoluteAndRelative(t *testing.T) {
	tmp := t.TempDir()
	cfg := &Config{Path: filepath.Join(tmp, "bs-ai-config.json")}
	absPath := filepath.Join(tmp, "abs", "path", "db")
	if got := cfg.ResolvePath(absPath); got != absPath {
		t.Fatalf("absolute path not preserved: %s", got)
	}
	if got := cfg.ResolvePath("nested/db"); got != filepath.Join(tmp, "nested", "db") {
		t.Fatalf("relative path = %s", got)
	}
}

func TestResolveWebSearchEnvMissing(t *testing.T) {
	os.Unsetenv("BS_AI_BRAVE_KEY")
	_, _ = setupBothConfigAndModels(t, `{
		"default_provider": "openrouter",
		"web_search": {
			"enabled": true,
			"providers": {"brave": {"enabled": true, "api_key": "env:BS_AI_BRAVE_KEY"}}
		}
	}`, `{
		"providers": {
			"openrouter": {
				"type": "openai_compatible",
				"base_url": "https://openrouter.ai/api/v1",
				"api_key": "k",
				"models": [{"id": "a", "label": "A", "supports_tools": true, "default": true, "max_output_tokens": 4096}]
			}
		}
	}`)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "BS_AI_BRAVE_KEY") {
		t.Fatalf("expected web_search brave env error, got %v", err)
	}
}

func TestNestedPresentDetectsRawFields(t *testing.T) {
	raw := map[string]json.RawMessage{
		"chat": json.RawMessage(`{"temperature": 0.1}`),
	}
	if !nestedPresent(raw, "chat", "temperature") {
		t.Fatal("expected temperature to be present")
	}
	if nestedPresent(raw, "chat", "stream") {
		t.Fatal("stream should not be present")
	}
	if nestedPresent(raw, "missing", "anything") {
		t.Fatal("missing section should not be present")
	}
}

func TestValidateFileToolsGlobs(t *testing.T) {
	cfg := FileToolsConfig{
		MaxReadBytes:        4096,
		MaxLineReadBytes:    4096,
		MaxLineCount:        100,
		MaxFileSizeWarnMB:   1,
		BlockedPathPatterns: []string{"[unterminated"},
	}
	if err := validateFileTools(cfg); err == nil || !strings.Contains(err.Error(), "invalid glob") {
		t.Fatalf("expected invalid glob error, got %v", err)
	}
}

func TestParseErrorReturnsWrapped(t *testing.T) {
	_, _ = setupBothConfigAndModels(t, "{not json", minimalProviderModels())
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "parse AI config") {
		t.Fatalf("expected wrapped parse error, got %v", err)
	}
	if !errors.Is(err, err) { // sanity: errors package imported
		t.Fatal("errors package should be in scope")
	}
}

// memoryUsedFields is the set of MemoryConfig JSON field names that are read
// by the memory package (or validated here). If a new field is added to
// MemoryConfig without being consumed anywhere, this test fails so dead config
// cannot accumulate silently.
var memoryUsedFields = map[string]bool{
	"enabled": true, "directory": true, "fragments_dir": true, "archive_dir": true,
	"max_body_kb": true, "max_links_per_fragment": true, "max_ops_per_call": true,
	"max_result_bytes": true, "default_depth": true, "max_depth": true, "spread_factor": true,
	"persona_token_budget": true, "inject_persona": true, "inject_usage_guide": true,
	"retention_days": true, "auto_cleanup": true, "maintenance_interval": true,
	"salience_decay_per_week": true, "archive_threshold": true,
	"secret_scan": true, "near_duplicate_threshold": true,
	"synthesizer": true, "embeddings": true,
}

var synthesizerUsedFields = map[string]bool{
	"enabled": true, "provider": true, "model": true, "temperature": true,
	"max_output_tokens": true, "timeout_ms": true, "fallback_on_error": true,
}

var embeddingsUsedFields = map[string]bool{
	"enabled": true, "provider": true, "model": true, "dims": true,
}

func TestAllMemoryFieldsUsed(t *testing.T) {
	check := func(t *testing.T, typ reflect.Type, used map[string]bool, prefix string) {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			key := field.Name
			if tag, ok := field.Tag.Lookup("json"); ok {
				if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
					key = name
				}
			}
			if !used[key] {
				t.Errorf("config field %s.%s is not consumed anywhere", prefix, key)
			}
		}
	}
	check(t, reflect.TypeOf(MemoryConfig{}), memoryUsedFields, "memory")
	check(t, reflect.TypeOf(SynthesizerConfig{}), synthesizerUsedFields, "memory.synthesizer")
	check(t, reflect.TypeOf(EmbeddingsConfig{}), embeddingsUsedFields, "memory.embeddings")
}

package browserconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "bs-browser-config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingIsDisabled(t *testing.T) {
	ResetForTest()
	cfg, err := LoadPath(filepath.Join(t.TempDir(), "bs-browser-config.json"))
	if err != nil {
		t.Fatalf("LoadPath() unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("missing file should return a disabled config")
	}
	if cfg.ToolEnabled("browser_navigate") {
		t.Fatal("disabled config must not expose tools")
	}
}

func TestLoadDefaultsEveryToolOn(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"enabled": true}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled {
		t.Fatal("config should be enabled")
	}
	for _, name := range ToolNames() {
		if !cfg.ToolEnabled(name) {
			t.Errorf("tool %s should default to enabled", name)
		}
	}
}

func TestLoadPerToolGate(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
  "enabled": true,
  "tools": {
    "browser_screenshot": false,
    "browser_eval": false
  }
}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ToolEnabled("browser_screenshot") {
		t.Fatal("browser_screenshot should be disabled")
	}
	if cfg.ToolEnabled("browser_eval") {
		t.Fatal("browser_eval should be disabled")
	}
	if !cfg.ToolEnabled("browser_navigate") {
		t.Fatal("unlisted tool should default to enabled")
	}
	if cfg.ToolEnabled("browser_list_instances") == false {
		t.Fatal("unlisted discovery tool should stay enabled")
	}
}

func TestLoadEnabledFalseHidesAll(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"enabled": false}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("config should be disabled")
	}
	for _, name := range ToolNames() {
		if cfg.ToolEnabled(name) {
			t.Errorf("tool %s must be unavailable while the feature is disabled", name)
		}
	}
}

func TestLoadRejectsUnknownTool(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
  "enabled": true,
  "tools": { "browser_typo": true }
}`)
	_, err := LoadPath(path)
	if err == nil {
		t.Fatal("unknown tool name was accepted")
	}
	if !strings.Contains(err.Error(), "browser_typo") {
		t.Fatalf("error %q should mention the offending tool", err)
	}
}

func TestValidateBytes(t *testing.T) {
	if err := ValidateBytes([]byte(`{"enabled": true, "tools": {"browser_click": true}}`)); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if err := ValidateBytes([]byte(`{"enabled": true, "tools": {"browser_nope": true}}`)); err == nil {
		t.Fatal("invalid config accepted")
	}
	if err := ValidateBytes([]byte(`{"enabled": true, "tools": {"browser_click": "yes"}}`)); err == nil {
		t.Fatal("non-boolean tool value accepted")
	}
	if err := ValidateBytes([]byte(`not json`)); err == nil {
		t.Fatal("malformed JSON accepted")
	}
}

func TestEvalDefaultsAndMatching(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
  "enabled": true,
  "eval": {
    "default_mode": "cdp",
    "domains": { "youtube.com": "inject", "*.twitch.tv": "cdp" }
  }
}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultEvalMode() != "cdp" {
		t.Fatalf("default mode = %q, want cdp", cfg.DefaultEvalMode())
	}
	if got := cfg.EvalModeForURL("https://www.youtube.com/watch?v=1"); got != "inject" {
		t.Fatalf("youtube override = %q, want inject", got)
	}
	if got := cfg.EvalModeForURL("https://m.twitch.tv/x"); got != "cdp" {
		t.Fatalf("twitch override = %q, want cdp", got)
	}
	if got := cfg.EvalModeForURL("https://example.com/"); got != "cdp" {
		t.Fatalf("unlisted host should fall back to the default, got %q", got)
	}
	if got := cfg.EvalModeForURL(""); got != "cdp" {
		t.Fatalf("empty URL should fall back to the default, got %q", got)
	}
	if got := cfg.EvalModeForURL("HTTPS://WWW.YOUTUBE.COM/a"); got != "inject" {
		t.Fatalf("matching should be case-insensitive, got %q", got)
	}
}

func TestEvalDomainsArrayForm(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
  "enabled": true,
  "eval": { "domains": ["example.com"] }
}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultEvalMode() != "inject" {
		t.Fatalf("default mode = %q, want inject", cfg.DefaultEvalMode())
	}
	if got := cfg.EvalModeForURL("https://example.com/path"); got != "cdp" {
		t.Fatalf("array-form domain should map to cdp, got %q", got)
	}
	if got := cfg.EvalModeForURL("https://sub.example.com/path"); got != "cdp" {
		t.Fatalf("subdomain of array-form domain should map to cdp, got %q", got)
	}
}

func TestEvalValidation(t *testing.T) {
	cases := []string{
		`{"eval": {"default_mode": "remote"}}`,
		`{"eval": {"domains": {"example.com": "remote"}}}`,
		`{"eval": {"domains": {"": "cdp"}}}`,
		`{"eval": {"domains": {"*": "cdp"}}}`,
		`{"eval": {"domains": "cdp"}}`,
	}
	for _, c := range cases {
		if err := ValidateBytes([]byte(c)); err == nil {
			t.Errorf("config %s should be rejected", c)
		}
	}
}

func TestTimeoutsDefaultsAndBounds(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"enabled": true}`)
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultTimeoutMS() != 60000 || cfg.MaxTimeoutMS() != 600000 || cfg.SelectorTimeoutMS() != 10000 {
		t.Fatalf("unexpected defaults: %d/%d/%d", cfg.DefaultTimeoutMS(), cfg.MaxTimeoutMS(), cfg.SelectorTimeoutMS())
	}

	dir2 := t.TempDir()
	path2 := writeConfig(t, dir2, `{
  "enabled": true,
  "timeouts": {
    "default_command_timeout_ms": 30000,
    "max_command_timeout_ms": 90000,
    "selector_timeout_ms": 5000
  }
}`)
	cfg2, err := LoadPath(path2)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.DefaultTimeoutMS() != 30000 || cfg2.MaxTimeoutMS() != 90000 || cfg2.SelectorTimeoutMS() != 5000 {
		t.Fatalf("unexpected values: %d/%d/%d", cfg2.DefaultTimeoutMS(), cfg2.MaxTimeoutMS(), cfg2.SelectorTimeoutMS())
	}
}

func TestTimeoutsValidation(t *testing.T) {
	cases := []string{
		`{"timeouts": {"default_command_timeout_ms": 5000, "max_command_timeout_ms": 1000}}`,
		`{"timeouts": {"default_command_timeout_ms": 10}}`,
		`{"timeouts": {"selector_timeout_ms": -1}}`,
		`{"timeouts": {"max_command_timeout_ms": "huge"}}`,
	}
	for _, c := range cases {
		if err := ValidateBytes([]byte(c)); err == nil {
			t.Errorf("config %s should be rejected", c)
		}
	}
}

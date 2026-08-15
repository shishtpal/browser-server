// Package browserconfig loads bs-browser-config.json from the executable
// directory (or BS_BROWSER_CONFIG_PATH). It is an optional per-tool gate for
// the browser automation AI tools: when the file is missing, or "enabled" is
// false, or a tool's flag is false, the affected tools are unavailable and do
// not appear in the model's toolset, in the Chat tools panel, or on the
// execution path. It also carries two tuning sections — "eval" (per-domain
// execution modes for browser_eval) and "timeouts" (per-command and selector
// polling bounds) — which the browser bus reads live so admin editor changes
// hot-reload without a restart.
package browserconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	aiconfig "browser-server/internal/ai/config"
)

const defaultConfigFile = "bs-browser-config.json"

// Eval execution modes and timeout defaults shared with the browser bus. Keep
// them aligned with the extension's own fallbacks (inject default, 10s
// selector poll, 10m hard cap).
const (
	defaultEvalMode          = "inject"
	defaultCommandTimeoutMS  = 60000
	maxCommandTimeoutMS      = 600000
	defaultSelectorTimeoutMS = 10000
	minTimeoutMS             = 100
)

// CatalogEntry documents one configurable browser tool. The catalog is the
// single source of truth for which browser tools exist: the loader defaults
// every entry to enabled and rejects unknown keys, while the tool
// constructors in internal/ai/browser consult ToolEnabled for the same names.
type CatalogEntry struct {
	Name        string
	Description string
}

// catalog lists every browser_* tool the AI layer can expose. Keep it in sync
// with the tools built by internal/ai/browser (Tools): adding a tool there
// without a catalog entry here makes it unusable (rejected as unknown), and
// removing one here orphans the flag.
var catalog = []CatalogEntry{
	{Name: "browser_list_instances", Description: "List online browser profiles"},
	{Name: "browser_list_tabs", Description: "List tabs for a browser profile"},
	{Name: "browser_stats", Description: "Browser automation health snapshot"},
	{Name: "browser_navigate", Description: "Navigate a tab to a URL and wait for load"},
	{Name: "browser_click", Description: "Click an element"},
	{Name: "browser_type", Description: "Type text into an input"},
	{Name: "browser_press", Description: "Send a key or shortcut"},
	{Name: "browser_scroll", Description: "Scroll the page or an element"},
	{Name: "browser_wait", Description: "Wait for a duration or a selector"},
	{Name: "browser_scrape", Description: "Extract structured page data"},
	{Name: "browser_eval", Description: "Run JavaScript in the page"},
	{Name: "browser_screenshot", Description: "Capture a PNG screenshot"},
	{Name: "browser_new_tab", Description: "Open a new tab"},
	{Name: "browser_close_tab", Description: "Close the target tab"},
	{Name: "browser_focus", Description: "Focus an element"},
	{Name: "browser_select", Description: "Set a <select> option"},
	{Name: "browser_check", Description: "Check or uncheck a checkbox"},
	{Name: "browser_hover", Description: "Fire hover on an element"},
	{Name: "browser_drag", Description: "Drag an element to a target"},
	{Name: "browser_get_cookies", Description: "Read cookies for an origin"},
	{Name: "browser_set_cookie", Description: "Set a cookie"},
	{Name: "browser_pdf", Description: "Export the page to PDF"},
	{Name: "browser_bring_to_front", Description: "Foreground the target tab"},
	{Name: "browser_storage", Description: "Read/write localStorage or sessionStorage"},
	{Name: "browser_execute", Description: "Run a multi-step browser workflow"},
}

// Config is the parsed bs-browser-config.json.
type Config struct {
	Enabled  bool            `json:"enabled"`
	Path     string          `json:"-"`
	Tools    map[string]bool `json:"tools"`
	Eval     EvalConfig      `json:"eval"`
	Timeouts TimeoutsConfig  `json:"timeouts"`
}

// EvalConfig controls how browser_eval and eval steps execute. The extension
// supports two modes: "inject" (isolated-world <script> injection, no
// infobar) and "cdp" (CDP Runtime.evaluate via the debugger API, which shows
// an infobar but bypasses page CSP that blocks injected scripts). A call's
// explicit "mode" param always wins; these defaults apply when the call omits
// it, and the domain map lets an operator force a mode for specific hosts so
// CSP-strict sites work without the model having to request CDP explicitly.
type EvalConfig struct {
	// DefaultMode is used when a call omits "mode" and no domain rule
	// applies. "" resolves to "inject".
	DefaultMode string `json:"default_mode"`
	// Domains maps a host (optionally with a "*. " subdomain wildcard) to a
	// mode. The JSON form accepts either an object {"youtube.com": "cdp"} or
	// an array of host names, each of which maps to "cdp".
	Domains DomainModes `json:"domains"`
}

// DomainModes maps a host/wildcard pattern to an eval mode.
type DomainModes map[string]string

// UnmarshalJSON accepts either an object {"host": "cdp", ...} or an array of
// host names, each of which maps to "cdp", so operators can pick the shape
// they prefer.
func (d *DomainModes) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err == nil {
		*d = obj
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	m := make(DomainModes, len(list))
	for _, host := range list {
		m[host] = "cdp"
	}
	*d = m
	return nil
}

// TimeoutsConfig bounds per-command execution and selector polling.
type TimeoutsConfig struct {
	// DefaultCommandTimeoutMS applies when a command omits timeout_ms.
	// 0 resolves to 60000.
	DefaultCommandTimeoutMS int `json:"default_command_timeout_ms"`
	// MaxCommandTimeoutMS is a hard ceiling on timeout_ms across every entry
	// point (AI tools, REST, CLI). 0 resolves to 600000. Must be >= default.
	MaxCommandTimeoutMS int `json:"max_command_timeout_ms"`
	// SelectorTimeoutMS is the default selector-polling budget for
	// browser_wait. 0 resolves to 10000.
	SelectorTimeoutMS int `json:"selector_timeout_ms"`
}

var global atomic.Pointer[Config]

func init() {
	global.Store(&Config{})
}

// Get returns the last-loaded config. It is never nil; before Load it returns
// a disabled zero config.
func Get() *Config { return global.Load() }

// Enabled reports whether the feature-level flag is on for the active config.
func Enabled() bool { return Get().Enabled }

// ResetForTest replaces the global config with a fresh disabled one. It is
// only intended for tests; production callers should use Load.
func ResetForTest() { global.Store(&Config{}) }

// ToolNames returns the canonical browser tool names, sorted.
func ToolNames() []string {
	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}

// ToolEnabled reports whether a browser tool is currently available. Missing
// keys default to enabled; the feature-level Enabled flag must also be set.
func (c *Config) ToolEnabled(name string) bool {
	if c == nil || !c.Enabled {
		return false
	}
	enabled, ok := c.Tools[name]
	return !ok || enabled
}

// DefaultEvalMode returns the configured global eval mode.
func (c *Config) DefaultEvalMode() string {
	if c == nil || c.Eval.DefaultMode == "" {
		return defaultEvalMode
	}
	return c.Eval.DefaultMode
}

// EvalModeForURL returns the effective eval mode for a tab URL: the first
// matching domain override, or the global default. An empty host (for example
// a not-yet-navigated "new" tab) still resolves to the global default so the
// configured mode applies deterministically.
func (c *Config) EvalModeForURL(rawURL string) string {
	if mode, ok := c.DomainMode(rawURL); ok {
		return mode
	}
	return c.DefaultEvalMode()
}

// DomainMode returns the configured mode override for a tab URL, if any.
// Matching is host-level and case-insensitive: a pattern such as "youtube.com"
// matches the exact host plus every subdomain (www.youtube.com,
// music.youtube.com, ...), so a leading "*. " is optional.
func (c *Config) DomainMode(rawURL string) (string, bool) {
	if c == nil || len(c.Eval.Domains) == 0 {
		return "", false
	}
	host := hostOf(rawURL)
	if host == "" {
		return "", false
	}
	for pattern, mode := range c.Eval.Domains {
		if host == pattern || strings.HasSuffix(host, "."+pattern) {
			return mode, true
		}
	}
	return "", false
}

// DefaultTimeoutMS returns the configured per-command default timeout.
func (c *Config) DefaultTimeoutMS() int {
	if c == nil || c.Timeouts.DefaultCommandTimeoutMS <= 0 {
		return defaultCommandTimeoutMS
	}
	return c.Timeouts.DefaultCommandTimeoutMS
}

// MaxTimeoutMS returns the configured hard ceiling on per-command timeouts.
func (c *Config) MaxTimeoutMS() int {
	if c == nil || c.Timeouts.MaxCommandTimeoutMS <= 0 {
		return maxCommandTimeoutMS
	}
	return c.Timeouts.MaxCommandTimeoutMS
}

// SelectorTimeoutMS returns the configured default selector-polling budget.
func (c *Config) SelectorTimeoutMS() int {
	if c == nil || c.Timeouts.SelectorTimeoutMS <= 0 {
		return defaultSelectorTimeoutMS
	}
	return c.Timeouts.SelectorTimeoutMS
}

// EvalModeForURL resolves the effective eval mode for a tab URL from the
// currently active config. It is installed on the browser bus so eval and
// orchestrate commands that omit an explicit "mode" default to the configured
// mode, letting operators force CDP execution on CSP-strict domains. With no
// config it returns the extension's natural default ("inject").
func EvalModeForURL(rawURL string) string {
	return Get().EvalModeForURL(rawURL)
}

// CommandLimits returns the active per-command timeout bounds in milliseconds
// (default, then hard maximum). It is installed on the browser bus so the AI
// executor, the REST /api/browser/cmd path, and the CLI all honor
// bs-browser-config.json and pick up admin editor changes live.
func CommandLimits() (int, int) {
	return Get().DefaultTimeoutMS(), Get().MaxTimeoutMS()
}

// hostOf extracts the lowercase hostname from a tab URL or a bare host. It is
// lenient: an empty or unparseable input returns "".
func hostOf(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// normalizeDomain canonicalizes a configured domain pattern to a bare,
// lowercase hostname. Scheme and port are stripped, "*.example.com" collapses
// to "example.com" (the matcher treats a bare host as covering its
// subdomains), and empty or wildcard-only patterns are rejected.
func normalizeDomain(s string) (string, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid domain %q", s)
	}
	host := strings.TrimPrefix(strings.TrimPrefix(u.Hostname(), "*."), "*")
	host = strings.TrimSuffix(host, ".")
	if host == "" || strings.ContainsAny(host, " /@") {
		return "", fmt.Errorf("invalid domain %q", s)
	}
	return strings.ToLower(host), nil
}

// Load reads bs-browser-config.json, applies defaults, validates, and
// atomically replaces the global config. A missing file yields a disabled
// config and is not an error (mirrors bs-quiz-config.json behaviour).
func Load() (*Config, error) {
	path := os.Getenv("BS_BROWSER_CONFIG_PATH")
	if path == "" {
		exeDir, err := aiconfig.ExecutableDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(exeDir, defaultConfigFile)
	}
	return LoadPath(path)
}

// LoadPath reloads an explicit path and atomically publishes the resulting
// rules. It intentionally ignores BS_BROWSER_CONFIG_PATH so the admin file
// API always reloads the same file it just committed.
func LoadPath(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			cfg.Enabled = false
			cfg.Path = path
			global.Store(cfg)
			log.Printf("Browser automation tools disabled: no config file at %s", path)
			return cfg, nil
		}
		return nil, fmt.Errorf("read browser config: %w", err)
	}

	cfg, err := parseBytes(content, path)
	if err != nil {
		return nil, err
	}
	global.Store(cfg)
	return cfg, nil
}

// ValidateBytes performs the same defaulting and semantic checks as Load
// without changing the process-global active browser rules.
func ValidateBytes(content []byte) error {
	_, err := parseBytes(content, defaultConfigFile)
	return err
}

func defaultConfig() *Config {
	tools := make(map[string]bool, len(catalog))
	for _, entry := range catalog {
		tools[entry.Name] = true
	}
	return &Config{Enabled: true, Tools: tools}
}

func parseBytes(content []byte, path string) (*Config, error) {
	cfg := defaultConfig()
	cfg.Path = path
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse browser config: %w", err)
	}
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("browser config: %w", err)
	}
	return cfg, nil
}

func validate(cfg *Config) error {
	known := make(map[string]bool, len(catalog))
	for _, entry := range catalog {
		known[entry.Name] = true
	}
	for name := range cfg.Tools {
		if !known[name] {
			return fmt.Errorf("tools: unknown browser tool %q", name)
		}
	}
	// Missing keys default to enabled so the config stays forward-compatible
	// with newly added tools.
	for _, entry := range catalog {
		if _, ok := cfg.Tools[entry.Name]; !ok {
			cfg.Tools[entry.Name] = true
		}
	}
	if err := validateEval(cfg); err != nil {
		return err
	}
	return validateTimeouts(cfg)
}

// validateEval normalizes the eval section: modes are canonicalized, empty
// default_mode resolves to "inject", and domain patterns are collapsed to
// bare hostnames. Invalid modes or patterns are rejected so a typo surfaces
// at save time.
func validateEval(cfg *Config) error {
	rawMode := strings.ToLower(strings.TrimSpace(cfg.Eval.DefaultMode))
	mode := normalizeMode(rawMode)
	if rawMode != "" && mode == "" {
		return fmt.Errorf("eval.default_mode must be \"inject\" or \"cdp\"")
	}
	if mode == "" {
		mode = defaultEvalMode
	}
	cfg.Eval.DefaultMode = mode

	for raw, m := range cfg.Eval.Domains {
		host, err := normalizeDomain(raw)
		if err != nil {
			return fmt.Errorf("eval.domains: %w", err)
		}
		value := normalizeMode(m)
		if value == "" {
			return fmt.Errorf("eval.domains[%q]: mode must be \"inject\" or \"cdp\"", raw)
		}
		delete(cfg.Eval.Domains, raw)
		cfg.Eval.Domains[host] = value
	}
	return nil
}

// validateTimeouts normalizes the timeouts section. Zero values fall back to
// the package defaults; negative values are rejected as typos and the hard
// maximum must be >= the default so a command can always express at least the
// default timeout.
func validateTimeouts(cfg *Config) error {
	if cfg.Timeouts.DefaultCommandTimeoutMS < 0 {
		return fmt.Errorf("timeouts.default_command_timeout_ms must be at least %d ms", minTimeoutMS)
	}
	if cfg.Timeouts.DefaultCommandTimeoutMS == 0 {
		cfg.Timeouts.DefaultCommandTimeoutMS = defaultCommandTimeoutMS
	}
	if cfg.Timeouts.DefaultCommandTimeoutMS < minTimeoutMS {
		return fmt.Errorf("timeouts.default_command_timeout_ms must be at least %d ms", minTimeoutMS)
	}
	if cfg.Timeouts.MaxCommandTimeoutMS < 0 {
		return fmt.Errorf("timeouts.max_command_timeout_ms must be positive")
	}
	if cfg.Timeouts.MaxCommandTimeoutMS == 0 {
		cfg.Timeouts.MaxCommandTimeoutMS = maxCommandTimeoutMS
	}
	if cfg.Timeouts.MaxCommandTimeoutMS < cfg.Timeouts.DefaultCommandTimeoutMS {
		return fmt.Errorf("timeouts.max_command_timeout_ms (%d) must be >= default_command_timeout_ms (%d)",
			cfg.Timeouts.MaxCommandTimeoutMS, cfg.Timeouts.DefaultCommandTimeoutMS)
	}
	if cfg.Timeouts.SelectorTimeoutMS < 0 {
		return fmt.Errorf("timeouts.selector_timeout_ms must be at least %d ms", minTimeoutMS)
	}
	if cfg.Timeouts.SelectorTimeoutMS == 0 {
		cfg.Timeouts.SelectorTimeoutMS = defaultSelectorTimeoutMS
	}
	if cfg.Timeouts.SelectorTimeoutMS < minTimeoutMS {
		return fmt.Errorf("timeouts.selector_timeout_ms must be at least %d ms", minTimeoutMS)
	}
	return nil
}

func normalizeMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "inject":
		return "inject"
	case "cdp":
		return "cdp"
	default:
		return ""
	}
}

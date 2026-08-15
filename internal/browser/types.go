// Package browser implements the extension command channel: a per-instance
// command bus that lets the AI (web chat and bs-ai-chat) drive the user's
// real browser with explicit multi-browser / multi-tab targeting.
//
// State model
//
//   - One Instance per browser profile (extension instance_id, UUID persisted
//     in chrome.storage.local). Heartbeats keep it online; a TTL marks it
//     offline when the browser closes.
//   - One Tab per tab_uuid (extension-assigned, stable across tabId reuse).
//   - One Command per AI action; it travels queued → (extension) → finished
//     (succeeded/failed/timed_out).
//
// The bus is intentionally in-memory: the long-running server is the single
// source of truth, the extension talks to it over HTTP/SSE, and the CLI talks
// to it over the same HTTP API. No SQLite persistence is needed for v1.
package browser

import (
	"encoding/json"
	"time"
)

// Command statuses.
const (
	StatusQueued    = "queued"
	StatusSent      = "sent"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusTimedOut  = "timed_out"
)

// Action names understood by the extension content script / background.
const (
	ActionNavigate   = "navigate"
	ActionClick      = "click"
	ActionType       = "type"
	ActionPress      = "press"
	ActionScroll     = "scroll"
	ActionWait       = "wait"
	ActionScrape     = "scrape"
	ActionEval       = "eval"
	ActionScreenshot = "screenshot"
	ActionNewTab     = "new_tab"
	ActionCloseTab   = "close_tab"

	// Element-state interactions (M9).
	ActionFocus        = "focus"
	ActionSelect       = "select"
	ActionCheck        = "check"
	ActionHover        = "hover"
	ActionDrag         = "drag"
	ActionBringToFront = "bring_to_front"

	// Background-API actions (M9): handled in the extension background script.
	ActionGetCookies = "get_cookies"
	ActionSetCookie  = "set_cookie"
	ActionPDF        = "pdf"

	// Page-storage actions (M11): read/write localStorage and sessionStorage in
	// the page's main world (handled by the content script, so it also works as
	// a browser_execute step).
	ActionStorage = "storage"

	// Orchestration (M10): atomic multi-step workflow.
	ActionOrchestrate = "orchestrate"
)

// Instance is one browser profile (one extension instance).
type Instance struct {
	InstanceID string    `json:"instance_id"`
	UserID     int       `json:"user_id"`
	Browser    string    `json:"browser"` // chrome | firefox | edge | ...
	Version    string    `json:"version"`
	Label      string    `json:"label"`
	Online     bool      `json:"online"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// Tab is one browser tab within an instance, addressed by its stable
// extension-assigned tab_uuid.
type Tab struct {
	TabUUID    string `json:"tab_uuid"`
	InstanceID string `json:"instance_id"`
	TabID      int    `json:"tab_id"`
	WindowID   int    `json:"window_id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Active     bool   `json:"active"`
	// LastActiveAt is nil when the tab has never been active (the extension
	// sends JSON null for inactive tabs; an empty string would fail
	// time.Time.UnmarshalJSON and break the entire tab-sync request).
	LastActiveAt *time.Time `json:"last_active_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// Command is one AI browser action addressed to a specific tab.
type Command struct {
	CommandID  string          `json:"command_id"`
	InstanceID string          `json:"instance_id"`
	TabUUID    string          `json:"tab_uuid"`
	SessionID  string          `json:"session_id"`
	Action     string          `json:"action"`
	Params     json.RawMessage `json:"params,omitempty"`
	Status     string          `json:"status"`
	TimeoutMS  int             `json:"timeout_ms,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	SentAt     *time.Time      `json:"sent_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	Result     *CommandResult  `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// CommandResult is the extension's final payload for one command.
type CommandResult struct {
	Page   *PageInfo       `json:"page,omitempty"`
	Scrape json.RawMessage `json:"scrape,omitempty"` // action-specific structured data
	Data   string          `json:"data,omitempty"`   // data URL for screenshots, text for eval etc.
	Path   string          `json:"path,omitempty"`   // local filesystem path when the server persisted the payload (screenshots)
	Via    string          `json:"via,omitempty"`    // how the result was produced when it differs from the requested mode, e.g. "cdp-fallback"
}

// PageInfo describes the page after an action so the AI can confirm context.
type PageInfo struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// BrowserRef selects a browser profile (level-1 targeting).
type BrowserRef struct {
	InstanceID  string `json:"instance_id,omitempty"`
	Label       string `json:"label,omitempty"`
	FirstOnline bool   `json:"first_online,omitempty"`
}

// TabRef selects a tab within the browser (level-2 targeting).
type TabRef struct {
	UUID  string `json:"uuid,omitempty"`  // exact tab_uuid
	URL   string `json:"url,omitempty"`   // glob, e.g. https://gmail.com/*
	Title string `json:"title,omitempty"` // glob, e.g. Gmail*
	Role  string `json:"role,omitempty"`  // active | new
}

// TargetRef is the full targeting model every browser tool accepts.
type TargetRef struct {
	Browser   *BrowserRef `json:"browser,omitempty"`
	Tab       *TabRef     `json:"tab,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
}

// CreateCommandRequest is what the AI tool layer submits to the bus.
type CreateCommandRequest struct {
	Target    TargetRef       `json:"target"`
	Action    string          `json:"action"`
	Params    json.RawMessage `json:"params,omitempty"`
	TimeoutMS int             `json:"timeout_ms,omitempty"`
}

// ResultRequest is what the extension posts back.
type ResultRequest struct {
	CommandID string         `json:"command_id"`
	Status    string         `json:"status"` // processing | succeeded | failed
	Result    *CommandResult `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ResolutionError carries a stable code and candidate lists so the AI can
// self-correct without a second round trip.
type ResolutionError struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Candidates []string `json:"candidates,omitempty"`
}

func (e *ResolutionError) Error() string {
	return e.Message
}

// Error codes used by resolution and command creation.
const (
	ErrCodeBrowserNotFound       = "browser_not_found"
	ErrCodeBrowserAmbiguous      = "browser_ambiguous"
	ErrCodeBrowserOffline        = "browser_offline"
	ErrCodeTabNotFound           = "tab_not_found"
	ErrCodeTabAmbiguous          = "tab_ambiguous"
	ErrCodeTabBusy               = "tab_busy"
	ErrCodeNoBrowser             = "no_browser_online"
	ErrCodeCommandNotFound       = "command_not_found"
	ErrCodeInvalidTarget         = "invalid_target"
	ErrCodeUnknownAction         = "unknown_action"
	ErrCodeInstanceNotRegistered = "instance_not_registered"
)

// Event is pushed over SSE to one instance's command channel.
type Event struct {
	Type    string   `json:"type"`
	Command *Command `json:"command,omitempty"`
}

// Event types.
const (
	EventCommand = "command"
	EventPing    = "ping"
)

// TTLs and defaults.
const (
	// InstanceTTL marks an instance offline when no heartbeat arrives within
	// this window. The extension heartbeat alarm runs every 30s.
	InstanceTTL = 90 * time.Second
	// DefaultCommandTimeout bounds a command if the tool did not specify one.
	DefaultCommandTimeout = 60 * time.Second
	// MaxCommandTimeout is a hard ceiling on per-command timeouts.
	MaxCommandTimeout = 10 * time.Minute
	// MaxCommandsPerInstance bounds the retained command log per instance.
	MaxCommandsPerInstance = 200
	// DefaultSessionID is used when the tool omits session_id; it makes every
	// such command contend for the same per-tab owner slot, which is fine.
	DefaultSessionID = "default"
)

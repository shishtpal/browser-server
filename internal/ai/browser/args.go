package browser

import (
	"encoding/json"

	corebrowser "browser-server/internal/browser"
)

// args is the union of all browser tool arguments. Per-action field
// validation uses actionFields so providers cannot sneak unknown keys through.
type args struct {
	Target    corebrowser.TargetRef `json:"target"`
	TimeoutMS int                   `json:"timeout_ms"`

	// Mode selects how JS eval steps run inside browser_execute: "inject"
	// (default, main-world <script> injection) or "cdp" (debugger fallback).
	Mode string `json:"mode"`

	URL               string          `json:"url"`
	Selector          json.RawMessage `json:"selector"`
	Source            json.RawMessage `json:"source"`
	To                json.RawMessage `json:"to"`
	Value             string          `json:"value"`
	Label             string          `json:"label"`
	Text              string          `json:"text"`
	Clear             bool            `json:"clear"`
	Keys              string          `json:"keys"`
	Direction         string          `json:"direction"`
	Amount            int             `json:"amount"`
	X                 int             `json:"x"`
	Y                 int             `json:"y"`
	Checked           bool            `json:"checked"`
	WaitMs            int             `json:"wait_ms"`
	SelectorTimeoutMS int             `json:"selector_timeout_ms"`
	Expression        string          `json:"expression"`
	Extract           []string        `json:"extract"`
	Scope             string          `json:"scope"`
	RowsToCSV         bool            `json:"rows_to_csv"`
	MaxLinks          int             `json:"max_links"`
	IncludeHidden     bool            `json:"include_hidden"`
	Name              string          `json:"name"`
	Path              string          `json:"path"`
	Domain            string          `json:"domain"`
	Secure            bool            `json:"secure"`
	HTTPOnly          bool            `json:"http_only"`

	// Storage selects the Web Storage area ("local" | "session") and the
	// operation ("get" | "set" | "remove" | "list"). Key applies to
	// get/set/remove; value to set; raw opts into untruncated values.
	StorageType   string `json:"type"`
	StorageAction string `json:"action"`
	Key           string `json:"key"`
	Raw           bool   `json:"raw"`

	Steps           []step `json:"steps"`
	ScreenshotAfter bool   `json:"screenshot_after"`
	RollbackURL     string `json:"rollback_url"`
}

// step is one primitive inside a browser_execute orchestration. Params reuse
// the same shape as the primitive tool payloads.
type step struct {
	Action     string          `json:"action"`
	Params     json.RawMessage `json:"params,omitempty"`
	Selector   json.RawMessage `json:"selector,omitempty"`   // convenience: step-scoped selector
	Screenshot bool            `json:"screenshot,omitempty"` // capture after this step
}

// actionFields is the exact set of keys accepted by any browser_* tool.
var actionFields = map[string]bool{
	"target": true, "timeout_ms": true,
	"url": true, "selector": true, "source": true, "to": true,
	"value": true, "label": true, "text": true, "clear": true, "keys": true,
	"direction": true, "amount": true, "x": true, "y": true,
	"checked": true, "wait_ms": true, "selector_timeout_ms": true,
	"expression": true, "extract": true, "scope": true, "rows_to_csv": true,
	"max_links": true, "include_hidden": true,
	"name": true, "path": true, "domain": true, "secure": true, "http_only": true,
	"type": true, "action": true, "key": true, "raw": true,
	"steps": true, "mode": true, "screenshot_after": true, "rollback_url": true,
}

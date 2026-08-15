package browser

import (
	"encoding/json"
	"fmt"
	"strings"

	browserconfig "browser-server/internal/ai/browser/config"
	corebrowser "browser-server/internal/browser"
)

// actionSpec centralizes everything unique about one browser action so the
// shared executor stays generic: read-only flag, per-action validation, and
// command param extraction.
type actionSpec struct {
	// readOnly means no command is created (registry info tool).
	readOnly bool
	// validate enforces per-action argument requirements.
	validate func(*args) error
	// params builds the extension-side params for the command.
	params func(*args) map[string]any
}

// specs is the single dispatch table for every action the extension and bus
// know. Adding a tool = one entry here + one spec in the tool list files.
var specs = map[string]actionSpec{
	corebrowser.ActionNavigate: {
		validate: func(a *args) error {
			if strings.TrimSpace(a.URL) == "" {
				return fmt.Errorf("url is required for navigate")
			}
			if !strings.HasPrefix(a.URL, "http://") && !strings.HasPrefix(a.URL, "https://") {
				return fmt.Errorf("url must start with http:// or https://")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return put(map[string]any{}, "url", a.URL)
		},
	},
	corebrowser.ActionClick: {
		validate: requireSelector("click"),
		params:   func(a *args) map[string]any { return put(map[string]any{}, "selector", a.Selector) },
	},
	corebrowser.ActionType: {
		validate: func(a *args) error {
			if err := requireSelector("type")(a); err != nil {
				return err
			}
			if a.Text == "" && !a.Clear {
				return fmt.Errorf("text (or clear=true) is required for type")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{}, "selector", a.Selector, "text", a.Text, "clear", a.Clear)
		},
	},
	corebrowser.ActionPress: {
		validate: func(a *args) error {
			if strings.TrimSpace(a.Keys) == "" {
				return fmt.Errorf("keys is required for press (e.g. Enter, Tab, Escape)")
			}
			return nil
		},
		params: func(a *args) map[string]any { return put(map[string]any{}, "keys", a.Keys) },
	},
	corebrowser.ActionScroll: {
		validate: func(a *args) error {
			if a.Direction != "" && a.Direction != "up" && a.Direction != "down" && a.Direction != "top" && a.Direction != "bottom" {
				return fmt.Errorf("direction must be up, down, top, or bottom")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{},
				"direction", a.Direction, "amount", a.Amount,
				"x", a.X, "y", a.Y, "selector", a.Selector)
		},
	},
	corebrowser.ActionWait: {
		validate: func(a *args) error {
			if a.WaitMs <= 0 && len(a.Selector) == 0 {
				return fmt.Errorf("wait_ms or selector is required for wait")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			timeout := a.SelectorTimeoutMS
			if timeout <= 0 {
				timeout = browserconfig.Get().SelectorTimeoutMS()
			}
			return puts(map[string]any{},
				"wait_ms", a.WaitMs, "selector", a.Selector,
				"selector_timeout_ms", timeout)
		},
	},
	corebrowser.ActionScrape: {
		params: func(a *args) map[string]any {
			return puts(map[string]any{},
				"selector", a.Selector,
				"extract", a.Extract, "max_links", a.MaxLinks,
				"include_hidden", a.IncludeHidden, "scope", a.Scope, "rows_to_csv", a.RowsToCSV)
		},
	},
	corebrowser.ActionEval: {
		validate: func(a *args) error {
			if strings.TrimSpace(a.Expression) == "" {
				return fmt.Errorf("expression is required for eval")
			}
			return validateMode(a.Mode)
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{}, "expression", a.Expression, "mode", a.Mode)
		},
	},
	corebrowser.ActionScreenshot: {},
	corebrowser.ActionNewTab: {
		validate: func(a *args) error {
			if a.URL != "" && !strings.HasPrefix(a.URL, "http://") && !strings.HasPrefix(a.URL, "https://") {
				return fmt.Errorf("url must start with http:// or https://")
			}
			return nil
		},
		params: func(a *args) map[string]any { return put(map[string]any{}, "url", a.URL) },
	},
	corebrowser.ActionCloseTab: {},
	corebrowser.ActionFocus: {
		validate: requireSelector("focus"),
		params:   func(a *args) map[string]any { return put(map[string]any{}, "selector", a.Selector) },
	},
	corebrowser.ActionSelect: {
		validate: func(a *args) error {
			if err := requireSelector("select")(a); err != nil {
				return err
			}
			if a.Value == "" && a.Label == "" {
				return fmt.Errorf("value or label is required for select")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{}, "selector", a.Selector, "value", a.Value, "label", a.Label)
		},
	},
	corebrowser.ActionCheck: {
		validate: requireSelector("check"),
		params: func(a *args) map[string]any {
			// Checked:true is hard-coded because the executor's
			// fireInputEvents already dispatches proper change events; the
			// only useful state is "checked". A false value would require a
			// separate "uncheck" action, which the extension does not
			// implement.
			return puts(map[string]any{}, "selector", a.Selector, "checked", true)
		},
	},
	corebrowser.ActionHover: {
		validate: requireSelector("hover"),
		params:   func(a *args) map[string]any { return put(map[string]any{}, "selector", a.Selector) },
	},
	corebrowser.ActionDrag: {
		validate: func(a *args) error {
			if len(a.Source) == 0 || string(a.Source) == "null" {
				return fmt.Errorf("source selector is required for drag")
			}
			if len(a.To) == 0 || string(a.To) == "null" {
				return fmt.Errorf("to is required for drag (selector or {x,y})")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{}, "source", a.Source, "to", a.To)
		},
	},
	corebrowser.ActionBringToFront: {},
	corebrowser.ActionGetCookies: {
		params: func(a *args) map[string]any {
			return puts(map[string]any{}, "domain", a.Domain, "name", a.Name)
		},
	},
	corebrowser.ActionSetCookie: {
		validate: func(a *args) error {
			if strings.TrimSpace(a.Name) == "" {
				return fmt.Errorf("name is required for set_cookie")
			}
			return nil
		},
		params: func(a *args) map[string]any {
			return puts(map[string]any{},
				"name", a.Name, "value", a.Value, "url", a.URL,
				"path", a.Path, "domain", a.Domain, "secure", a.Secure, "http_only", a.HTTPOnly)
		},
	},
	corebrowser.ActionPDF: {},
	corebrowser.ActionStorage: {
		validate: validateStorage,
		params: func(a *args) map[string]any {
			return puts(map[string]any{},
				"type", a.StorageType, "action", a.StorageAction,
				"key", a.Key, "value", a.Value, "raw", a.Raw)
		},
	},
	corebrowser.ActionOrchestrate: {
		// Steps reuse the primitive validators via a synthetic args check in
		// workflow.go; params pass the raw step list straight through.
		params: orchestrateParams,
	},
}

func requireSelector(action string) func(*args) error {
	return func(a *args) error {
		if len(a.Selector) == 0 || string(a.Selector) == "null" {
			return fmt.Errorf("selector is required for %s", action)
		}
		return nil
	}
}

// validateStorage enforces browser_storage arguments: a storage area
// (local|session), an operation (get|set|remove|list), a key for the
// key-scoped operations, and a value for set.
func validateStorage(a *args) error {
	switch a.StorageType {
	case "local", "session":
	default:
		return fmt.Errorf("type must be \"local\" or \"session\"")
	}
	switch a.StorageAction {
	case "get", "set", "remove", "list":
	default:
		return fmt.Errorf("action must be \"get\", \"set\", \"remove\", or \"list\"")
	}
	if a.StorageAction != "list" && strings.TrimSpace(a.Key) == "" {
		return fmt.Errorf("key is required for storage %s", a.StorageAction)
	}
	if a.StorageAction == "set" && strings.TrimSpace(a.Value) == "" {
		return fmt.Errorf("value is required for storage set")
	}
	return nil
}

// put writes k=v into m when v is non-default (empty strings, zero numbers
// and false bools are treated as absent, keeping command payloads tight).
func put(m map[string]any, k string, v any) map[string]any {
	switch x := v.(type) {
	case nil:
		return m
	case string:
		if x == "" {
			return m
		}
	case int:
		if x == 0 {
			return m
		}
	case bool:
		if !x {
			return m
		}
	case []string:
		if len(x) == 0 {
			return m
		}
	case json.RawMessage:
		if len(x) == 0 || string(x) == "null" {
			return m
		}
	}
	m[k] = v
	return m
}

// puts applies put for each (key, value) pair.
func puts(m map[string]any, pairs ...any) map[string]any {
	for i := 0; i+1 < len(pairs); i += 2 {
		key, _ := pairs[i].(string)
		put(m, key, pairs[i+1])
	}
	return m
}

func paramsJSON(spec actionSpec, a *args) json.RawMessage {
	m := map[string]any{}
	if spec.params != nil {
		m = spec.params(a)
	}
	b, _ := json.Marshal(m)
	return b
}

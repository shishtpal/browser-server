package browser

import (
	"encoding/json"
	"fmt"

	browserconfig "browser-server/internal/ai/browser/config"
)

// targetSchema is inlined into every browser tool's parameter schema so
// providers see a self-contained definition (JSON $ref is not resolved by
// most OpenAI-compatible endpoints).
var targetSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"browser": map[string]any{
			"type": "object",
			"description": "Selects the browser profile. Omit to use the only online browser. " +
				"Set instance_id (exact), label (unique online label), or first_online=true.",
			"properties": map[string]any{
				"instance_id":  map[string]any{"type": "string", "description": "Exact instance UUID from browser_list_instances"},
				"label":        map[string]any{"type": "string", "description": "User-defined label, e.g. \"Work Chrome\""},
				"first_online": map[string]any{"type": "boolean", "description": "Use the single online browser (errors if more than one)"},
			},
		},
		"tab": map[string]any{
			"type": "object",
			"description": "Selects the tab. Set uuid (exact), url (glob, e.g. https://gmail.com/*), " +
				"title (glob, e.g. Gmail*), or role (active|new).",
			"properties": map[string]any{
				"uuid":  map[string]any{"type": "string", "description": "Exact tab_uuid from browser_list_tabs"},
				"url":   map[string]any{"type": "string", "description": "URL glob pattern"},
				"title": map[string]any{"type": "string", "description": "Title glob pattern"},
				"role":  map[string]any{"type": "string", "enum": []any{"active", "new"}, "description": "active = current tab, new = create a tab"},
			},
		},
		"session_id": map[string]any{
			"type":        "string",
			"description": "Task ownership token. Reuse the same value across a multi-step task so the tab stays locked to this task.",
		},
	},
}

// selectorSchema: how the model addresses an element on the page.
var selectorSchema = map[string]any{
	"oneOf": []any{
		map[string]any{"type": "object", "properties": map[string]any{"css": map[string]any{"type": "string", "description": "CSS selector, e.g. button[data-testid='compose']"}}, "required": []any{"css"}},
		map[string]any{"type": "object", "properties": map[string]any{"xpath": map[string]any{"type": "string", "description": "XPath expression"}}, "required": []any{"xpath"}},
		map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string", "description": "Visible text to find and click"}}, "required": []any{"text"}},
	},
}

// buildSchema composes a self-contained tool parameter schema. Target is
// always available; requireTarget puts it into required. The timeout_ms
// description and ceiling reflect bs-browser-config.json at build time.
func buildSchema(params map[string]any, required []string, description string, requireTarget bool) json.RawMessage {
	cfg := browserconfig.Get()
	def := cfg.DefaultTimeoutMS()
	max := cfg.MaxTimeoutMS()
	props := map[string]any{
		"target": targetSchema,
		"timeout_ms": map[string]any{
			"type":        "integer",
			"description": fmt.Sprintf("Per-command timeout in ms (default %d, max %d)", def, max),
			"minimum":     100,
			"maximum":     max,
		},
	}
	for k, v := range params {
		props[k] = v
	}
	req := required
	if requireTarget {
		req = append([]string{"target"}, required...)
	}
	b, _ := json.Marshal(map[string]any{
		"type":        "object",
		"description": description,
		"properties":  props,
		"required":    req,
	})
	return b
}

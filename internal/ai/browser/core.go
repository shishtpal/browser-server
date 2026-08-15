package browser

import (
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// core.go — the original Path-1 tool group: navigation, interaction and
// extraction. Tool specs map 1:1 onto actions the bus supports and reuse the
// shared executor.
func coreTools(client corebrowser.Client) []tools.Tool {
	return []tools.Tool{
		mk(client, "navigate",
			"Navigate the target tab to a URL and wait for the page to load. Returns the resulting page URL and title.",
			map[string]any{
				"url": map[string]any{"type": "string", "description": "Absolute http(s) URL"},
			},
			[]string{"url"}, true),
		mk(client, "click",
			"Click an element in the target tab using a CSS selector, XPath, or visible text.",
			map[string]any{"selector": selectorSchema}, []string{"selector"}, true),
		mk(client, "type",
			"Type text into an input (or clear it) in the target tab. Focuses the element, sets its value, and fires framework-compatible input events.",
			map[string]any{
				"selector": selectorSchema,
				"text":     map[string]any{"type": "string", "description": "Text to type"},
				"clear":    map[string]any{"type": "boolean", "description": "Clear the field before typing"},
			},
			[]string{"selector"}, true),
		mk(client, "press",
			"Send a key or shortcut to the focused element or page (Enter, Tab, Escape, ArrowDown, Ctrl+Enter, ...).",
			map[string]any{"keys": map[string]any{"type": "string", "description": "Key name or combo, e.g. Enter, Tab, Escape, Ctrl+S"}},
			[]string{"keys"}, true),
		mk(client, "scroll",
			"Scroll the page or an element. Use direction (up/down/top/bottom), an amount in px, or scroll a selector into view.",
			map[string]any{
				"direction": map[string]any{"type": "string", "enum": []any{"up", "down", "top", "bottom"}},
				"amount":    map[string]any{"type": "integer", "description": "Pixels to scroll"},
				"selector":  selectorSchema,
				"x":         map[string]any{"type": "integer"},
				"y":         map[string]any{"type": "integer"},
			}, nil, true),
		mk(client, "wait",
			"Wait for a duration and/or for a selector to appear in the target tab. Use after navigation before interacting with dynamic content.",
			map[string]any{
				"wait_ms":             map[string]any{"type": "integer", "description": "Sleep duration in ms"},
				"selector":            selectorSchema,
				"selector_timeout_ms": map[string]any{"type": "integer", "description": "How long to poll for the selector (default 10000)"},
			}, nil, true),
		mk(client, "scrape",
			"Extract structured data from the target tab: text, links (with hrefs), attributes, form fields, or an HTML table. Returns JSON.",
			map[string]any{
				"selector":       selectorSchema,
				"scope":          map[string]any{"type": "string", "enum": []any{"page", "element", "table"}, "description": "page = whole document; element = selector subtree; table = first <table>"},
				"extract":        map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []any{"text", "links", "attributes", "forms", "table"}}, "description": "What to extract (default: text + links)"},
				"rows_to_csv":    map[string]any{"type": "boolean", "description": "When extracting a table, also render rows to CSV (extension-side parsing)"},
				"max_links":      map[string]any{"type": "integer", "description": "Cap on extracted links (default 100)"},
				"include_hidden": map[string]any{"type": "boolean", "description": "Include hidden elements (default false)"},
			}, nil, true),
		mk(client, "eval",
			"Run a JavaScript expression in the target tab's main world and return its JSON-serialized value as a string under result.data (undefined becomes \"null\"). mode \"inject\" (default) evaluates by injecting a <script> tag from the content script — no debugger permission or infobar. If the page CSP blocks inline scripts (or nothing comes back), the extension automatically retries via CDP and the result reports result.via: \"cdp-fallback\" (the debugging infobar appears). mode \"cdp\" evaluates via CDP Runtime.evaluate from the background — bypasses page CSP, needs the debugger permission, shows the debugging infobar. Prefer expressions that return plain data (primitives, arrays, objects); non-serializable results (circular values, e.g. window) fail, and DOM nodes degrade to {}. Gated by configuration (browser_eval must be enabled).",
			map[string]any{
				"expression": map[string]any{"type": "string", "description": "JavaScript expression, e.g. document.title or JSON.stringify([...])"},
				"mode": map[string]any{
					"type":        "string",
					"enum":        []any{"inject", "cdp"},
					"description": "How to evaluate. Omit to use the server's bs-browser-config.json eval mode for the tab's domain (normally inject; the operator can force cdp per domain). inject = main-world <script> injection from the content script (no debugger permission or infobar); when the page CSP blocks inline scripts the extension automatically retries via CDP and reports result.via: \"cdp-fallback\". cdp = CDP Runtime.evaluate via the debugger API (bypasses page CSP, needs the debugger permission, shows the debugging infobar).",
				},
			},
			[]string{"expression"}, true),
		mk(client, "screenshot",
			"Capture a PNG screenshot of the target tab's visible area. Returns the server URL under result.data and the local file path under result.path; use result.path with ocr_image/read_file for further analysis.",
			map[string]any{}, nil, true),
		mk(client, "new_tab",
			"Open a new tab in the target browser (optionally at a URL) and wait for its content script to be ready. Returns the new tab_uuid.",
			map[string]any{"url": map[string]any{"type": "string", "description": "Optional URL to open"}},
			nil, true),
		mk(client, "close_tab",
			"Close the target tab.",
			map[string]any{}, nil, true),
	}
}

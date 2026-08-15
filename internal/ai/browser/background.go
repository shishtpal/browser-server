package browser

import (
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// background.go — background-API tools: whole-browser actions handled in the
// extension's background script (cookies, PDF export, window focus).
func backgroundTools(client corebrowser.Client) []tools.Tool {
	return []tools.Tool{
		mk(client, "get_cookies",
			"Read cookies for the target tab's current origin or a specific domain (background API). Returns match_name and value only (never includes httpOnly secrets).",
			map[string]any{
				"domain": map[string]any{"type": "string", "description": "Cookie domain (e.g. example.com). Default: tab origin."},
				"name":   map[string]any{"type": "string", "description": "Exact cookie name filter (default: all on the domain)."},
			}, nil, true),
		mk(client, "set_cookie",
			"Set one cookie on the target tab's current origin. Use for session persistence; prefer login flows via browser_click / browser_type over planting auth cookies.",
			map[string]any{
				"name":      map[string]any{"type": "string", "description": "Cookie name"},
				"value":     map[string]any{"type": "string", "description": "Cookie value"},
				"url":       map[string]any{"type": "string", "description": "Explicit url (default: tab origin). Required when the domain has no path."},
				"path":      map[string]any{"type": "string", "description": "Cookie path (default /)"},
				"domain":    map[string]any{"type": "string", "description": "Cookie domain (default: tab origin)"},
				"secure":    map[string]any{"type": "boolean", "description": "Mark as Secure"},
				"http_only": map[string]any{"type": "boolean", "description": "Mark as HttpOnly"},
			}, []string{"name", "value"}, true),
		mk(client, "pdf",
			"Export the current page of the target tab to PDF via CDP Page.printToPDF (works on Chromium browsers: Chrome, Edge, Brave, ...). Returns the server URL under result.data and the local file path under result.path; use result.path with read_file for further analysis.",
			map[string]any{}, nil, true),
		mk(client, "bring_to_front",
			"Switch focus to the target tab (foregrounds its window).",
			map[string]any{}, nil, true),
		mk(client, "storage",
			"Read or write the target tab's page Web Storage (localStorage or sessionStorage) from the page's main world. Actions: get (read one key), set (write one key; the value is stored as a raw string — JSON.stringify the value first if you need a structured payload), remove (delete one key), list (enumerate keys and values). Use raw:true to opt into full untruncated values; by default oversized values are capped (get: 8 KiB, list entries: 512 chars).",
			map[string]any{
				"type": map[string]any{
					"type": "string", "enum": []any{"local", "session"},
					"description": "Storage area: local = localStorage (persists per origin), session = sessionStorage (cleared when the tab closes). Required.",
				},
				"action": map[string]any{
					"type": "string", "enum": []any{"get", "set", "remove", "list"},
					"description": "Operation to run. Required.",
				},
				"key":   map[string]any{"type": "string", "description": "Storage key. Required for get/set/remove."},
				"value": map[string]any{"type": "string", "description": "Value to store (raw string; required for set)."},
				"raw":   map[string]any{"type": "boolean", "description": "Return full untruncated values (default false caps oversized values)."},
			}, []string{"type", "action"}, true),
	}
}

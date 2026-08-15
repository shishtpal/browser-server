package browser

import (
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// workflowTools returns the top-level orchestration tool.
//
// browser_execute collapses multi-step browser workflows into a single
// command so the tab is not re-resolved between steps (avoids stale-UUID
// races after navigate) and gives the model one result covering the whole
// flow.
func workflowTools(client corebrowser.Client) []tools.Tool {
	return []tools.Tool{
		mk(client, "execute",
			"Run a sequence of browser actions (navigate → wait → click/type/press/select/check → scrape → eval/storage → ...) atomically in one tab. This is the canonical way to do multi-step flows; individual browser_* tools are for probes only.",
			map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []any{"inject", "cdp"},
					"description": "How JS eval steps run. Omit to use the server's bs-browser-config.json eval mode for the tab's domain (normally inject; the operator can force cdp per domain). inject = evaluate in the page main world by injecting a <script> tag from the content script (no debugger permission or infobar). If the page CSP blocks inline scripts (or nothing comes back), the step automatically retries via CDP and reports via: \"cdp-fallback\" (debugging infobar appears). cdp = CDP Runtime.evaluate from the background via the debugger API (bypasses page CSP, but needs the debugger permission and shows the debugging infobar). A step's params.mode overrides this for that step.",
				},
				"steps": map[string]any{
					"type":        "array",
					"description": "Ordered steps; each entry is {action, params?, selector?}. Params take precedence over top-level keys for primitives that use them. Eval steps accept params {expression, mode?, timeout_ms?}.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"action":     map[string]any{"type": "string", "description": "Browser action name (same as the standalone tools without the browser_ prefix)"},
							"params":     map[string]any{"type": "object", "description": "Action-specific params (url, text, extract, expression, mode, timeout_ms, ...)"},
							"selector":   selectorSchema,
							"screenshot": map[string]any{"type": "boolean", "description": "Capture a screenshot after this step"},
						},
						"required": []any{"action"},
					},
					"minItems": 1,
				},
				"screenshot_after": map[string]any{"type": "boolean", "description": "Capture a screenshot after all steps complete."},
				"rollback_url":     map[string]any{"type": "string", "description": "Optional URL to restore when any step fails (best effort; ignored on success)."},
			},
			[]string{"steps"}, true),
	}
}

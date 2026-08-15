package browser

import (
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// elements.go — element-state tools that extend the core click/type domain
// with direct focus/selection/check/hover/drag semantics.
func elementTools(client corebrowser.Client) []tools.Tool {
	return []tools.Tool{
		mk(client, "focus",
			"Focus an element (input, textarea, button, link) so subsequent type/press operations target it. Use before type when a site requires explicit focus.",
			map[string]any{"selector": selectorSchema}, []string{"selector"}, true),
		mk(client, "select",
			"Set a <select> element by option value or visible label, firing change/input events.",
			map[string]any{
				"selector": selectorSchema,
				"value":    map[string]any{"type": "string", "description": "Option value to select (takes precedence over label)"},
				"label":    map[string]any{"type": "string", "description": "Visible option label to select"},
			}, []string{"selector"}, true),
		mk(client, "check",
			"Set a checkbox or radio button to checked (checked=true, default) or unchecked (checked=false). Dispatches proper change events.",
			map[string]any{
				"selector": selectorSchema,
				"checked":  map[string]any{"type": "boolean", "description": "Desired state (default true)"},
			}, []string{"selector"}, true),
		mk(client, "hover",
			"Fire mouseover/mousemove/mouseenter on an element (for menus, tooltips, and hover-only controls).",
			map[string]any{"selector": selectorSchema}, []string{"selector"}, true),
		mk(client, "drag",
			"Drag a source element to a target selector or coordinates. Dispatches full mousedown/mousemove/mouseup sequences compatible with most drag libraries.",
			map[string]any{
				"source": selectorSchema,
				"to":     map[string]any{"oneOf": []any{selectorSchema, map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "integer"}, "y": map[string]any{"type": "integer"}}, "required": []any{"x", "y"}}}, "description": "Destination selector or {x, y} coordinates"},
			}, []string{"source", "to"}, true),
	}
}

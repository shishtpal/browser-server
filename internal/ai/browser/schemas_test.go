package browser

import (
	"encoding/json"
	"testing"

	corebrowser "browser-server/internal/browser"
)

// TestEvalModeSchemaHasNoDefault guards against providers pre-filling the
// eval mode. The mode must NOT declare a JSON Schema default: OpenAI-compatible
// providers fill omitted args with declared defaults, which would hard-code
// "inject" into every browser_eval/browser_execute call and defeat the
// bs-browser-config.json eval section (default_mode and per-domain rules).
func TestEvalModeSchemaHasNoDefault(t *testing.T) {
	tools := Tools(nil)
	find := func(name string) map[string]any {
		t.Helper()
		for _, tool := range tools {
			if tool.Name != name {
				continue
			}
			var schema map[string]any
			if err := json.Unmarshal(tool.Schema, &schema); err != nil {
				t.Fatalf("%s: unmarshal schema: %v", name, err)
			}
			return schema
		}
		t.Fatalf("tool %s not found", name)
		return nil
	}
	properties := func(schema map[string]any) map[string]any {
		p, _ := schema["properties"].(map[string]any)
		return p
	}
	for _, name := range []string{"browser_eval", "browser_execute"} {
		mode, ok := properties(find(name))["mode"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema is missing the mode property", name)
		}
		if _, hasDefault := mode["default"]; hasDefault {
			t.Fatalf("%s mode schema must not declare a default; the bus injects the configured mode", name)
		}
	}
}

// TestEvalParamsOmitModeForInjection verifies the mechanism the bus relies on:
// when the model omits mode, the command params carry no "mode" key, so
// Bus.injectEvalModeLocked can default it from bs-browser-config.json. An
// explicit mode is preserved verbatim.
func TestEvalParamsOmitModeForInjection(t *testing.T) {
	spec := specs[corebrowser.ActionEval]

	raw := paramsJSON(spec, &args{Expression: "document.title"})
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if _, hasMode := m["mode"]; hasMode {
		t.Fatal("eval params must omit mode when the model does not set it, so the bus can inject the configured mode")
	}

	raw = paramsJSON(spec, &args{Expression: "document.title", Mode: "cdp"})
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if got := string(m["mode"]); got != `"cdp"` {
		t.Fatalf("explicit mode = %s, want cdp", got)
	}
}

package browser

import (
	"encoding/json"
	"fmt"

	corebrowser "browser-server/internal/browser"
)

// orchestrate.go — browser_execute: step validation and param rendering.

// validateMode enforces the eval execution mode used by browser_execute.
func validateMode(mode string) error {
	switch mode {
	case "", "inject", "cdp":
		return nil
	default:
		return fmt.Errorf("mode must be \"inject\" or \"cdp\"")
	}
}

// validateSteps enforces one atomic browser_execute request by reusing the
// primitive validators against synthetic args built from each step's params.
func validateSteps(steps []step) error {
	if len(steps) == 0 {
		return fmt.Errorf("steps is required for browser_execute")
	}
	for i, s := range steps {
		if s.Action == "" {
			return fmt.Errorf("steps[%d].action is required", i)
		}
		spec, ok := specs[s.Action]
		if !ok {
			return fmt.Errorf("steps[%d].action %q is not a known command action", i, s.Action)
		}
		if spec.readOnly {
			return fmt.Errorf("steps[%d].action %q is not a command action", i, s.Action)
		}
		if s.Action == corebrowser.ActionOrchestrate {
			return fmt.Errorf("steps[%d] cannot nest browser_execute", i)
		}
		if s.Action == corebrowser.ActionEval {
			if err := validateMode(paramString(s.Params, "mode")); err != nil {
				return fmt.Errorf("steps[%d] (eval): %w", i, err)
			}
		}
		if spec.validate == nil {
			continue
		}
		var synth args
		synth.URL = paramString(s.Params, "url")
		synth.Selector = firstRaw(s.Selector, paramRaw(s.Params, "selector"))
		synth.Value = paramString(s.Params, "value")
		synth.Label = paramString(s.Params, "label")
		synth.Text = paramString(s.Params, "text")
		synth.Keys = paramString(s.Params, "keys")
		synth.Expression = paramString(s.Params, "expression")
		synth.Name = paramString(s.Params, "name")
		synth.Source = paramRaw(s.Params, "source")
		synth.To = paramRaw(s.Params, "to")
		synth.StorageType = paramString(s.Params, "type")
		synth.StorageAction = paramString(s.Params, "action")
		synth.Key = paramString(s.Params, "key")
		synth.Raw = paramBool(s.Params, "raw")
		synth.WaitMs = 1 // keep wait-validation out of orchestrate's way
		if err := spec.validate(&synth); err != nil {
			return fmt.Errorf("steps[%d] (%s): %w", i, s.Action, err)
		}
	}
	return nil
}

// orchestrateParams renders browser_execute into one extension-side
// orchestrate command; steps pass through untouched apart from the
// arg-level top-ups.
func orchestrateParams(a *args) map[string]any {
	return puts(map[string]any{},
		"steps", a.Steps,
		"mode", a.Mode,
		"screenshot_after", a.ScreenshotAfter,
		"rollback_url", a.RollbackURL,
	)
}

func paramString(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return ""
	}
	return s
}

func paramBool(raw json.RawMessage, key string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false
	}
	return b
}

func paramRaw(raw json.RawMessage, key string) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m[key]
}

func firstRaw(a, b json.RawMessage) json.RawMessage {
	if len(a) > 0 && string(a) != "null" {
		return a
	}
	return b
}

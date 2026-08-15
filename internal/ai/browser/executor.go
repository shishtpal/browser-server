package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corebrowser "browser-server/internal/browser"
)

// strict validates that a JSON raw message only contains allowed keys, then
// unmarshals it into dst. Browser tool args flow through this on every call.
func strict(raw json.RawMessage, dst any, allowed map[string]bool) error {
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	if fields == nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("unknown argument %q", k)
		}
	}
	return json.Unmarshal(raw, dst)
}

// execute returns the closure shared by every browser tool.
func execute(action string, client corebrowser.Client) func(context.Context, json.RawMessage) (any, error) {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		spec, ok := specs[action]
		if !ok {
			return renderError(fmt.Errorf("unknown action %q", action)), nil
		}
		if spec.readOnly {
			return executeReadOnly(ctx, action, client, raw)
		}

		var a args
		if err := strict(raw, &a, actionFields); err != nil {
			return nil, err
		}

		// Orchestrate carries its own step list rather than single-action
		// validation.
		if action == corebrowser.ActionOrchestrate {
			if err := validateSteps(a.Steps); err != nil {
				return nil, err
			}
			if err := validateMode(a.Mode); err != nil {
				return nil, err
			}
		} else if spec.validate != nil {
			if err := spec.validate(&a); err != nil {
				return nil, err
			}
		}

		if a.TimeoutMS <= 0 {
			a.TimeoutMS = int(corebrowser.DefaultCommandTimeout / time.Millisecond)
		}
		if a.TimeoutMS > int(corebrowser.MaxCommandTimeout/time.Millisecond) {
			a.TimeoutMS = int(corebrowser.MaxCommandTimeout / time.Millisecond)
		}

		cmd, err := client.CreateCommand(ctx, corebrowser.CreateCommandRequest{
			Target:    a.Target,
			Action:    action,
			Params:    paramsJSON(spec, &a),
			TimeoutMS: a.TimeoutMS,
		})
		if err != nil {
			return renderError(err), nil
		}

		timeout := time.Duration(a.TimeoutMS) * time.Millisecond
		finished, err := client.WaitCommand(ctx, cmd.CommandID, timeout)
		if err != nil {
			return renderError(err), nil
		}
		return renderCommand(finished), nil
	}
}

// executeReadOnly serves registry tools that talk to the bus directly
// (list_instances / list_tabs) — they create no command.
func executeReadOnly(ctx context.Context, action string, client corebrowser.Client, raw json.RawMessage) (any, error) {
	var a args
	if err := strict(raw, &a, actionFields); err != nil {
		return nil, err
	}
	switch action {
	case "list_instances":
		instances, err := client.ListInstances(ctx)
		if err != nil {
			return renderError(err), nil
		}
		return map[string]any{"instances": instances, "count": len(instances)}, nil
	case "list_tabs":
		var instanceID string
		if a.Target.Browser != nil {
			var err error
			instanceID, err = resolveBrowserFromTarget(ctx, client, *a.Target.Browser)
			if err != nil {
				return renderError(err), nil
			}
		} else {
			var err error
			instanceID, err = resolveBrowserFromTarget(ctx, client, corebrowser.BrowserRef{})
			if err != nil {
				return renderError(err), nil
			}
		}
		tabs, err := client.ListTabs(ctx, instanceID)
		if err != nil {
			return renderError(err), nil
		}
		return map[string]any{"tabs": tabs, "count": len(tabs)}, nil
	default:
		return renderError(fmt.Errorf("unknown read-only action %q", action)), nil
	}
}

// resolveBrowserFromTarget resolves the target's browser half via the bus /
// HTTP layer. Used by registry-read tools (list_tabs) so that omitting
// browser hits the exact same single-online shortcut / label / first_online
// resolution that mutating commands go through.
func resolveBrowserFromTarget(ctx context.Context, client corebrowser.Client, t corebrowser.BrowserRef) (string, error) {
	instances, err := client.ListInstances(ctx)
	if err != nil {
		return "", err
	}
	online := make([]corebrowser.Instance, 0, len(instances))
	for _, inst := range instances {
		if inst.Online {
			online = append(online, inst)
		}
	}

	switch {
	case t.InstanceID != "":
		for _, inst := range instances {
			if inst.InstanceID == t.InstanceID {
				if !inst.Online {
					return "", fmt.Errorf("browser %q is offline", t.InstanceID)
				}
				return t.InstanceID, nil
			}
		}
		return "", fmt.Errorf("browser %q not found", t.InstanceID)
	case t.Label != "":
		var matches []corebrowser.Instance
		for _, inst := range online {
			if strings.EqualFold(inst.Label, t.Label) {
				matches = append(matches, inst)
			}
		}
		if len(matches) == 1 {
			return matches[0].InstanceID, nil
		}
		if len(matches) == 0 {
			return "", fmt.Errorf("no online browser labeled %q", t.Label)
		}
		return "", fmt.Errorf("multiple online browsers share label %q; set instance_id", t.Label)
	case t.FirstOnline || (t.InstanceID == "" && t.Label == "" && !t.FirstOnline):
		if len(online) == 1 {
			return online[0].InstanceID, nil
		}
		if len(online) == 0 {
			return "", fmt.Errorf("no browser instance is online")
		}
		return "", fmt.Errorf("multiple browsers are online; specify browser.instance_id or browser.label")
	}
	return "", fmt.Errorf("browser target must set instance_id, label, or first_online")
}

// renderCommand renders a finished command for the model.
func renderCommand(cmd corebrowser.Command) map[string]any {
	out := map[string]any{
		"command_id":  cmd.CommandID,
		"action":      cmd.Action,
		"status":      cmd.Status,
		"instance_id": cmd.InstanceID,
		"tab_uuid":    cmd.TabUUID,
		"session_id":  cmd.SessionID,
	}
	if cmd.Error != "" {
		out["error"] = cmd.Error
	}
	if cmd.Result != nil {
		out["result"] = cmd.Result
	}
	return out
}

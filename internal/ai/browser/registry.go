package browser

import (
	"context"
	"encoding/json"

	"browser-server/internal/ai/browser/config"
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// registry.go — read-only tools that bypass the command channel.

type listHandler func(ctx context.Context, a *args) (any, error)

// mkListTool builds a read-only tool whose handler gets parsed args. Its
// availability follows bs-browser-config.json like the command tools.
func mkListTool(name, desc string, params map[string]any, required []string, requireTarget bool, h listHandler) tools.Tool {
	fullName := "browser_" + name
	return tools.Tool{
		Name:        fullName,
		Category:    "Browser",
		Description: desc,
		Schema:      buildSchema(params, required, desc, requireTarget),
		Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a args
			if err := strict(raw, &a, actionFields); err != nil {
				return nil, err
			}
			return h(ctx, &a)
		},
		Available: func() bool {
			return browserconfig.Get().ToolEnabled(fullName)
		},
	}
}

func listInstancesTool(client corebrowser.Client) tools.Tool {
	return mkListTool("list_instances",
		"List online browser profiles (instance_id, label, browser, version). Use before the first browser_* call to discover the target browser. Each browser profile (Chrome, Firefox, Canary, separate --user-data-dir) is one instance.",
		map[string]any{}, nil, false, func(ctx context.Context, _ *args) (any, error) {
			instances, err := client.ListInstances(ctx)
			if err != nil {
				return renderError(err), nil
			}
			return map[string]any{"instances": instances, "count": len(instances)}, nil
		})
}

func listTabsTool(client corebrowser.Client) tools.Tool {
	return mkListTool("list_tabs",
		"List tabs for a browser profile: tab_uuid, url, title, active. Use to find tab.uuid for targeting and to disambiguate ambiguous url/title patterns.",
		map[string]any{}, nil, true, func(ctx context.Context, a *args) (any, error) {
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
		})
}

func statsTool(client corebrowser.Client) tools.Tool {
	return mkListTool("stats",
		"Get the browser automation health snapshot: online instances, tab count, command counters, and a small sample of recent command ids (postmortem debugging).",
		map[string]any{}, nil, false, func(ctx context.Context, _ *args) (any, error) {
			return statsSnapshot(ctx, client)
		})
}

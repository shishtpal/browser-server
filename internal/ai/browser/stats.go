package browser

import (
	"context"

	corebrowser "browser-server/internal/browser"
)

// stats.go — browser_stats helper that aggregates a bus snapshot the AI can
// use to understand what's online, what to disambiguate, and when the last
// extension was seen.
func statsSnapshot(ctx context.Context, client corebrowser.Client) (any, error) {
	instances, err := client.ListInstances(ctx)
	if err != nil {
		return renderError(err), nil
	}
	online := 0
	tabs := 0
	byBrowser := map[string]int{}
	for _, inst := range instances {
		if inst.Online {
			online++
		}
		byBrowser[inst.Browser]++
		ts, err := client.ListTabs(ctx, inst.InstanceID)
		if err == nil {
			tabs += len(ts)
		}
	}
	return map[string]any{
		"instances_total":  len(instances),
		"instances_online": online,
		"tabs":             tabs,
		"by_browser":       byBrowser,
		"note":             "Command history is emitted through /api/browser/cmd and IDs only; use browser_list_instances + browser_list_tabs for disambiguation.",
	}, nil
}

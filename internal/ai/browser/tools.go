package browser

import (
	"browser-server/internal/ai/browser/config"
	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

// mk builds one tool entry wired through the shared executor. Available is
// bound to the tool's name so bs-browser-config.json can disable it; the
// registry re-evaluates the closure when building specs and at execution, so
// a hot reload takes effect on the next provider step.
func mk(client corebrowser.Client, action, desc string, params map[string]any, required []string, requireTarget bool) tools.Tool {
	name := "browser_" + action
	return tools.Tool{
		Name:        name,
		Category:    "Browser",
		Description: desc,
		Schema:      buildSchema(params, required, desc, requireTarget),
		Execute:     execute(action, client),
		Available: func() bool {
			return browserconfig.Get().ToolEnabled(name)
		},
	}
}

// Tools builds the browser automation tool set against a client. Bootstrap
// wires the in-process local client (server) or the HTTP client (CLI), so
// both entry points expose the identical tool surface.
func Tools(client corebrowser.Client) []tools.Tool {
	if client == nil {
		client = corebrowser.NewHTTPClient(corebrowser.ServerURL(), corebrowser.OperatorTokenProvider())
	}
	out := []tools.Tool{
		listInstancesTool(client),
		listTabsTool(client),
		statsTool(client),
	}
	out = append(out, coreTools(client)...)
	out = append(out, elementTools(client)...)
	out = append(out, backgroundTools(client)...)
	out = append(out, workflowTools(client)...)
	return out
}

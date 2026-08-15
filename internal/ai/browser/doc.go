// Package browser provides the AI-facing browser automation tool set. The
// tools drive the user's real browser through the extension command channel
// (Path 1) via a corebrowser.Client: the web chat server passes an in-process
// LocalClient, and bs-ai-chat passes the HTTP client that relays through the
// running server.
//
// The package is split by tool group so individual tools stay small and can
// be reused across entry points:
//
//	schemas.go    — shared target/selector schemas and schema builders.
//	args.go       — argument union plus the strict key set.
//	dispatch.go   — per-action spec table (validation, params, read-only).
//	executor.go   — shared executor (resolve → create → wait → render).
//	error.go      — structured self-correcting error rendering.
//	registry.go   — read-only registry tools (list_instances / list_tabs).
//	core.go       — navigation, interaction, extraction tools.
//	elements.go   — element-state tools (focus/select/check/hover/drag).
//	background.go — background-API tools (cookies/pdf/bring_to_front).
//	orchestrate.go — browser_execute step validation and param rendering.
//	stats.go       — bus snapshot for the browser_stats tool.
//	workflow.go    — workflow tooling (browser_execute).
//	tools.go       — the Tools constructor used by bootstrap.
//
// Per-tool availability is gated by bs-browser-config.json (see
// internal/ai/browser/config). Every tool carries an Available closure backed
// by that config, so the registry can drop disabled tools from the model's
// toolset and reject their execution without touching the registry itself.
package browser

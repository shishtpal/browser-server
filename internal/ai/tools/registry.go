package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
)

// Tool represents a single callable tool in the AI system.
type Tool struct {
	Name        string
	Description string
	Category    string
	Schema      json.RawMessage
	Execute     func(context.Context, json.RawMessage) (any, error)
	// RawContentFunc extracts raw output from an Execute result. When the tool
	// is listed in tools.raw_output (or a request forces raw mode), the
	// registry calls this function instead of JSON-marshaling. Return
	// (bytes, true) to use raw output; (nil, false) to fall back to JSON
	// marshaling. When nil, the tool never produces raw output.
	RawContentFunc func(any) ([]byte, bool)
}

// Registry holds all registered tools and provides lookup/execution.
type Registry struct {
	tools   map[string]Tool
	shell   ShellInfo
	allowed []string
	limits  toolLimits
	paths   config.PathsConfig
}

// Options configures optional subsystems when constructing a Registry.
type Options struct {
	Memory    config.MemoryConfig
	Skills    *skills.Registry
	WebSearch config.WebSearchConfig
	FileTools config.FileToolsConfig
	Tools     config.ToolsConfig
	Allowed   []string
	Paths     config.PathsConfig
	External  []Tool
}

// New creates a Registry with all built-in tools registered.
func New(options ...Options) *Registry {
	r, err := newRegistry(options...)
	if err != nil {
		// New is retained for built-in-only callers. External callers must use
		// NewWithExternal so registration errors cannot be ignored.
		panic(err)
	}
	return r
}

// NewWithExternal creates a Registry and returns validation or collision
// errors from externally discovered tools instead of silently overwriting a
// built-in registration.
func NewWithExternal(options Options) (*Registry, error) {
	return newRegistry(options)
}

func newRegistry(options ...Options) (*Registry, error) {
	shell := DetectShell()
	r := &Registry{tools: map[string]Tool{}, shell: shell, limits: defaultToolLimits()}
	if len(options) > 0 {
		o := options[0]
		r.allowed = append([]string(nil), o.Allowed...)
		r.limits = toolLimits{
			maxOutput:        o.Tools.MaxOutputBytes(),
			gitTimeout:       o.Tools.GitTimeout(),
			gitMaxOutput:     o.Tools.MaxOutputBytes(),
			gitMaxDiffOutput: o.Tools.MaxDiffOutputBytes(),
		}
		r.limits.rawOutput = map[string]bool{}
		for _, name := range o.Tools.RawOutput {
			r.limits.rawOutput[name] = true
		}
		r.paths = o.Paths
	}

	var memory config.MemoryConfig
	var skillsReg *skills.Registry
	if len(options) > 0 {
		memory = options[0].Memory
		skillsReg = options[0].Skills
	}

	// Memory tools (self-registering)
	registerMemoryTools(r, newMemoryStore(memory))

	// Skill tools (self-registering)
	if skillsReg != nil {
		RegisterSkillTools(r, skillsReg)
	}

	// General tools
	registerGetCurrentTime(r)
	registerAskQuestions(r)
	registerSearchTodos(r)
	registerAddTodoRecord(r)
	registerUpdateTodoRecord(r)
	registerSearchPrompts(r)
	registerManageCalendar(r)
	registerManagePrompt(r)
	registerSearchCalendar(r)
	registerSearchBookmarks(r)
	registerSearchHistory(r)
	registerExecuteCommand(r, shell, r.paths)
	registerExecutePython(r, r.paths)
	if len(options) > 0 && options[0].WebSearch.Enabled {
		registerWebTools(r, options[0].WebSearch)
	}

	// File operation tools
	var fileToolsCfg config.FileToolsConfig
	if len(options) > 0 {
		fileToolsCfg = options[0].FileTools
	}
	registerReadFile(r, fileToolsCfg)
	registerWriteFile(r)
	registerEditFile(r)
	registerMultiEdit(r)
	registerListDirectory(r)
	registerDeleteFile(r)
	registerMoveFile(r)
	registerCopyFile(r)
	registerDirectoryTree(r)

	// Code intelligence tools
	registerSearchCode(r)
	registerAnalyzeCode(r)
	registerGetDiagnostics(r, r.paths)

	// Git tools
	registerGitStatus(r, r.paths)
	registerGitDiff(r, r.paths)
	registerGitLog(r, r.paths)
	registerGitBranch(r, r.paths)
	registerGitCheckout(r, r.paths)
	registerGitCommit(r, r.paths)
	registerGitPush(r, r.paths)
	registerGitPull(r, r.paths)
	registerGitMerge(r, r.paths)

	if len(options) > 0 {
		for _, external := range options[0].External {
			if external.Name == "" || external.Execute == nil || len(external.Schema) == 0 {
				return nil, fmt.Errorf("invalid external tool registration %q", external.Name)
			}
			if _, exists := r.tools[external.Name]; exists {
				return nil, fmt.Errorf("external tool %q conflicts with an existing tool", external.Name)
			}
			var schema map[string]any
			if err := json.Unmarshal(external.Schema, &schema); err != nil || schema == nil {
				return nil, fmt.Errorf("external tool %q has an invalid schema", external.Name)
			}
			r.add(external)
		}
	}

	// Tool discovery is registered last so it can search the complete registry.
	registerSearchTool(r)

	return r, nil
}

// add registers a tool in the registry.
func (r *Registry) add(t Tool) { r.tools[t.Name] = t }

// Categories returns a map of tool name → category for all allowed tools.
func (r *Registry) Categories(allowed []string) map[string]string {
	out := make(map[string]string, len(allowed))
	for _, n := range allowed {
		if t, ok := r.tools[n]; ok {
			out[n] = t.Category
		}
	}
	return out
}

// Specs returns tool specifications for allowed tools.
func (r *Registry) Specs(allowed []string) []provider.ToolSpec {
	var out []provider.ToolSpec
	for _, n := range allowed {
		if t, ok := r.tools[n]; ok {
			out = append(out, provider.ToolSpec{Name: t.Name, Description: t.Description, Parameters: t.Schema})
		}
	}
	return out
}

// Execute runs a tool by name with the given JSON arguments.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) ([]byte, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool")
	}
	if t.Execute == nil {
		return nil, fmt.Errorf("tool is not directly executable")
	}
	// Attach this registry's limits to the caller's context so both the tool
	// function and the output checks below see the same values (raw_output
	// allowlist, max_output, git settings). Reading limitsFrom(ctx) without
	// this would silently use the defaults because callers normally pass a
	// plain context.
	ctx = withToolLimits(ctx, r.limits)
	v, err := t.Execute(ctx, args)
	if err != nil {
		return nil, err
	}
	return r.FormatResult(ctx, name, v)
}

// FormatResult serializes a tool result using the registry's raw-output policy.
// Interactive tools can use it after their result is collected outside Execute.
func (r *Registry) FormatResult(ctx context.Context, name string, value any) ([]byte, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool")
	}
	ctx = withToolLimits(ctx, r.limits)

	// Decide raw vs JSON: a per-request override (WithRawOutputOverride) wins;
	// otherwise fall back to the config tools.raw_output allowlist. Forced raw
	// mode only applies to tools that actually provide a RawContentFunc; tools
	// without one always fall through to JSON.
	lim := limitsFrom(ctx)
	useRaw := lim.rawOutput[name]
	if override := rawOverrideFrom(ctx); override != nil {
		useRaw = *override
	}
	outLimit := outputLimitFor(name, lim)
	if useRaw && t.RawContentFunc != nil {
		if raw, ok := t.RawContentFunc(value); ok {
			if len(raw) > outLimit {
				return nil, fmt.Errorf("tool output exceeds limit")
			}
			return raw, nil
		}
	}

	b, err := json.Marshal(value)
	if len(b) > outLimit {
		return nil, fmt.Errorf("tool output exceeds limit")
	}
	return b, err
}

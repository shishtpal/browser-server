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
}

// New creates a Registry with all built-in tools registered.
func New(options ...Options) *Registry {
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

	// Tool discovery is registered last so it can search the complete registry.
	registerSearchTool(r)

	return r
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
	v, err := t.Execute(withToolLimits(ctx, r.limits), args)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if len(b) > r.limits.maxOutput {
		return nil, fmt.Errorf("tool output exceeds limit")
	}
	return b, err
}

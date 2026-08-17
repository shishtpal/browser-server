package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/memory"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
)

// Tool represents a single callable tool in the AI system.
type Tool struct {
	Name        string
	Description string
	Category    string
	// Keywords carry extra search terms for discovery; MCP/external tools are
	// the intended source. search_tool matches them between category and
	// description priority.
	Keywords []string
	Schema   json.RawMessage
	Execute  func(context.Context, json.RawMessage) (any, error)
	// Available is evaluated when definitions are built and again at execution.
	// Hot-swappable external tools use it to disappear while their leaf service
	// is disabled and reappear on the next provider step after a reload.
	Available func() bool
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
	Memory config.MemoryConfig
	// MemoryStore, when non-nil, is the shared memory.Store to expose tools
	// over (the process singleton created in bootstrap). When nil the registry
	// creates its own via memory.New, so tools, the chat persona injector and
	// the admin endpoint always share one instance keyed by resolved root.
	MemoryStore *memory.Store
	Skills      *skills.Registry
	WebSearch   config.WebSearchConfig
	FileTools   config.FileToolsConfig
	Tools       config.ToolsConfig
	Allowed     []string
	Paths       config.PathsConfig
	External    []Tool
	// ConfigPath is the resolved path of bs-ai-config.json; OCR uses it to
	// anchor relative ocr.output_dir values.
	ConfigPath string
	// OCR gates the built-in ocr_image tool (vision OCR + Poppler PDF
	// rasterization). Providers must be set when OCR.Enabled is true.
	OCR            config.OCRConfig
	ExploreProject config.ExploreProjectConfig
	Providers      map[string]config.ProviderConfig
	// OpenRouter carries the editable attribution headers (site_url/app_name)
	// sent to OpenRouter on chat-completions calls; forwarded to the OCR client.
	OpenRouter config.OpenRouterConfig
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

	var memoryCfg config.MemoryConfig
	var skillsReg *skills.Registry
	var memStore *memory.Store
	if len(options) > 0 {
		memoryCfg = options[0].Memory
		memStore = options[0].MemoryStore
		skillsReg = options[0].Skills
	}

	// Memory tools (self-registering). Prefer the shared singleton passed in
	// by bootstrap; otherwise build the process singleton from config so the
	// tools always target the same store as persona injection / admin.
	if memStore == nil {
		memStore = memory.New(memoryCfg)
	}
	registerMemoryTools(r, memStore)

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
	registerSearchQuestions(r)
	registerManageQuestion(r)
	registerSearchCalendar(r)
	registerSearchBookmarks(r)
	registerSearchHistory(r)
	registerExecuteCommand(r, shell, r.paths)
	registerExecutePython(r, r.paths)
	if len(options) > 0 && options[0].WebSearch.Enabled {
		registerWebTools(r, options[0].WebSearch)
	}
	if len(options) > 0 && options[0].OCR.Enabled {
		registerOCRImage(r, options[0].OCR, options[0].ConfigPath, options[0].Providers, options[0].Paths.AdditionalDirs, options[0].OpenRouter)
	}

	// File operation tools
	var fileToolsCfg config.FileToolsConfig
	if len(options) > 0 {
		fileToolsCfg = options[0].FileTools
	}
	registerReadFile(r, fileToolsCfg) // also registers read_files
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

	// explore_project is registered last among the built-ins: its registration
	// inspects the schemas of the read-only tools it drives (search_code,
	// read_file, git tools, ...), so those must already be registered.
	if len(options) > 0 && options[0].ExploreProject.Enabled {
		registerExploreProject(r, options[0].ExploreProject, options[0].ConfigPath, options[0].Providers, options[0].Skills, options[0].OpenRouter)
	}

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
		if tool, ok := r.tools[n]; ok && toolAvailable(tool) {
			out[n] = tool.Category
		}
	}
	return out
}

// Specs returns tool specifications for allowed tools.
func (r *Registry) Specs(allowed []string) []provider.ToolSpec {
	var out []provider.ToolSpec
	for _, n := range allowed {
		if tool, ok := r.tools[n]; ok && toolAvailable(tool) {
			out = append(out, provider.ToolSpec{Name: tool.Name, Description: tool.Description, Parameters: tool.Schema})
		}
	}
	return out
}

func toolAvailable(tool Tool) bool {
	return tool.Available == nil || tool.Available()
}

// Execute runs a tool by name with the given JSON arguments.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) ([]byte, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool")
	}
	if !toolAvailable(t) {
		return nil, fmt.Errorf("tool is unavailable")
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

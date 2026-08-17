package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
)

//go:embed schemas/explore_project.json
var exploreProjectSchema []byte

// exploreReadOnlyTools is the closed set of tools the internal explorer agent
// may call. It is strictly read-only: no write/edit/delete/git-mutating tools
// are reachable, so the explorer can never modify the project it is exploring.
// git_branch is deliberately excluded even though it is a Git tool: its
// create/delete/rename operations mutate the repository, so it has no place in
// a read-only explorer.
var exploreReadOnlyTools = []string{
	"search_code", "analyze_code", "read_file", "read_files",
	"list_directory", "directory_tree",
	"git_status", "git_log", "git_diff",
}

// maxExploreToolResultBytes caps a single tool result fed back to the explorer
// LLM so the agentic loop cannot blow the model context window.
const maxExploreToolResultBytes = 20000

// exploreClientFactory builds the provider client for one explorer call. Default
// wraps provider.New; tests inject fakes so no outbound HTTP happens.
type exploreClientFactory func(typ, baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration, orSiteURL, orAppName string) provider.Client

type exploreProjectTool struct {
	cfg       config.ExploreProjectConfig
	cfgPath   string
	providers map[string]config.ProviderConfig
	skillsReg *skills.Registry
	registry  *Registry
	newClient exploreClientFactory
	orSiteURL string
	orAppName string
	// defaultPrompt is the resolved explorer system prompt (explorer.md body +
	// operational footer), computed once at registration.
	defaultPrompt string
	// allowedKeys maps each read-only tool to the property names its schema
	// permits, so we can strip stray fields the model emits before executing
	// (the underlying tools parse strictly and reject unknown fields).
	allowedKeys map[string]map[string]bool
}

// registerExploreProject wires the explore_project tool. It is only registered
// when explore_project.enabled is true (mirrors ocr_image). The explorer LLM's
// default system prompt is the body of the configured .skills/<skill_name>.md
// (explorer.md), falling back to a direct file read, then to a built-in default.
func registerExploreProject(r *Registry, cfg config.ExploreProjectConfig, cfgPath string, providers map[string]config.ProviderConfig, skillsReg *skills.Registry, orCfg config.OpenRouterConfig) {
	t := &exploreProjectTool{
		cfg:       cfg,
		cfgPath:   cfgPath,
		providers: providers,
		skillsReg: skillsReg,
		registry:  r,
		orSiteURL: orCfg.SiteURL,
		orAppName: orCfg.AppName,
	}
	t.newClient = func(typ, baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration, siteURL, appName string) provider.Client {
		return provider.New(typ, baseURL, apiKey, timeout, retryAttempts, retryDelay, siteURL, appName)
	}
	t.defaultPrompt = t.resolveDefaultPrompt()
	t.allowedKeys = buildAllowedKeys(r)
	r.add(Tool{
		Name:           "explore_project",
		Category:       "Code Intelligence",
		Description:    "Agentic codebase explorer. Given a natural-language query and a project root, it runs an internal read-only 'explorer' LLM that iteratively calls search/read/git tools to locate the relevant files and functions, then returns the synthesized answer as raw text. Use it to answer 'which files implement X', 'trace the flow of Y', or 'where is Z handled'. It never modifies the project.",
		Schema:         json.RawMessage(exploreProjectSchema),
		Execute:        t.execute,
		RawContentFunc: exploreProjectRaw,
	})
}

// resolveDefaultPrompt returns the explorer system prompt: the configured
// skill's body plus an operational footer. Falls back to a built-in default if
// the skill cannot be resolved.
func (t *exploreProjectTool) resolveDefaultPrompt() string {
	skillName := t.cfg.SkillName
	if skillName == "" {
		skillName = "explorer"
	}
	var body string
	if t.skillsReg != nil {
		if s, ok := t.skillsReg.Get(skillName); ok && s.Content != "" {
			body = s.Content
		}
	}
	if body == "" && t.cfgPath != "" {
		if b, err := skills.ReadSkillBody(filepath.Dir(t.cfgPath), skillName); err == nil {
			body = b
		}
	}
	if strings.TrimSpace(body) == "" {
		body = defaultExplorePrompt
	}
	return body + "\n\n" + exploreOperationalFooter(t.cfg)
}

const defaultExplorePrompt = `You are a codebase navigator. Help understand code structure, find implementations, and trace data flow. You operate in read-only mode.

## When Exploring
- Start broad (directory tree) then narrow to specific files.
- Trace imports and dependencies to understand connections.
- Identify architectural patterns and conventions.
- Summarize findings concisely — the user wants understanding, not exhaustive listings.
- Point out relevant documentation files when they exist.`

func exploreOperationalFooter(cfg config.ExploreProjectConfig) string {
	var sb strings.Builder
	sb.WriteString("OPERATIONAL CONSTRAINTS (enforced):\n")
	sb.WriteString("- You are an internal read-only explorer agent. NEVER modify files. Use only the tools listed below.\n")
	sb.WriteString("- Always cite file paths and line numbers in your answer.\n")
	if len(cfg.ExcludedDirs) > 0 {
		sb.WriteString("- Ignore these directory globs when searching: " + strings.Join(cfg.ExcludedDirs, ", ") + ".\n")
	}
	sb.WriteString("- Available tools (exact names): " + strings.Join(exploreReadOnlyTools, ", ") + ".\n")
	sb.WriteString("- When calling search_code, list_directory, or directory_tree, always set path to the project root (or a subdirectory of it).\n")
	sb.WriteString("- analyze_code expects a single file path and returns its symbols/functions.\n")
	sb.WriteString("- Stop and give your final answer once you have enough evidence; do not loop indefinitely.\n")
	return sb.String()
}

// buildAllowedKeys extracts each read-only tool's permitted property names from
// the registry so stray model-emitted fields can be stripped before execution.
// It reads directly from r.tools (not r.Specs) because the explorer's internal
// tools may not be in tools.allowed — they are only exposed to the explorer LLM,
// not the parent agent.
func buildAllowedKeys(r *Registry) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, name := range exploreReadOnlyTools {
		tool, ok := r.tools[name]
		if !ok || len(tool.Schema) == 0 {
			continue
		}
		var sch struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(tool.Schema, &sch); err != nil {
			continue
		}
		keys := map[string]bool{}
		for k := range sch.Properties {
			keys[k] = true
		}
		out[name] = keys
	}
	return out
}

type exploreArgs struct {
	Query         string  `json:"query"`
	ProjectPath   string  `json:"project_path"`
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	SystemPrompt  string  `json:"system_prompt"`
	Temperature   float64 `json:"temperature"`
	MaxIterations int     `json:"max_iterations"`
}

var exploreArgFields = map[string]bool{
	"query": true, "project_path": true, "provider": true, "model": true,
	"system_prompt": true, "temperature": true, "max_iterations": true,
}

func (t *exploreProjectTool) execute(ctx context.Context, raw json.RawMessage) (any, error) {
	var a exploreArgs
	if err := strict(raw, &a, exploreArgFields); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if strings.TrimSpace(a.ProjectPath) == "" {
		return nil, fmt.Errorf("project_path is required")
	}
	root := filepath.Clean(a.ProjectPath)
	if !filepath.IsAbs(root) && t.cfgPath != "" {
		root = filepath.Join(filepath.Dir(t.cfgPath), root)
		root = filepath.Clean(root)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("project_path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project_path %q is not a directory", root)
	}

	provName := a.Provider
	if provName == "" {
		provName = t.cfg.DefaultProvider
	}
	modelID := a.Model
	if modelID == "" {
		modelID = t.cfg.DefaultModel
	}
	if provName == "" || modelID == "" {
		return nil, fmt.Errorf("provider/model could not be resolved (set explore_project.default_provider/default_model or pass provider/model)")
	}
	pcfg, ok := t.providers[provName]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured in the models file", provName)
	}
	model, supportsTools := findModel(pcfg, modelID)
	if !supportsTools {
		return nil, fmt.Errorf("model %q under provider %q has supports_tools=false; the explorer agent requires function calling", modelID, provName)
	}

	temperature := a.Temperature
	if temperature == 0 {
		temperature = t.cfg.Temperature
	}
	if temperature < 0 || temperature > 2 {
		return nil, fmt.Errorf("temperature must be between 0 and 2")
	}
	maxIterations := a.MaxIterations
	if maxIterations == 0 {
		maxIterations = t.cfg.MaxIterations
	}
	// Bound whatever the caller settles on: Registry.Execute does not validate
	// arguments against the JSON schema, so an out-of-range value here would
	// otherwise become a runaway (or zero-length) agent loop.
	if maxIterations < 1 || maxIterations > 50 {
		return nil, fmt.Errorf("max_iterations must be between 1 and 50")
	}
	maxOutputTokens := t.cfg.MaxOutputTokens
	if maxOutputTokens == 0 {
		maxOutputTokens = model.MaxOutputTokens
	}

	timeout := time.Duration(t.cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := t.newClient(pcfg.Type, pcfg.BaseURL, pcfg.APIKey, timeout, pcfg.RetryAttempts, time.Duration(pcfg.RetryDelaySeconds)*time.Second, t.orSiteURL, t.orAppName)

	systemPrompt := a.SystemPrompt
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = t.defaultPrompt
	}
	// Ensure the operational footer always reflects the actual project root.
	systemPrompt = ensureProjectRootInPrompt(systemPrompt, root)

	specs := t.registry.Specs(exploreReadOnlyTools)

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "Exploration query: " + a.Query},
	}

	finalAnswer := ""
	iterations := 0
	for i := 0; i < maxIterations; i++ {
		iterations = i + 1
		resp, err := client.Complete(ctx, provider.ChatRequest{
			Provider:        provName,
			Model:           modelID,
			Messages:        messages,
			Temperature:     temperature,
			MaxOutputTokens: maxOutputTokens,
			Tools:           specs,
		})
		if err != nil {
			return nil, fmt.Errorf("explorer call %s/%s: %w", provName, modelID, err)
		}
		if len(resp.ToolCalls) == 0 {
			finalAnswer = strings.TrimSpace(resp.Content)
			break
		}
		// Append the assistant turn (with its tool calls) so the provider can
		// pair subsequent tool results.
		assistant := provider.Message{Role: "assistant", Content: resp.Content}
		for _, tc := range resp.ToolCalls {
			assistant.ToolCalls = append(assistant.ToolCalls, provider.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
		}
		messages = append(messages, assistant)
		for _, tc := range resp.ToolCalls {
			result := t.execToolCall(ctx, tc.Name, tc.Arguments)
			if len(result) > maxExploreToolResultBytes {
				result = truncateUTF8(result, maxExploreToolResultBytes) + "\n…[truncated]"
			}
			messages = append(messages, provider.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
	}

	if finalAnswer == "" {
		// Reached max iterations without a terminal answer; surface the last
		// assistant content with a clear marker.
		if len(messages) > 0 && messages[len(messages)-1].Role == "assistant" {
			finalAnswer = strings.TrimSpace(messages[len(messages)-1].Content)
		}
		if finalAnswer == "" {
			finalAnswer = "[explorer produced no final answer]"
		}
		finalAnswer += fmt.Sprintf("\n\n[explorer stopped after %d iterations without a terminal answer]", maxIterations)
	}

	return map[string]any{
		"answer":       finalAnswer,
		"provider":     provName,
		"model":        modelID,
		"iterations":   iterations,
		"project_path": root,
	}, nil
}

// execToolCall executes one read-only tool call on behalf of the explorer LLM.
// Only allowlisted tools run; stray fields are stripped so the underlying
// strict parsers accept the arguments. search_code receives the configured
// excluded_dirs.
func (t *exploreProjectTool) execToolCall(ctx context.Context, name, args string) string {
	allowlisted := false
	for _, n := range exploreReadOnlyTools {
		if n == name {
			allowlisted = true
			break
		}
	}
	if !allowlisted {
		return fmt.Sprintf("error: tool %q is not permitted for exploration", name)
	}
	cleaned, err := t.normalizeArgs(name, args)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	res, err := t.registry.Execute(ctx, name, json.RawMessage(cleaned))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(res)
}

// normalizeArgs strips keys not declared by the tool's schema and, for
// search_code, merges the configured excluded_dirs into the exclude list.
func (t *exploreProjectTool) normalizeArgs(name, args string) (string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		// Not an object — pass through verbatim and let the tool report the error.
		return args, nil
	}
	allowed := t.allowedKeys[name]
	if allowed != nil {
		for k := range m {
			if !allowed[k] {
				delete(m, k)
			}
		}
	}
	if name == "search_code" {
		m = mergeExcludedDirs(m, t.cfg.ExcludedDirs)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return args, nil
	}
	return string(b), nil
}

// mergeExcludedDirs adds cfg.ExcludedDirs to the search_code exclude array,
// de-duplicating and preserving any caller-supplied excludes.
func mergeExcludedDirs(m map[string]json.RawMessage, excluded []string) map[string]json.RawMessage {
	if len(excluded) == 0 {
		return m
	}
	var existing []string
	if raw, ok := m["exclude"]; ok {
		_ = json.Unmarshal(raw, &existing)
	}
	seen := map[string]bool{}
	var merged []string
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		merged = append(merged, s)
	}
	for _, e := range existing {
		add(e)
	}
	for _, e := range excluded {
		add(e)
	}
	if len(merged) == 0 {
		return m
	}
	b, _ := json.Marshal(merged)
	m["exclude"] = b
	return m
}

// ensureProjectRootInPrompt makes sure the operational footer pins the actual
// project root, overriding any previously rendered root.
func ensureProjectRootInPrompt(prompt, root string) string {
	const marker = "Project root to explore:"
	if idx := strings.Index(prompt, marker); idx >= 0 {
		start := idx + len(marker)
		end := strings.Index(prompt[start:], "\n")
		if end < 0 {
			end = len(prompt)
		} else {
			end += start
		}
		return prompt[:idx] + marker + " " + root + prompt[end:]
	}
	return prompt + "\n" + marker + " " + root
}

// findModel returns the model config and whether it supports tool calling.
func findModel(p cfgProvider, modelID string) (config.ModelConfig, bool) {
	for _, m := range p.Models {
		if m.ID == modelID {
			return m, m.SupportsTools
		}
	}
	return config.ModelConfig{}, false
}

// cfgProvider is the subset of config.ProviderConfig used by findModel.
type cfgProvider = config.ProviderConfig

// exploreProjectRaw extracts the synthesized answer text for raw-output mode,
// so the parent agent receives the answer verbatim rather than a JSON envelope.
func exploreProjectRaw(v any) ([]byte, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	answer, ok := m["answer"].(string)
	if !ok || answer == "" {
		return nil, false
	}
	return []byte(answer), true
}

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"browser-server/internal/ai/bootstrap"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/db"
	quizconfig "browser-server/internal/quiz/config"
)

// runCLI orchestrates one bs-ai-chat invocation and returns the process exit
// code. It follows the same flow as the HTTP server's Init: bootstrap the
// provider-agnostic runtime (config → profiles → skills → store → MCP →
// chat service), then resolve the prompt, stage images, and drive the real
// conversation pipeline through chat.Service.SubmitStream.
func runCLI(opts options) int {
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// bootstrap.Init logs the same startup lines as the server (profiles, skills,
	// MCP summary, "AI enabled with N provider(s)"). Those are noise for a
	// one-shot CLI run, so they are dropped unless the caller asked for a trace.
	if !opts.verbose || opts.json {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(0)
	}

	rt, err := bootstrap.Init(bootstrap.Options{ConfigPath: opts.config, ReconcilePending: false})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer rt.Close()

	if opts.listModels {
		return listModels(rt)
	}
	if opts.listTools {
		return listTools(rt)
	}
	if !rt.Config.Enabled {
		fmt.Fprintf(os.Stderr, "Error: AI is disabled: no config found at %s (set BS_AI_CONFIG_PATH or use --config)\n", rt.Config.Path)
		return 1
	}

	// Tool gating. A one-shot CLI run has no operator to answer the approval
	// channel, so tools are auto-approved (--yolo) or disabled (--no-tools).
	// Anything else would block until the --timeout deadline.
	toolsEnabled := !opts.noTools && rt.Config.Tools.Enabled
	if !opts.noTools && !rt.Config.Tools.Enabled && (opts.yolo || opts.tools != "") {
		fmt.Fprintln(os.Stderr, "Error: tools are disabled in bs-ai-config.json; remove --yolo/--tools or enable tools")
		return 1
	}
	if toolsEnabled && !opts.yolo {
		fmt.Fprintln(os.Stderr, "Error: tools require --yolo (interactive approval is not yet supported). Use --no-tools to disable them.")
		return 1
	}

	// The domain tools (todos, calendar, bookmarks, history, prompts,
	// questions) read the same SQLite files as the HTTP server through the
	// package-level handles in internal/db. Without this the tool functions
	// dereference a nil *sql.DB and panic.
	if toolsEnabled {
		dataPath := db.GetDataPath()
		db.InitAll(dataPath)
		defer db.CloseAll()

		quizCfg, quizErr := quizconfig.Load()
		if quizErr != nil {
			fmt.Fprintf(os.Stderr, "Error: load quiz config: %v\n", quizErr)
			return 1
		}
		if quizCfg.Enabled {
			db.InitQuizDB(dataPath)
			defer db.CloseQuizDB()
		}
	}

	// Resolve provider/model, preferring --conversation's stored selection when
	// the caller did not override it explicitly.
	providerName, modelID := opts.provider, opts.model
	if opts.conversation != "" {
		conv, _, err := rt.Store.GetConversation(ctx, opts.conversation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: load conversation %s: %v\n", opts.conversation, err)
			return 1
		}
		if providerName == "" {
			providerName = conv.Provider
		}
		if modelID == "" {
			modelID = conv.Model
		}
	}
	providerName, modelID, err = rt.Service.ResolveSelection(providerName, modelID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v (configured providers: %s)\n", err, providerNames(rt.Config))
		return 1
	}
	_, modelCfg, ok := rt.Config.FindModel(providerName, modelID)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown provider/model selection")
		return 1
	}

	// Fail on images before any work so a bad selection never leaves an orphan
	// conversation; SubmitStream checks the same two conditions later.
	if len(opts.images) > 0 {
		if !rt.Config.Chat.Attachments.Enabled {
			fmt.Fprintln(os.Stderr, "Error: image attachments are disabled in bs-ai-config.json")
			return 1
		}
		if !modelCfg.SupportsVision {
			fmt.Fprintf(os.Stderr, "Error: model %q does not support image attachments\n", modelID)
			return 1
		}
	}

	// Prompt resolution: --prompt → positional → piped stdin. --file contents
	// are inlined ahead of the prompt text regardless of the source.
	prompt := opts.prompt
	if strings.TrimSpace(prompt) == "" {
		stdinPrompt, err := readStdinIfPiped()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: read stdin: %v\n", err)
			return 1
		}
		prompt = stdinPrompt
	}
	content, err := buildPrompt(opts.files, prompt, rt.Config.FileTools.MaxReadBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if strings.TrimSpace(content) == "" && len(opts.images) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no prompt provided (use --prompt, a positional argument, or pipe stdin)")
		return 1
	}

	profile := opts.profile
	if profile != "" {
		if _, ok := rt.Profiles.Get(profile); !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown profile %q (available: %s)\n", profile, profileNames(rt.Profiles))
			return 1
		}
	}

	skillNames := splitList(opts.skills)
	for _, name := range skillNames {
		if _, ok := rt.Skills.Get(name); !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown skill %q\n", name)
			return 1
		}
	}

	activeTools := splitList(opts.tools)

	convID := opts.conversation
	if convID == "" {
		conv, err := rt.Store.CreateConversation(ctx, conversationTitle(content), providerName, modelID, profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: create conversation: %v\n", err)
			return 1
		}
		convID = conv.ID
	}

	var attachmentIDs []string
	if len(opts.images) > 0 {
		attachmentIDs, err = stageImages(ctx, rt, convID, opts.images)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	render := newRenderer(opts)
	req := chat.SubmitRequest{
		Content:                   content,
		AttachmentIDs:             attachmentIDs,
		Provider:                  providerName,
		Model:                     modelID,
		ToolsEnabled:              toolsEnabled,
		YOLOMode:                  opts.yolo,
		ActiveTools:               activeTools,
		Skills:                    skillNames,
		IncludeAllToolDefinitions: true,
	}
	resp, err := rt.Service.SubmitStream(ctx, convID, req, render.Emit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	render.Finish(resp, providerName, modelID)
	return 0
}

// listModels prints the configured providers and models (no model call).
func listModels(rt *bootstrap.Runtime) int {
	if !rt.Config.Enabled {
		fmt.Fprintf(os.Stderr, "Error: AI is disabled: no config found at %s\n", rt.Config.Path)
		return 1
	}
	if len(rt.Config.Providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers configured")
		return 1
	}
	for _, name := range sortedProviderNames(rt.Config) {
		marker := ""
		if name == rt.Config.DefaultProvider {
			marker = " (default)"
		}
		fmt.Printf("%s%s\n", name, marker)
		for _, model := range rt.Config.Providers[name].Models {
			def := ""
			if model.Default {
				def = " (default)"
			}
			fmt.Printf("  %s%s\n", model.ID, def)
		}
	}
	return 0
}

// listTools prints the effective tool allowlist including MCP-discovered tools.
func listTools(rt *bootstrap.Runtime) int {
	if !rt.Config.Enabled {
		fmt.Fprintf(os.Stderr, "Error: AI is disabled: no config found at %s\n", rt.Config.Path)
		return 1
	}
	if !rt.Config.Tools.Enabled {
		fmt.Fprintln(os.Stderr, "Error: tools are disabled in bs-ai-config.json")
		return 1
	}
	toolNames := rt.Service.AllowedTools()
	categories := rt.Service.ToolCategories()
	if len(toolNames) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no tools are allowed")
		return 1
	}
	for _, name := range toolNames {
		cat := categories[name]
		if cat == "" {
			cat = "builtin"
		}
		fmt.Printf("%s (%s)\n", name, cat)
	}
	return 0
}

func providerNames(cfg *aiconfig.Config) string {
	return strings.Join(sortedProviderNames(cfg), ", ")
}

func sortedProviderNames(cfg *aiconfig.Config) []string {
	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// conversationTitle derives the persisted conversation title from the prompt:
// whitespace collapsed, capped at 60 chars (the store truncates at 120).
func conversationTitle(prompt string) string {
	title := strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
	if title == "" {
		return "New chat"
	}
	// Slice by runes, not bytes, so titles containing multi-byte characters
	// (emoji, CJK, the UTF-8 BOM in inlined files) never end up with an
	// invalid half-rune at the cut point.
	runes := []rune(title)
	if len(runes) > 60 {
		title = string(runes[:60])
	}
	return title
}

func profileNames(reg *profiles.Registry) string {
	names := make([]string, 0)
	for _, p := range reg.List() {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

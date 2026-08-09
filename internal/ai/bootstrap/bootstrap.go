// Package bootstrap wires the provider-agnostic AI runtime shared by the HTTP
// server and the bs-ai-chat CLI: config → profiles → skills → store → MCP →
// chat service. Server-only concerns (voice, task runner, attachment-cleanup
// goroutine, route registration) live in internal/ai/api on top of this core,
// so a second binary can reuse the exact same wiring without duplicating it.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/images"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/memory"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
)

// Options controls bootstrap behavior.
type Options struct {
	// ConfigPath overrides the config file location (BS_AI_CONFIG_PATH).
	// Empty means the default resolution chain in aiconfig.Load.
	ConfigPath string
	// ReconcilePending controls the store's startup reconciliation of pending
	// messages. The long-running server passes true; a short-lived secondary
	// process like the CLI passes false so it never cancels another process's
	// in-flight turn.
	ReconcilePending bool
}

// Runtime is the fully wired, provider-agnostic AI runtime.
type Runtime struct {
	Config         *aiconfig.Config
	Store          *store.Store
	Service        *chat.Service
	Profiles       *profiles.Registry
	Skills         *skills.Registry
	MCP            *aimcp.Manager
	Memory         *memory.Store
	AttachmentsDir string
	Images         *images.Service
}

// Init loads the AI config and builds the full runtime. When AI is disabled
// (missing config or models file, or "enabled": false), it returns a Runtime
// with Config.Enabled == false and Profiles loaded for reporting; callers must
// check Enabled and exit with a clear message.
func Init(opts Options) (*Runtime, error) {
	if opts.ConfigPath != "" {
		if err := os.Setenv("BS_AI_CONFIG_PATH", opts.ConfigPath); err != nil {
			return nil, fmt.Errorf("set BS_AI_CONFIG_PATH: %w", err)
		}
	}
	cfg, err := aiconfig.Load()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		log.Printf("AI disabled: no config found at %s", cfg.Path)
		// Still load profiles even if AI is disabled so the config endpoint
		// can report them.
		baseDir := filepath.Dir(cfg.Path)
		profileReg, _ := profiles.Load(baseDir)
		return &Runtime{Config: cfg, Profiles: profileReg}, nil
	}
	baseDir := filepath.Dir(cfg.Path)
	profileReg, err := profiles.Load(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}
	if len(profileReg.List()) > 0 {
		log.Printf("AI profiles loaded: %d profile(s) from %s/.profiles/", len(profileReg.List()), baseDir)
	}
	// Load skills
	var skillReg *skills.Registry
	if cfg.Skills.Enabled {
		skillReg, err = skills.Load(baseDir)
		if err != nil {
			return nil, fmt.Errorf("load skills: %w", err)
		}
		if len(skillReg.List()) > 0 {
			log.Printf("AI skills loaded: %d skill(s) from %s/.skills/", len(skillReg.List()), baseDir)
		}
	} else {
		skillReg = &skills.Registry{}
	}

	dbPath := cfg.ResolvePath(cfg.Logging.DBPath)
	st, err := store.OpenWithOptions(dbPath, store.Options{ReconcilePending: opts.ReconcilePending})
	if err != nil {
		return nil, fmt.Errorf("init AI store: %w", err)
	}
	if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
		st.Close()
		return nil, fmt.Errorf("AI retention cleanup: %w", err)
	}
	attachmentsDir := attachments.Dir(cfg.ResolvePath(".data"))
	imageCfg, err := images.Load(baseDir)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("load AI image models: %w", err)
	}
	imageService, err := images.New(imageCfg, cfg.ResolvePath(".data"))
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("initialize AI image service: %w", err)
	}

	var externalTools []tools.Tool
	if imageService != nil {
		externalTools = append(externalTools, tools.Tool{Name: "generate_image", Category: "Images", Description: "Generate or edit an image from a prompt. Optionally pass existing gallery image IDs as source_image_ids.", Schema: json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"},"provider":{"type":"string"},"model":{"type":"string"},"image_size":{"type":"string"},"source_image_ids":{"type":"array","items":{"type":"string"}}},"required":["prompt"],"additionalProperties":false}`), Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Prompt, Provider, Model, ImageSize string
				SourceImageIDs                     []string `json:"source_image_ids"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			sources := make([][]byte, 0, len(a.SourceImageIDs))
			for _, id := range a.SourceImageIDs {
				_, b, err := imageService.Read(ctx, id)
				if err != nil {
					return nil, fmt.Errorf("read source image: %w", err)
				}
				sources = append(sources, b)
			}
			x, err := imageService.Generate(ctx, images.GenerateRequest{Prompt: a.Prompt, Provider: a.Provider, Model: a.Model, ImageSize: a.ImageSize, Sources: sources})
			if err != nil {
				return nil, err
			}
			return map[string]any{"image": x, "url": "/api/ai/images/" + x.ID + "/file"}, nil
		}})
	}
	var mcpManager *aimcp.Manager
	if cfg.Tools.Enabled {
		mcpCfg, loadErr := aimcp.Load(baseDir)
		if loadErr != nil {
			_ = imageService.Close()
			st.Close()
			return nil, fmt.Errorf("load AI MCP config: %w", loadErr)
		}
		mcpManager, err = aimcp.NewManager(context.Background(), mcpCfg)
		if err != nil {
			_ = imageService.Close()
			st.Close()
			return nil, fmt.Errorf("initialize AI MCP servers: %w", err)
		}
		discovered := mcpManager.Tools()
		allowedSet := make(map[string]bool, len(cfg.Tools.Allowed)+len(discovered))
		for _, name := range cfg.Tools.Allowed {
			allowedSet[name] = true
		}
		for _, discoveredTool := range discovered {
			name := discoveredTool.Name
			externalTools = append(externalTools, tools.Tool{
				Name:        name,
				Description: discoveredTool.Description,
				Category:    discoveredTool.Category,
				Schema:      discoveredTool.Schema,
				Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
					return mcpManager.Execute(ctx, name, raw)
				},
			})
			if !allowedSet[name] {
				cfg.Tools.Allowed = append(cfg.Tools.Allowed, name)
				allowedSet[name] = true
			}
		}
		statuses := mcpManager.Statuses()
		connected := 0
		unavailable := 0
		for _, status := range statuses {
			switch status.Status {
			case "connected":
				connected++
			case "unavailable":
				unavailable++
			}
		}
		if mcpManager.Configured() {
			log.Printf("AI MCP: %d configured, %d connected, %d unavailable, %d usable tool(s)", len(statuses), connected, unavailable, len(discovered))
		}
	}
	// Memory graph store (process singleton shared with tools, persona
	// injection and the admin endpoint). Wire the "librarian" synthesizer to a
	// cheap model referenced from bs-ai-models.json when enabled.
	var memStore *memory.Store
	if cfg.Memory.Enabled {
		memStore = memory.New(cfg.Memory)
		if cfg.Memory.Synthesizer.Enabled {
			if pc, ok := cfg.Providers[cfg.Memory.Synthesizer.Provider]; ok {
				client := provider.NewOpenAICompatibleClient(
					pc.BaseURL, pc.APIKey,
					time.Duration(pc.RequestTimeoutSeconds)*time.Second,
					pc.RetryAttempts,
					time.Duration(pc.RetryDelaySeconds)*time.Second,
				)
				model := cfg.Memory.Synthesizer.Model
				memStore.SetCompleter(memory.CompleterFunc(func(ctx context.Context, req memory.CompletionRequest) (memory.CompletionResponse, error) {
					resp, err := client.Complete(ctx, provider.ChatRequest{
						Model:           model,
						Messages:        []provider.Message{{Role: "system", Content: req.System}, {Role: "user", Content: req.User}},
						Temperature:     req.Temperature,
						MaxOutputTokens: req.MaxOutputTokens,
					})
					if err != nil {
						return memory.CompletionResponse{}, err
					}
					return memory.CompletionResponse{Content: resp.Content}, nil
				}))
				log.Printf("AI memory synthesizer: provider=%s model=%s", cfg.Memory.Synthesizer.Provider, cfg.Memory.Synthesizer.Model)
			} else {
				log.Printf("WARN: memory.synthesizer.provider %q not found in models file; synthesis disabled", cfg.Memory.Synthesizer.Provider)
			}
		}
	}

	service, err := chat.NewServiceWithTools(cfg, st, profileReg, skillReg, externalTools)
	if err != nil {
		if mcpManager != nil {
			mcpManager.Close()
		}
		_ = imageService.Close()
		st.Close()
		return nil, fmt.Errorf("initialize AI tools: %w", err)
	}
	log.Printf("AI enabled with %d provider(s) (models: %s); store: %s", len(cfg.Providers), cfg.ModelsPath, dbPath)
	return &Runtime{
		Config:         cfg,
		Store:          st,
		Service:        service,
		Profiles:       profileReg,
		Skills:         skillReg,
		MCP:            mcpManager,
		Memory:         memStore,
		AttachmentsDir: attachmentsDir,
		Images:         imageService,
	}, nil
}

// Close releases the runtime in dependency order: chat service first (so
// in-flight steps unwind through their own cancellation path), then MCP
// sessions, then the store.
func (r *Runtime) Close() error {
	if r == nil || r.Store == nil {
		return nil
	}
	if r.Service != nil {
		r.Service.Close()
	}
	var mcpErr error
	if r.MCP != nil {
		mcpErr = r.MCP.Close()
	}
	var imageErr error
	if r.Images != nil {
		imageErr = r.Images.Close()
	}
	return errors.Join(mcpErr, imageErr, r.Store.Close())
}

// Package bootstrap wires the provider-agnostic AI runtime shared by the HTTP
// server and the bs-ai-chat CLI: config → profiles → skills → store → MCP →
// chat service. Server-only concerns (voice, task runner, attachment-cleanup
// goroutine, route registration) live in internal/ai/api on top of this core,
// so a second binary can reuse the exact same wiring without duplicating it.
package bootstrap

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"browser-server/internal/ai/attachments"
	aibrowser "browser-server/internal/ai/browser"
	browserconfig "browser-server/internal/ai/browser/config"
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
	"browser-server/internal/ai/tts"
	"browser-server/internal/ai/videos"
	"browser-server/internal/browser"
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
	// BrowserBus, when non-nil, is the in-process browser command bus the
	// server owns. Browser tools use a LocalClient against it. When nil (the
	// CLI), browser tools use the HTTP client that relays through the running
	// server on localhost.
	BrowserBus *browser.Bus
}

// ServiceHolder publishes a hot-swappable service while keeping the retired
// instance alive until all requests that acquired it have completed. The
// atomic pointer makes availability checks cheap; the RWMutex supplies the
// lifecycle guarantee needed before closing SQLite-backed services.
type ServiceHolder[T any] struct {
	ptr atomic.Pointer[T]
	mu  sync.RWMutex
}

// NewServiceHolder initializes a holder with service, which may be nil.
func NewServiceHolder[T any](service *T) *ServiceHolder[T] {
	holder := &ServiceHolder[T]{}
	holder.ptr.Store(service)
	return holder
}

// Available reports whether the holder currently contains a service.
func (h *ServiceHolder[T]) Available() bool {
	return h != nil && h.ptr.Load() != nil
}

// Acquire returns a stable service and a release callback. Callers must defer
// release before using a non-nil service so a concurrent swap cannot close it.
func (h *ServiceHolder[T]) Acquire() (*T, func()) {
	if h == nil {
		return nil, func() {}
	}
	h.mu.RLock()
	service := h.ptr.Load()
	if service == nil {
		h.mu.RUnlock()
		return nil, func() {}
	}
	return service, h.mu.RUnlock
}

// Swap installs service and, after all users of the previous instance drain,
// passes that previous instance to closeOld while holding the lifecycle lock.
func (h *ServiceHolder[T]) Swap(service *T, closeOld func(*T) error) error {
	if h == nil {
		return errors.New("service holder is nil")
	}
	h.mu.Lock()
	old := h.ptr.Swap(service)
	var err error
	if old != nil && closeOld != nil {
		err = closeOld(old)
	}
	h.mu.Unlock()
	return err
}

// ServiceHolders are shared by the bootstrap tool closures and HTTP module so
// both paths observe leaf-config reloads on their next request.
type ServiceHolders struct {
	TTS    *ServiceHolder[tts.Service]
	Images *ServiceHolder[images.Service]
	Videos *ServiceHolder[videos.Service]
}

func newServiceHolders() *ServiceHolders {
	return &ServiceHolders{
		TTS:    NewServiceHolder[tts.Service](nil),
		Images: NewServiceHolder[images.Service](nil),
		Videos: NewServiceHolder[videos.Service](nil),
	}
}

// Close drains and closes both leaf services.
func (h *ServiceHolders) Close() error {
	if h == nil {
		return nil
	}
	var imageErr, ttsErr, videoErr error
	if h.Images != nil {
		imageErr = h.Images.Swap(nil, func(service *images.Service) error { return service.Close() })
	}
	if h.TTS != nil {
		ttsErr = h.TTS.Swap(nil, func(service *tts.Service) error { return service.Close() })
	}
	if h.Videos != nil {
		videoErr = h.Videos.Swap(nil, func(service *videos.Service) error { return service.Close() })
	}
	return errors.Join(imageErr, ttsErr, videoErr)
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
	Holders        *ServiceHolders
	// Browser is the in-process browser command bus when this process owns one
	// (the server). Nil in the CLI, which relays through the server's HTTP API.
	Browser *browser.Bus
}

//go:embed generate_image.json
var generateImageSchema []byte

//go:embed text_to_speech.json
var textToSpeechSchema []byte

//go:embed generate_video.json
var generateVideoSchema []byte

// Init loads the AI config and builds the full runtime. When AI is disabled
// (missing config or models file, or "enabled": false), it returns a Runtime
// with Config.Enabled == false and Profiles loaded for reporting; callers must
// check Enabled and exit with a clear message.
func Init(opts Options) (*Runtime, error) {
	holders := newServiceHolders()
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
		return &Runtime{Config: cfg, Profiles: profileReg, Holders: holders}, nil
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
	ttsCfg, err := tts.Load(baseDir)
	if err != nil {
		_ = imageService.Close()
		st.Close()
		return nil, fmt.Errorf("load AI tts models: %w", err)
	}
	ttsService, err := tts.New(ttsCfg, cfg.ResolvePath(".data"))
	if err != nil {
		_ = imageService.Close()
		st.Close()
		return nil, fmt.Errorf("initialize AI tts service: %w", err)
	}
	holders.Images.ptr.Store(imageService)
	holders.TTS.ptr.Store(ttsService)

	videoCfg, err := videos.Load(baseDir)
	if err != nil {
		_ = imageService.Close()
		_ = ttsService.Close()
		st.Close()
		return nil, fmt.Errorf("load AI video models: %w", err)
	}
	videoService, err := videos.New(videoCfg, cfg.ResolvePath(".data"))
	if err != nil {
		// Videos is a leaf feature: one bad config (e.g. a missing API key env
		// var the tool can surface via the AI chat) must not take the rest of
		// the AI runtime down. Leave the disabled holder in place and log.
		log.Printf("AI video service unavailable: %v", err)
		videoService = nil
	}
	holders.Videos.ptr.Store(videoService)

	var externalTools []tools.Tool
	{
		externalTools = append(externalTools, tools.Tool{
			Name:        "generate_image",
			Category:    "Images",
			Description: "Generate or edit an image from a prompt. Optionally pass existing gallery image IDs as source_image_ids.",
			Schema:      json.RawMessage(generateImageSchema),
			Available:   holders.Images.Available,
			Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
				service, release := holders.Images.Acquire()
				if service == nil {
					return nil, errors.New("image generation is disabled")
				}
				defer release()
				var a struct {
					Prompt, Provider, Model, ImageSize string
					SourceImageIDs                     []string `json:"source_image_ids"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return nil, err
				}
				sources := make([][]byte, 0, len(a.SourceImageIDs))
				for _, id := range a.SourceImageIDs {
					_, b, err := service.Read(ctx, id)
					if err != nil {
						return nil, fmt.Errorf("read source image: %w", err)
					}
					sources = append(sources, b)
				}
				x, err := service.Generate(ctx, images.GenerateRequest{Prompt: a.Prompt, Provider: a.Provider, Model: a.Model, ImageSize: a.ImageSize, Sources: sources})
				if err != nil {
					return nil, err
				}
				return map[string]any{"image": x, "url": "/api/ai/images/" + x.ID + "/file"}, nil
			}})
	}
	{
		externalTools = append(externalTools, tools.Tool{
			Name:        "text_to_speech",
			Category:    "Audio",
			Description: "Convert text to speech and save an MP3 in the local gallery. Optionally pass provider, model, and voice; omitted fields use the configured defaults.",
			Schema:      json.RawMessage(textToSpeechSchema),
			Available:   holders.TTS.Available,
			Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
				service, release := holders.TTS.Acquire()
				if service == nil {
					return nil, errors.New("text-to-speech is disabled")
				}
				defer release()
				var a struct {
					Text, Provider, Model, Voice string
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return nil, err
				}
				x, err := service.Generate(ctx, tts.GenerateRequest{Text: a.Text, Provider: a.Provider, Model: a.Model, Voice: a.Voice})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"speech": x,
					"url":    "/api/ai/voices/" + x.ID + "/file",
					"path":   service.FilePath(x.Filename),
				}, nil
			}})
	}
	{
		externalTools = append(externalTools, tools.Tool{
			Name:        "generate_video",
			Category:    "Video",
			Description: "Start an asynchronous video generation from a prompt and return immediately. Generation typically takes minutes (often longer than one chat turn), so this returns a queued gallery record — never wait for completion or claim the video is ready. Valid params keys, types, and defaults depend on the selected provider/model (mirrored by GET /api/ai/videos/config); pass only params that exist in that model's parameter schema. Do not fabricate a playable link from the returned URL until the record status flips to completed.",
			Schema:      json.RawMessage(generateVideoSchema),
			Available:   holders.Videos.Available,
			Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
				service, release := holders.Videos.Acquire()
				if service == nil {
					return nil, errors.New("video generation is disabled")
				}
				defer release()
				var a struct {
					Prompt   string         `json:"prompt"`
					Provider string         `json:"provider"`
					Model    string         `json:"model"`
					Params   map[string]any `json:"params"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return nil, err
				}
				v, err := service.Submit(ctx, videos.GenerateRequest{Prompt: a.Prompt, Provider: a.Provider, Model: a.Model, Params: a.Params})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"video": v,
					"url":   "/api/ai/videos/" + v.ID + "/file",
					"note":  "Queued. Generation is asynchronous and typically takes minutes: the record is queued now and the url above returns 404 until the task completes. Progress can be watched via GET /api/ai/videos (status queued/in_progress, up to the provider request_timeout_seconds). Never present the url to the user as ready; once status flips to completed the file is served at url.",
				}, nil
			}})
	}
	var mcpManager *aimcp.Manager
	if cfg.Tools.Enabled {
		mcpCfg, loadErr := aimcp.Load(baseDir)
		if loadErr != nil {
			_ = ttsService.Close()
			_ = imageService.Close()
			st.Close()
			return nil, fmt.Errorf("load AI MCP config: %w", loadErr)
		}
		mcpManager, err = aimcp.NewManager(context.Background(), mcpCfg)
		if err != nil {
			_ = ttsService.Close()
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
	// Browser automation tools. The server passes an in-process bus; the CLI
	// falls back to the HTTP relay against the running server. Tools are
	// auto-allowed when tools are enabled so both entry points expose them.
	// Per-tool availability is gated by bs-browser-config.json: a missing or
	// disabled config makes every browser tool unavailable through the same
	// Available closures the registry evaluates at build and execution time.
	if browserCfg, loadErr := browserconfig.Load(); loadErr != nil {
		_ = ttsService.Close()
		_ = imageService.Close()
		st.Close()
		return nil, fmt.Errorf("load browser config: %w", loadErr)
	} else if browserCfg.Enabled {
		toolNames := browserconfig.ToolNames()
		enabled := 0
		for _, name := range toolNames {
			if browserCfg.ToolEnabled(name) {
				enabled++
			}
		}
		log.Printf("Browser automation tools: %d of %d enabled (%s)", enabled, len(toolNames), browserCfg.Path)
	} else {
		log.Printf("Browser automation tools disabled by config (%s)", browserCfg.Path)
	}
	if cfg.Tools.Enabled {
		var browserClient browser.Client
		if opts.BrowserBus != nil {
			browserClient = &browser.LocalClient{Bus: opts.BrowserBus}
		} else {
			browserClient = browser.NewHTTPClient(browser.ServerURL(), browser.OperatorTokenProvider())
		}
		browserTools := aibrowser.Tools(browserClient)
		externalTools = append(externalTools, browserTools...)
		allowedSet := make(map[string]bool, len(cfg.Tools.Allowed))
		for _, name := range cfg.Tools.Allowed {
			allowedSet[name] = true
		}
		for _, bt := range browserTools {
			if !allowedSet[bt.Name] {
				cfg.Tools.Allowed = append(cfg.Tools.Allowed, bt.Name)
				allowedSet[bt.Name] = true
			}
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
				client := provider.New(
					pc.Type, pc.BaseURL, pc.APIKey,
					time.Duration(pc.RequestTimeoutSeconds)*time.Second,
					pc.RetryAttempts,
					time.Duration(pc.RetryDelaySeconds)*time.Second,
					cfg.OpenRouter.SiteURL,
					cfg.OpenRouter.AppName,
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
		_ = ttsService.Close()
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
		Holders:        holders,
		Browser:        opts.BrowserBus,
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
	var holdersErr error
	if r.Holders != nil {
		holdersErr = r.Holders.Close()
	}
	return errors.Join(mcpErr, holdersErr, r.Store.Close())
}

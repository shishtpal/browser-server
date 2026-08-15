package api

import (
	"browser-server/internal/ai/bootstrap"
	aibrowser "browser-server/internal/ai/browser"
	browserconfig "browser-server/internal/ai/browser/config"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/memory"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tasks"
	"browser-server/internal/ai/voice"
	"browser-server/internal/browser"
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
)

type Module struct {
	cfg             *aiconfig.Config
	store           *store.Store
	service         *chat.Service
	profiles        *profiles.Registry
	skills          *skills.Registry
	mcp             *aimcp.Manager
	voice           atomic.Pointer[voice.Config]
	tasks           *tasks.Runner
	memory          *memory.Store
	attachmentsDir  string
	holders         *bootstrap.ServiceHolders
	browser         *browser.Bus
	screenshotStore *aibrowser.ScreenshotStore
	pdfStore        *aibrowser.PdfStore
	configDir       string
	configMu        sync.RWMutex
	stop            chan struct{}
	wg              sync.WaitGroup
	closeOnce       sync.Once
	closeErr        error
}

// The Module struct keeps the same field names as the provider-agnostic runtime
// in internal/ai/bootstrap, so handler files throughout this package can keep
// referencing m.cfg / m.store / m.service unchanged.

func Init() (*Module, error) {
	browserBus := browser.New()
	rt, err := bootstrap.Init(bootstrap.Options{ReconcilePending: true, BrowserBus: browserBus})
	if err != nil {
		browserBus.Close()
		return nil, err
	}
	configDir, err := aiconfig.ExecutableDir()
	if err != nil {
		_ = rt.Close()
		return nil, fmt.Errorf("resolve config directory: %w", err)
	}
	module := &Module{
		cfg:            rt.Config,
		store:          rt.Store,
		service:        rt.Service,
		profiles:       rt.Profiles,
		skills:         rt.Skills,
		mcp:            rt.MCP,
		memory:         rt.Memory,
		attachmentsDir: rt.AttachmentsDir,
		holders:        rt.Holders,
		configDir:      configDir,
		browser:        rt.Browser,
	}
	// Browser screenshots are persisted to disk and referenced by URL instead
	// of inlined base64. Wired regardless of the AI enabled state so extension
	// captures are always stored and results stay small.
	module.screenshotStore = aibrowser.NewScreenshotStore(rt.Config.ResolvePath(".data"))
	// Browser PDFs are persisted the same way (the CDP print-to-PDF payload is
	// base64 that would otherwise blow the model output budget).
	module.pdfStore = aibrowser.NewPdfStore(rt.Config.ResolvePath(".data"))
	if browserBus != nil {
		browserBus.SetScreenshotSink(module.screenshotStore.Save)
		browserBus.SetPdfSink(module.pdfStore.Save)
		// Domain-based eval modes and timeout bounds come from
		// bs-browser-config.json. The funcs re-read the live config on every
		// call, so admin Project Settings edits hot-reload without a restart.
		browserBus.SetEvalModeFunc(browserconfig.EvalModeForURL)
		browserBus.SetCommandLimitsFunc(browserconfig.CommandLimits)
	}
	if !rt.Config.Enabled {
		return module, nil
	}
	baseDir := filepath.Dir(rt.Config.Path)
	voiceCfg, err := voice.Load(baseDir)
	if err != nil {
		_ = rt.Close()
		return nil, fmt.Errorf("load AI voice config: %w", err)
	}
	module.voice.Store(voiceCfg)
	module.stop = make(chan struct{})
	// The durable task runner is started after the store and chat service exist,
	// because it resumes checkpoints written by a previous process on startup.
	if rt.Config.Tasks.Enabled {
		module.tasks = tasks.NewRunner(rt.Config.Tasks, rt.Store, tasks.NewChatAgent(rt.Service, rt.Store, rt.Config.Tasks))
		module.tasks.Start()
	}
	// Reclaim abandoned staged uploads immediately at startup and then on a
	// bounded hourly schedule (the retention window is configured in hours).
	module.cleanupExpiredAttachments()
	module.wg.Add(1)
	go func() {
		defer module.wg.Done()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := rt.Store.CleanupRetention(context.Background(), rt.Config.Logging.RetentionDays); err != nil {
					log.Printf("AI retention cleanup failed: %v", err)
				}
				module.cleanupExpiredAttachments()
			case <-module.stop:
				return
			}
		}
	}()
	// Periodic memory maintenance (salience decay, archive, purge, verify,
	// reindex, dedupe) at boot and then on memory.maintenance_interval.
	if module.memory != nil && module.memory.Enabled() && rt.Config.Memory.AutoCleanup {
		module.runMemoryMaintenance(rt.Config.Memory.MaintenanceInterval)
	}
	return module, nil
}

// runMemoryMaintenance runs one Maintain shortly after boot (async, so a
// large fragment set does not delay startup), then periodically.
func (m *Module) runMemoryMaintenance(interval string) {
	run := func() {
		rep, err := m.memory.Maintain(context.Background())
		if err != nil {
			log.Printf("AI memory maintenance failed: %v", err)
			return
		}
		if len(rep.Archived) > 0 || rep.Purged > 0 || len(rep.DedupeHints) > 0 {
			log.Printf("AI memory maintenance: archived=%d purged=%d dedupe_hints=%d", len(rep.Archived), rep.Purged, len(rep.DedupeHints))
		}
	}
	d, err := time.ParseDuration(interval)
	if err != nil || d <= 0 {
		d = 6 * time.Hour
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		// Boot pass: delayed briefly so server startup is not blocked.
		boot := time.NewTimer(30 * time.Second)
		ticker := time.NewTicker(d)
		defer boot.Stop()
		defer ticker.Stop()
		for {
			select {
			case <-boot.C:
				run()
			case <-ticker.C:
				run()
			case <-m.stop:
				return
			}
		}
	}()
}

func (m *Module) CORSEnabled() bool {
	return m != nil && m.cfg != nil && m.cfg.CORSEnabled
}

func (m *Module) Close() error {
	if m == nil {
		return nil
	}
	m.closeOnce.Do(func() {
		if m.stop != nil {
			close(m.stop)
		}
		// Stop the task runner before the chat service so in-flight steps unwind
		// through their own cancellation path and checkpoint state stays coherent.
		if m.tasks != nil {
			m.tasks.Stop()
		}
		if m.browser != nil {
			m.browser.Close()
			m.browser = nil
		}
		if m.service != nil {
			m.service.Close()
		}
		m.wg.Wait()

		var holdersErr, mcpErr, storeErr error
		// Close MCP sessions before leaf holders: a long-running image or TTS
		// request may delay holder draining, while managed restart has a short
		// shutdown budget and must not orphan MCP child processes.
		if m.mcp != nil {
			mcpErr = m.mcp.Close()
		}
		if m.holders != nil {
			holdersErr = m.holders.Close()
		}
		if m.store != nil {
			storeErr = m.store.Close()
		}
		m.closeErr = errors.Join(mcpErr, holdersErr, storeErr)
	})
	return m.closeErr
}

// PrepareRestart closes MCP sessions first so managed shutdown cannot orphan
// child processes, then performs the ordinary idempotent module close. The field
// is nulled after the early close so a deferred Close cannot double-close the
// manager (Close can legitimately run after PrepareRestart). The module is only
// shut down on restart concurrency, so no concurrent Close callers race here.
func (m *Module) PrepareRestart() {
	if m == nil {
		return
	}
	if m.mcp != nil {
		_ = m.mcp.Close()
		m.mcp = nil
	}
	_ = m.Close()
}

// BrowserBus returns the in-process browser command bus (nil when AI is
// disabled). Handlers in cmd/server wire it into internal/handlers.
func (m *Module) BrowserBus() *browser.Bus {
	if m == nil {
		return nil
	}
	return m.browser
}

// ScreenshotDir returns the directory holding saved browser screenshot PNGs
// (empty when the module was never initialized). cmd/server wires it into
// internal/handlers for the serving route.
func (m *Module) ScreenshotDir() string {
	if m == nil || m.screenshotStore == nil {
		return ""
	}
	return m.screenshotStore.Dir()
}

// PdfDir returns the directory holding saved browser PDFs (empty when the
// module was never initialized). cmd/server wires it into internal/handlers for
// the serving route.
func (m *Module) PdfDir() string {
	if m == nil || m.pdfStore == nil {
		return ""
	}
	return m.pdfStore.Dir()
}

// Profiles returns the loaded profile registry for use by other components.
func (m *Module) Profiles() *profiles.Registry {
	if m == nil {
		return nil
	}
	return m.profiles
}

func (m *Module) Register(r *mux.Router) {
	r.HandleFunc("/ai/config", m.Config).Methods("GET")
	// Voice routes stay registered across enabled/disabled reloads. Handlers
	// resolve the current immutable config for each request.
	r.HandleFunc("/ai/voice/config", m.VoiceConfig).Methods("GET")
	r.HandleFunc("/ai/voice/transcribe", m.VoiceTranscribe).Methods("GET")
	r.HandleFunc("/ai/skills", m.requireAI(m.ListSkills)).Methods("GET")
	r.HandleFunc("/ai/skills/{name}", m.requireAI(m.GetSkill)).Methods("GET")
	r.HandleFunc("/ai/logs", m.requireAI(m.Logs)).Methods("GET")
	r.HandleFunc("/ai/monitoring", m.requireAI(m.Monitoring)).Methods("GET")
	r.HandleFunc("/ai/conversations", m.requireAI(m.ListConversations)).Methods("GET")
	r.HandleFunc("/ai/conversations", m.requireAI(m.CreateConversation)).Methods("POST")
	r.HandleFunc("/ai/conversations/archived", m.requireAI(m.ListArchivedConversations)).Methods("GET")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.GetConversation)).Methods("GET")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.UpdateConversation)).Methods("PATCH")
	r.HandleFunc("/ai/conversations/{id}", m.requireAI(m.DeleteConversation)).Methods("DELETE")
	r.HandleFunc("/ai/conversations/{id}/fork", m.requireAI(m.ForkConversation)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/messages", m.requireAI(m.SubmitMessage)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/attachments", m.requireAI(m.UploadAttachment)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/attachments/{attachmentId}", m.requireAI(m.DeleteAttachment)).Methods("DELETE")
	r.HandleFunc("/ai/conversations/{id}/attachments/{attachmentId}", m.requireAI(m.GetAttachment)).Methods("GET")
	r.HandleFunc("/ai/conversations/{id}/attachments/{attachmentId}", m.requireAI(m.RenameAttachment)).Methods("PATCH")
	r.HandleFunc("/ai/attachments", m.requireAI(m.ListAttachments)).Methods("GET")
	r.HandleFunc("/ai/images/config", m.ImageConfig).Methods("GET")
	r.HandleFunc("/ai/images", m.ListImages).Methods("GET")
	r.HandleFunc("/ai/images", m.GenerateImage).Methods("POST")
	r.HandleFunc("/ai/images/{id}", m.DeleteImage).Methods("DELETE")
	r.HandleFunc("/ai/images/{id}/file", m.GetImageFile).Methods("GET")
	r.HandleFunc("/ai/voices/config", m.VoiceGalleryConfig).Methods("GET")
	r.HandleFunc("/ai/voices", m.ListVoices).Methods("GET")
	r.HandleFunc("/ai/voices", m.GenerateSpeech).Methods("POST")
	r.HandleFunc("/ai/voices/{id}", m.DeleteSpeech).Methods("DELETE")
	r.HandleFunc("/ai/voices/{id}/file", m.GetSpeechFile).Methods("GET")
	r.HandleFunc("/ai/conversations/{id}/messages/append", m.requireAI(m.AppendMessage)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/messages/{msgId}", m.requireAI(m.UpdateMessage)).Methods("PATCH")
	r.HandleFunc("/ai/conversations/{id}/messages/{msgId}", m.requireAI(m.DeleteMessage)).Methods("DELETE")
	r.HandleFunc("/ai/conversations/{id}/tool-calls/{callID}", m.requireAI(m.DecideToolCall)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/stop", m.requireAI(m.StopGeneration)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/regenerate", m.requireAI(m.Regenerate)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/archive", m.requireAI(m.ArchiveConversation)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/restore", m.requireAI(m.RestoreConversation)).Methods("POST")
	r.HandleFunc("/ai/tasks", m.requireTasks(m.CreateTask)).Methods("POST")
	r.HandleFunc("/ai/tasks", m.requireTasks(m.ListTasks)).Methods("GET")
	r.HandleFunc("/ai/tasks/status", m.requireAI(m.TaskStatus)).Methods("GET")
	r.HandleFunc("/ai/tasks/{id}", m.requireTasks(m.GetTask)).Methods("GET")
	r.HandleFunc("/ai/tasks/{id}", m.requireTasks(m.DeleteTask)).Methods("DELETE")
	r.HandleFunc("/ai/tasks/{id}/cancel", m.requireTasks(m.CancelTask)).Methods("POST")
	// Memory admin endpoints expose full fragment bodies and accept arbitrary
	// batches (including delete); require the store to be explicitly enabled in
	// addition to requireAI so a half-configured AI stack cannot reach them.
	r.HandleFunc("/ai/memory/stats", m.requireAI(m.requireMemory(m.MemoryStats))).Methods("GET")
	r.HandleFunc("/ai/memory/maintain", m.requireAI(m.requireMemory(m.MemoryMaintain))).Methods("POST")
	r.HandleFunc("/ai/memory/graph", m.requireAI(m.requireMemory(m.MemoryGraph))).Methods("GET")
	r.HandleFunc("/ai/memory/fragments/{id}", m.requireAI(m.requireMemory(m.MemoryFragment))).Methods("GET")
	r.HandleFunc("/ai/memory/write", m.requireAI(m.requireMemory(m.MemoryWrite))).Methods("POST")
}

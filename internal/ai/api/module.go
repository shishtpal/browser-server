package api

import (
	"browser-server/internal/ai/bootstrap"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/images"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/memory"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tasks"
	"browser-server/internal/ai/tts"
	"browser-server/internal/ai/voice"
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Module struct {
	cfg            *aiconfig.Config
	store          *store.Store
	service        *chat.Service
	profiles       *profiles.Registry
	skills         *skills.Registry
	mcp            *aimcp.Manager
	voice          *voice.Config
	tasks          *tasks.Runner
	memory         *memory.Store
	attachmentsDir string
	images         *images.Service
	tts            *tts.Service
	stop           chan struct{}
	wg             sync.WaitGroup
}

// The Module struct keeps the same field names as the provider-agnostic runtime
// in internal/ai/bootstrap, so handler files throughout this package can keep
// referencing m.cfg / m.store / m.service unchanged.

func Init() (*Module, error) {
	rt, err := bootstrap.Init(bootstrap.Options{ReconcilePending: true})
	if err != nil {
		return nil, err
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
		images:         rt.Images,
		tts:            rt.TTS,
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
	module.voice = voiceCfg
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
	if m == nil || m.store == nil {
		return nil
	}
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	// Stop the task runner before the chat service so in-flight steps unwind
	// through their own cancellation path and checkpoint state stays coherent.
	if m.tasks != nil {
		m.tasks.Stop()
	}
	if m.service != nil {
		m.service.Close()
	}
	if m.images != nil {
		_ = m.images.Close()
	}
	if m.tts != nil {
		_ = m.tts.Close()
	}
	m.wg.Wait()
	var mcpErr error
	if m.mcp != nil {
		mcpErr = m.mcp.Close()
	}
	return errors.Join(mcpErr, m.store.Close())
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
	if m.voice != nil {
		r.HandleFunc("/ai/voice/config", m.VoiceConfig).Methods("GET")
		r.Handle("/ai/voice/transcribe", &voice.Proxy{Config: m.voice}).Methods("GET")
	}
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
	r.HandleFunc("/ai/images/config", m.requireImages(m.ImageConfig)).Methods("GET")
	r.HandleFunc("/ai/images", m.requireImages(m.ListImages)).Methods("GET")
	r.HandleFunc("/ai/images", m.requireImages(m.GenerateImage)).Methods("POST")
	r.HandleFunc("/ai/images/{id}", m.requireImages(m.DeleteImage)).Methods("DELETE")
	r.HandleFunc("/ai/images/{id}/file", m.requireImages(m.GetImageFile)).Methods("GET")
	r.HandleFunc("/ai/voices/config", m.requireTTS(m.VoiceGalleryConfig)).Methods("GET")
	r.HandleFunc("/ai/voices", m.requireTTS(m.ListVoices)).Methods("GET")
	r.HandleFunc("/ai/voices", m.requireTTS(m.GenerateSpeech)).Methods("POST")
	r.HandleFunc("/ai/voices/{id}", m.requireTTS(m.DeleteSpeech)).Methods("DELETE")
	r.HandleFunc("/ai/voices/{id}/file", m.requireTTS(m.GetSpeechFile)).Methods("GET")
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

package api

import (
	"browser-server/internal/ai/bootstrap"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tasks"
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
	attachmentsDir string
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
		attachmentsDir: rt.AttachmentsDir,
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
	return module, nil
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
}

package api

import (
	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"browser-server/internal/ai/tools"
	"browser-server/internal/ai/voice"
	"context"
	"encoding/json"
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
	attachmentsDir string
	stop           chan struct{}
	wg             sync.WaitGroup
}

func Init() (*Module, error) {
	cfg, err := aiconfig.Load()
	if err != nil {
		return nil, err
	}
	module := &Module{cfg: cfg}
	if !cfg.Enabled {
		log.Printf("AI disabled: no config found at %s", cfg.Path)
		// Still load profiles even if AI is disabled so the config endpoint can report them
		baseDir := filepath.Dir(cfg.Path)
		module.profiles, _ = profiles.Load(baseDir)
		return module, nil
	}
	baseDir := filepath.Dir(cfg.Path)
	voiceCfg, err := voice.Load(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load AI voice config: %w", err)
	}
	module.voice = voiceCfg
	profileReg, err := profiles.Load(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}
	module.profiles = profileReg
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
	module.skills = skillReg
	dbPath := cfg.ResolvePath(cfg.Logging.DBPath)
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init AI store: %w", err)
	}
	if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
		st.Close()
		return nil, fmt.Errorf("AI retention cleanup: %w", err)
	}
	module.store = st
	module.voice = voiceCfg
	module.attachmentsDir = attachments.Dir(cfg.ResolvePath(".data"))

	var externalTools []tools.Tool
	if cfg.Tools.Enabled {
		mcpCfg, loadErr := aimcp.Load(baseDir)
		if loadErr != nil {
			st.Close()
			return nil, fmt.Errorf("load AI MCP config: %w", loadErr)
		}
		mcpManager, managerErr := aimcp.NewManager(context.Background(), mcpCfg)
		if managerErr != nil {
			st.Close()
			return nil, fmt.Errorf("initialize AI MCP servers: %w", managerErr)
		}
		module.mcp = mcpManager
		discovered := mcpManager.Tools()
		externalTools = make([]tools.Tool, 0, len(discovered))
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
	service, err := chat.NewServiceWithTools(cfg, st, profileReg, skillReg, externalTools)
	if err != nil {
		if module.mcp != nil {
			module.mcp.Close()
		}
		st.Close()
		return nil, fmt.Errorf("initialize AI tools: %w", err)
	}
	module.service = service
	module.stop = make(chan struct{})
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
				if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
					log.Printf("AI retention cleanup failed: %v", err)
				}
				module.cleanupExpiredAttachments()
			case <-module.stop:
				return
			}
		}
	}()
	log.Printf("AI enabled with %d provider(s) (models: %s); store: %s", len(cfg.Providers), cfg.ModelsPath, dbPath)
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
}

package api

import (
	"browser-server/internal/ai/chat"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/profiles"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/store"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Module struct {
	cfg      *aiconfig.Config
	store    *store.Store
	service  *chat.Service
	profiles *profiles.Registry
	skills   *skills.Registry
	stop     chan struct{}
	wg       sync.WaitGroup
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
	module.service = chat.NewService(cfg, st, profileReg, skillReg)
	module.stop = make(chan struct{})
	module.wg.Add(1)
	go func() {
		defer module.wg.Done()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := st.CleanupRetention(context.Background(), cfg.Logging.RetentionDays); err != nil {
					log.Printf("AI retention cleanup failed: %v", err)
				}
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
	m.service.Close()
	m.wg.Wait()
	return m.store.Close()
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
	r.HandleFunc("/ai/conversations/{id}/messages/append", m.requireAI(m.AppendMessage)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/messages/{msgId}", m.requireAI(m.UpdateMessage)).Methods("PATCH")
	r.HandleFunc("/ai/conversations/{id}/messages/{msgId}", m.requireAI(m.DeleteMessage)).Methods("DELETE")
	r.HandleFunc("/ai/conversations/{id}/tool-calls/{callID}", m.requireAI(m.DecideToolCall)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/stop", m.requireAI(m.StopGeneration)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/regenerate", m.requireAI(m.Regenerate)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/archive", m.requireAI(m.ArchiveConversation)).Methods("POST")
	r.HandleFunc("/ai/conversations/{id}/restore", m.requireAI(m.RestoreConversation)).Methods("POST")
}

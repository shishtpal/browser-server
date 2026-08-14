package api

import (
	"encoding/json"
	"net/http"

	"browser-server/internal/ai/voice"
)

func (m *Module) VoiceConfig(w http.ResponseWriter, _ *http.Request) {
	config := m.voice.Load()
	if config == nil {
		config = &voice.Config{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config.Sanitized())
}

func (m *Module) VoiceTranscribe(w http.ResponseWriter, r *http.Request) {
	config := m.voice.Load()
	if config == nil || !config.Enabled {
		writeError(w, http.StatusServiceUnavailable, "voice_disabled", "Voice typing is disabled. Configure bs-ai-voice.json.")
		return
	}
	(&voice.Proxy{Config: config}).ServeHTTP(w, r)
}

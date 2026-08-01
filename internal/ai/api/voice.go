package api

import (
	"encoding/json"
	"net/http"
)

func (m *Module) VoiceConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m.voice.Sanitized())
}

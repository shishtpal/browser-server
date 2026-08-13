package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"browser-server/internal/ai/tts"

	"github.com/gorilla/mux"
)

func (m *Module) requireTTS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.tts == nil {
			writeError(w, http.StatusServiceUnavailable, "tts_disabled", "Text-to-speech is disabled. Configure bs-ai-tts.json and restart the server.")
			return
		}
		next(w, r)
	}
}

func (m *Module) VoiceGalleryConfig(w http.ResponseWriter, r *http.Request) {
	c := m.tts.Config()
	type safeProvider struct {
		Models []tts.Model `json:"models"`
	}
	out := map[string]any{"enabled": true, "default_provider": c.DefaultProvider, "providers": map[string]safeProvider{}}
	p := out["providers"].(map[string]safeProvider)
	for n, v := range c.Providers {
		p[n] = safeProvider{Models: v.Models}
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) ListVoices(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := m.tts.List(r.Context(), limit)
	if err != nil {
		writeError(w, 500, "tts_list_failed", "Unable to list generated speech")
		return
	}
	writeJSON(w, 200, v)
}

func (m *Module) GenerateSpeech(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text     string `json:"text"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Voice    string `json:"voice"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		writeError(w, 400, "invalid_request", "Invalid speech request")
		return
	}
	x, err := m.tts.Generate(r.Context(), tts.GenerateRequest{Text: in.Text, Provider: in.Provider, Model: in.Model, Voice: in.Voice})
	if err != nil {
		if errors.Is(err, tts.ErrProvider) {
			log.Printf("AI speech generation failed: %v", err)
			writeError(w, http.StatusBadGateway, "generation_failed", "The speech provider rejected the request. Check the server log for details.")
			return
		}
		writeError(w, http.StatusBadRequest, "generation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"speech": x, "url": "/api/ai/voices/" + x.ID + "/file"})
}

func (m *Module) GetSpeechFile(w http.ResponseWriter, r *http.Request) {
	x, b, err := m.tts.Read(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, 404, "not_found", "Speech not found")
		return
	}
	w.Header().Set("Content-Type", x.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(x.Filename, `"`, "")+`"`)
	http.ServeContent(w, r, x.Filename, x.CreatedAt, bytes.NewReader(b))
}

func (m *Module) DeleteSpeech(w http.ResponseWriter, r *http.Request) {
	if err := m.tts.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, 404, "not_found", "Speech not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

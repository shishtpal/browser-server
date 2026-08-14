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

func (m *Module) acquireTTS(w http.ResponseWriter) (*tts.Service, func(), bool) {
	if m == nil || m.holders == nil || m.holders.TTS == nil {
		writeError(w, http.StatusServiceUnavailable, "tts_disabled", "Text-to-speech is disabled. Configure bs-ai-tts.json.")
		return nil, nil, false
	}
	service, release := m.holders.TTS.Acquire()
	if service == nil {
		writeError(w, http.StatusServiceUnavailable, "tts_disabled", "Text-to-speech is disabled. Configure bs-ai-tts.json.")
		return nil, nil, false
	}
	return service, release, true
}

func (m *Module) VoiceGalleryConfig(w http.ResponseWriter, _ *http.Request) {
	service, release, ok := m.acquireTTS(w)
	if !ok {
		return
	}
	defer release()
	config := service.Config()
	type safeProvider struct {
		Models []tts.Model `json:"models"`
	}
	out := map[string]any{"enabled": true, "default_provider": config.DefaultProvider, "providers": map[string]safeProvider{}}
	providers := out["providers"].(map[string]safeProvider)
	for name, provider := range config.Providers {
		providers[name] = safeProvider{Models: provider.Models}
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) ListVoices(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireTTS(w)
	if !ok {
		return
	}
	defer release()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	voices, err := service.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tts_list_failed", "Unable to list generated speech")
		return
	}
	writeJSON(w, http.StatusOK, voices)
}

func (m *Module) GenerateSpeech(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireTTS(w)
	if !ok {
		return
	}
	defer release()
	var in struct {
		Text     string `json:"text"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Voice    string `json:"voice"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid speech request")
		return
	}
	speech, err := service.Generate(r.Context(), tts.GenerateRequest{Text: in.Text, Provider: in.Provider, Model: in.Model, Voice: in.Voice})
	if err != nil {
		if errors.Is(err, tts.ErrProvider) {
			log.Printf("AI speech generation failed: %v", err)
			writeError(w, http.StatusBadGateway, "generation_failed", "The speech provider rejected the request. Check the server log for details.")
			return
		}
		writeError(w, http.StatusBadRequest, "generation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"speech": speech, "url": "/api/ai/voices/" + speech.ID + "/file"})
}

func (m *Module) GetSpeechFile(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireTTS(w)
	if !ok {
		return
	}
	defer release()
	speech, data, err := service.Read(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Speech not found")
		return
	}
	w.Header().Set("Content-Type", speech.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(speech.Filename, `"`, "")+`"`)
	http.ServeContent(w, r, speech.Filename, speech.CreatedAt, bytes.NewReader(data))
}

func (m *Module) DeleteSpeech(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireTTS(w)
	if !ok {
		return
	}
	defer release()
	if err := service.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Speech not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

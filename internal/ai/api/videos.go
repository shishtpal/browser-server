package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"browser-server/internal/ai/videos"

	"github.com/gorilla/mux"
)

func (m *Module) acquireVideos(w http.ResponseWriter) (*videos.Service, func(), bool) {
	if m == nil || m.holders == nil || m.holders.Videos == nil {
		writeError(w, http.StatusServiceUnavailable, "videos_disabled", "Video generation is disabled. Configure bs-ai-video-models.json.")
		return nil, nil, false
	}
	service, release := m.holders.Videos.Acquire()
	if service == nil {
		writeError(w, http.StatusServiceUnavailable, "videos_disabled", "Video generation is disabled. Configure bs-ai-video-models.json.")
		return nil, nil, false
	}
	return service, release, true
}

func (m *Module) VideoConfig(w http.ResponseWriter, _ *http.Request) {
	service, release, ok := m.acquireVideos(w)
	if !ok {
		return
	}
	defer release()
	c := service.Config()
	type safeModel struct {
		ID         string             `json:"id"`
		Label      string             `json:"label"`
		Default    bool               `json:"default"`
		Parameters []videos.ParamSpec `json:"parameters"`
	}
	type safeProvider struct {
		Models []safeModel `json:"models"`
	}
	out := map[string]any{"enabled": true, "default_provider": c.DefaultProvider, "providers": map[string]safeProvider{}}
	providers := out["providers"].(map[string]safeProvider)
	for name, provider := range c.Providers {
		models := make([]safeModel, 0, len(provider.Models))
		for _, model := range provider.Models {
			models = append(models, safeModel{ID: model.ID, Label: model.Label, Default: model.Default, Parameters: normalizedSpecs(model.Parameters)})
		}
		providers[name] = safeProvider{Models: models}
	}
	writeJSON(w, http.StatusOK, out)
}

// normalizedSpecs clamps provider numeric bounds against the hard limits the
// server-side constraint validators enforce, so the frontend never advertises
// an out-of-range option that the hard validator would reject (e.g. Agnes
// num_frames has a documented absolute ceiling of 441 even if a config
// declares a larger max).
func normalizedSpecs(specs []videos.ParamSpec) []videos.ParamSpec {
	out := make([]videos.ParamSpec, len(specs))
	copy(out, specs)
	for i := range out {
		if out[i].Key == "num_frames" && (out[i].Max == nil || *out[i].Max > 441) {
			max := 441.0
			out[i].Max = &max
		}
	}
	return out
}

func (m *Module) ListVideos(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireVideos(w)
	if !ok {
		return
	}
	defer release()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := service.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "video_list_failed", "Unable to list videos")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (m *Module) GenerateVideo(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireVideos(w)
	if !ok {
		return
	}
	defer release()
	var in struct {
		Prompt   string         `json:"prompt"`
		Provider string         `json:"provider"`
		Model    string         `json:"model"`
		Params   map[string]any `json:"params"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid video request")
		return
	}
	v, err := service.Submit(r.Context(), videos.GenerateRequest{Prompt: in.Prompt, Provider: in.Provider, Model: in.Model, Params: in.Params})
	if err != nil {
		if errors.Is(err, videos.ErrProvider) {
			log.Printf("AI video generation failed: %v", err)
			writeError(w, http.StatusBadGateway, "generation_failed", "The video provider rejected the request. Check the server log for details.")
			return
		}
		writeError(w, http.StatusBadRequest, "generation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"video": v, "url": "/api/ai/videos/" + v.ID + "/file"})
}

func (m *Module) GetVideoFile(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireVideos(w)
	if !ok {
		return
	}
	defer release()
	video, data, err := service.Read(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Video not found")
		return
	}
	w.Header().Set("Content-Type", video.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(video.Filename, `"`, "")+`"`)
	http.ServeContent(w, r, video.Filename, video.CreatedAt, bytes.NewReader(data))
}

func (m *Module) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireVideos(w)
	if !ok {
		return
	}
	defer release()
	if err := service.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Video not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

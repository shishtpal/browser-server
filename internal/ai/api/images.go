package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"browser-server/internal/ai/images"

	"github.com/gorilla/mux"
)

func (m *Module) acquireImages(w http.ResponseWriter) (*images.Service, func(), bool) {
	if m == nil || m.holders == nil || m.holders.Images == nil {
		writeError(w, http.StatusServiceUnavailable, "images_disabled", "Image generation is disabled. Configure bs-ai-image-models.json.")
		return nil, nil, false
	}
	service, release := m.holders.Images.Acquire()
	if service == nil {
		writeError(w, http.StatusServiceUnavailable, "images_disabled", "Image generation is disabled. Configure bs-ai-image-models.json.")
		return nil, nil, false
	}
	return service, release, true
}

func (m *Module) ImageConfig(w http.ResponseWriter, _ *http.Request) {
	service, release, ok := m.acquireImages(w)
	if !ok {
		return
	}
	defer release()
	c := service.Config()
	type safeProvider struct {
		Models []images.Model `json:"models"`
	}
	out := map[string]any{"enabled": true, "default_provider": c.DefaultProvider, "providers": map[string]safeProvider{}}
	providers := out["providers"].(map[string]safeProvider)
	for name, provider := range c.Providers {
		providers[name] = safeProvider{Models: provider.Models}
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) ListImages(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireImages(w)
	if !ok {
		return
	}
	defer release()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	imagesList, err := service.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "image_list_failed", "Unable to list images")
		return
	}
	writeJSON(w, http.StatusOK, imagesList)
}

func (m *Module) GenerateImage(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireImages(w)
	if !ok {
		return
	}
	defer release()
	var in struct {
		Prompt         string   `json:"prompt"`
		Provider       string   `json:"provider"`
		Model          string   `json:"model"`
		ImageSize      string   `json:"image_size"`
		AspectRatio    string   `json:"aspect_ratio"`
		N              int      `json:"n"`
		Seed           *int     `json:"seed"`
		SourceImageIDs []string `json:"source_image_ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid image request")
		return
	}
	sources := make([][]byte, 0, len(in.SourceImageIDs))
	for _, id := range in.SourceImageIDs {
		_, data, err := service.Read(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "source_not_found", "A source image was not found")
			return
		}
		sources = append(sources, data)
	}
	generated, err := service.GenerateMany(r.Context(), images.GenerateRequest{
		Prompt: in.Prompt, Provider: in.Provider, Model: in.Model, ImageSize: in.ImageSize,
		AspectRatio: in.AspectRatio, N: in.N, Seed: in.Seed, Sources: sources,
	})
	if err != nil {
		if errors.Is(err, images.ErrProvider) {
			log.Printf("AI image generation failed: %v", err)
			writeError(w, http.StatusBadGateway, "generation_failed", "The image provider rejected the request. Check the server log for details.")
			return
		}
		writeError(w, http.StatusBadRequest, "generation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"image": generated[0], "images": generated, "url": "/api/ai/images/" + generated[0].ID + "/file"})
}

func (m *Module) GetImageFile(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireImages(w)
	if !ok {
		return
	}
	defer release()
	image, data, err := service.Read(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Image not found")
		return
	}
	w.Header().Set("Content-Type", image.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(image.Filename, `"`, "")+`"`)
	http.ServeContent(w, r, image.Filename, image.CreatedAt, bytes.NewReader(data))
}

func (m *Module) DeleteImage(w http.ResponseWriter, r *http.Request) {
	service, release, ok := m.acquireImages(w)
	if !ok {
		return
	}
	defer release()
	if err := service.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Image not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

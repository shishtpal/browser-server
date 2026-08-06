package api

import (
	"browser-server/internal/ai/images"
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func (m *Module) requireImages(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.images == nil {
			writeError(w, http.StatusServiceUnavailable, "images_disabled", "Image generation is disabled. Configure bs-ai-image-models.json and restart the server.")
			return
		}
		next(w, r)
	}
}
func (m *Module) ImageConfig(w http.ResponseWriter, r *http.Request) {
	c := m.images.Config()
	type safeProvider struct {
		Models []images.Model `json:"models"`
	}
	out := map[string]any{"enabled": true, "default_provider": c.DefaultProvider, "providers": map[string]safeProvider{}}
	p := out["providers"].(map[string]safeProvider)
	for n, v := range c.Providers {
		p[n] = safeProvider{Models: v.Models}
	}
	writeJSON(w, http.StatusOK, out)
}
func (m *Module) ListImages(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := m.images.List(r.Context(), limit)
	if err != nil {
		writeError(w, 500, "image_list_failed", "Unable to list images")
		return
	}
	writeJSON(w, 200, v)
}
func (m *Module) GenerateImage(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, 400, "invalid_request", "Invalid image request")
		return
	}
	sources := make([][]byte, 0, len(in.SourceImageIDs))
	for _, id := range in.SourceImageIDs {
		_, b, err := m.images.Read(r.Context(), id)
		if err != nil {
			writeError(w, 404, "source_not_found", "A source image was not found")
			return
		}
		sources = append(sources, b)
	}
	xs, err := m.images.GenerateMany(r.Context(), images.GenerateRequest{Prompt: in.Prompt, Provider: in.Provider, Model: in.Model, ImageSize: in.ImageSize, AspectRatio: in.AspectRatio, N: in.N, Seed: in.Seed, Sources: sources})
	if err != nil {
		if errors.Is(err, images.ErrProvider) {
			log.Printf("AI image generation failed: %v", err)
			writeError(w, http.StatusBadGateway, "generation_failed", "The image provider rejected the request. Check the server log for details.")
			return
		}
		writeError(w, http.StatusBadRequest, "generation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"image": xs[0], "images": xs, "url": "/api/ai/images/" + xs[0].ID + "/file"})
}
func (m *Module) GetImageFile(w http.ResponseWriter, r *http.Request) {
	x, b, err := m.images.Read(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, 404, "not_found", "Image not found")
		return
	}
	w.Header().Set("Content-Type", x.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(x.Filename, "\"", "")+`"`)
	http.ServeContent(w, r, x.Filename, x.CreatedAt, bytes.NewReader(b))
}
func (m *Module) DeleteImage(w http.ResponseWriter, r *http.Request) {
	if err := m.images.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, 404, "not_found", "Image not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

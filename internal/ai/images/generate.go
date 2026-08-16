package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"browser-server/internal/ai/attachments"
	"browser-server/internal/ai/openrouter"
	"browser-server/internal/ai/store"
)

// Generate produces a single image.
func (s *Service) Generate(ctx context.Context, r GenerateRequest) (Image, error) {
	images, err := s.GenerateMany(ctx, r)
	if err != nil {
		return Image{}, err
	}
	return images[0], nil
}

// GenerateMany persists every requested image. Providers that only return one
// image per request are called once per requested result to preserve the API's
// n contract without dropping successful outputs.
func (s *Service) GenerateMany(ctx context.Context, r GenerateRequest) ([]Image, error) {
	count := r.N
	if count == 0 {
		count = 1
	}
	if count < 1 || count > 6 {
		return nil, errors.New("n must be between 1 and 6")
	}
	r.N = 1
	result := make([]Image, 0, count)
	for range count {
		x, err := s.generateOne(ctx, r)
		if err != nil {
			return result, err
		}
		result = append(result, x)
	}
	return result, nil
}

func (s *Service) generateOne(ctx context.Context, r GenerateRequest) (Image, error) {
	if strings.TrimSpace(r.Prompt) == "" {
		return Image{}, errors.New("prompt is required")
	}
	pn := r.Provider
	if pn == "" {
		pn = s.cfg.DefaultProvider
	}
	p, ok := s.cfg.Providers[pn]
	if !ok {
		return Image{}, errors.New("unknown image provider")
	}
	m := r.Model
	var mc Model
	if m == "" {
		mc = p.Models[0]
		for _, v := range p.Models {
			if v.Default {
				mc = v
				break
			}
		}
		m = mc.ID
	} else {
		for _, v := range p.Models {
			if v.ID == m {
				mc = v
				break
			}
		}
	}
	if mc.ID == "" {
		return Image{}, errors.New("unknown image model")
	}
	size := r.ImageSize
	if size == "" {
		if len(mc.ImageSizes) > 0 {
			size = mc.ImageSizes[0]
		} else {
			size = "1K"
		}
	}
	if len(mc.ImageSizes) > 0 && !slices.Contains(mc.ImageSizes, size) {
		return Image{}, fmt.Errorf("image size %q is not supported by %s", size, mc.ID)
	}
	if r.AspectRatio != "" && len(mc.AspectRatios) > 0 && !slices.Contains(mc.AspectRatios, r.AspectRatio) {
		return Image{}, fmt.Errorf("aspect ratio %q is not supported by %s", r.AspectRatio, mc.ID)
	}
	if r.Seed != nil && !mc.SupportsSeed {
		return Image{}, errors.New("selected model does not support seed")
	}
	if len(r.Sources) > 0 && !mc.SupportsEditing {
		return Image{}, errors.New("selected model does not support editing")
	}

	var (
		payload map[string]any
		err     error
	)
	switch p.Type {
	case "openrouter_images":
		payload, err = openrouterRequest(mc, r, size)
	case "agnes_images":
		payload, err = agnesRequest(mc, r, size)
	default: // gemini_interactions
		payload, err = geminiRequest(mc, r, size)
	}
	if err != nil {
		return Image{}, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Image{}, err
	}

	u := strings.TrimRight(p.BaseURL, "/") + "/interactions"
	if p.Type == "openrouter_images" {
		u = strings.TrimRight(p.BaseURL, "/") + "/images"
	}
	if p.Type == "agnes_images" {
		u = strings.TrimRight(p.BaseURL, "/") + "/images/generations"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return Image{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Type == "openrouter_images" || p.Type == "agnes_images" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	} else {
		req.Header.Set("x-goog-api-key", p.APIKey)
	}
	openrouter.SetAttributionHeaders(req.Header, p.BaseURL, s.cfg.OpenRouterSiteURL, s.cfg.OpenRouterAppName)
	c := *s.client
	c.Timeout = time.Duration(p.RequestTimeoutSeconds) * time.Second
	resp, err := c.Do(req)
	if err != nil {
		return Image{}, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Image{}, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}

	var (
		data []byte
		ct   string
	)
	switch p.Type {
	case "openrouter_images":
		data, ct, err = extractOpenRouter(body)
	case "agnes_images":
		data, ct, err = s.extractAgnes(ctx, body)
	default: // gemini_interactions
		data, ct, err = extract(body)
	}
	if err != nil {
		return Image{}, fmt.Errorf("%w: %v", ErrProvider, err)
	}

	id := store.NewID("img")
	ext := attachments.ExtFor(ct)
	fn := id + ext
	if err = os.WriteFile(filepath.Join(s.root, fn), data, 0600); err != nil {
		return Image{}, err
	}
	x := Image{ID: id, Prompt: r.Prompt, Provider: pn, Model: m, ImageSize: size, ContentType: ct, Filename: fn, SizeBytes: int64(len(data)), CreatedAt: time.Now().UTC()}
	_, err = s.db.ExecContext(ctx, `INSERT INTO ai_images VALUES(?,?,?,?,?,?,?,?,?)`, x.ID, x.Prompt, x.Provider, x.Model, x.ImageSize, x.ContentType, x.Filename, x.SizeBytes, x.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		_ = os.Remove(filepath.Join(s.root, fn))
		return Image{}, err
	}
	return x, nil
}

// providerMessage pulls the human-readable reason out of a Google API error
// envelope so operators see "API key not valid" instead of a bare 400.
func providerMessage(b []byte) string {
	var v struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &v) == nil && v.Error.Message != "" {
		if v.Error.Status != "" {
			return v.Error.Status + ": " + v.Error.Message
		}
		return v.Error.Message
	}
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		s = s[:300]
	}
	if s == "" {
		return "no response body"
	}
	return s
}

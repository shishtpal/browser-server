package videos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// openrouterVideoProvider implements OpenRouter's asynchronous video API:
// submit to POST {origin}/api/v1/videos, poll GET /api/v1/videos/{id}, and
// download the finished file from GET /api/v1/videos/{id}/content. Unlike
// Agnes, the video bytes are fetched from the provider API with the same
// Authorization header, so the provider also implements contentFetcher.
type openrouterVideoProvider struct{}

// createPayload maps the normalized generation request onto an OpenRouter video
// body. Numeric strings produced by select params (e.g. duration "6") are
// coerced to integers, and frame_images / input_references are wrapped in the
// expected image_url reference shape.
func (openrouterVideoProvider) createPayload(m Model, r GenerateRequest) (map[string]any, error) {
	payload := map[string]any{"model": m.ID}
	if r.Prompt != "" {
		payload["prompt"] = r.Prompt
	}
	for _, key := range []string{"size", "resolution", "aspect_ratio", "callback_url"} {
		if v, ok := r.Params[key]; ok && !isEmpty(v) {
			payload[key] = v
		}
	}
	for _, key := range []string{"duration", "seed"} {
		if v, ok := r.Params[key]; ok && !isEmpty(v) {
			n, ok := toInt(v)
			if !ok {
				return nil, fmt.Errorf("parameter %q must be an integer", key)
			}
			payload[key] = n
		}
	}
	if v, ok := r.Params["generate_audio"]; ok && !isEmpty(v) {
		b, ok := toBool(v)
		if !ok {
			return nil, errors.New("generate_audio must be a boolean")
		}
		payload["generate_audio"] = b
	}
	if imgs := stringSlice(r.Params["frame_images"]); len(imgs) > 0 {
		refs := make([]any, 0, len(imgs))
		for i, u := range imgs {
			refs = append(refs, map[string]any{
				"type":       "image_url",
				"image_url":  map[string]any{"url": u},
				"frame_type": frameTypeFor(i, len(imgs)),
			})
		}
		payload["frame_images"] = refs
	}
	if imgs := stringSlice(r.Params["input_references"]); len(imgs) > 0 {
		refs := make([]any, 0, len(imgs))
		for _, u := range imgs {
			refs = append(refs, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": u},
			})
		}
		payload["input_references"] = refs
	}
	return payload, nil
}

// frameTypeFor assigns the frame role for a URL position: the first image pins
// the first frame and a second image pins the last frame.
func frameTypeFor(i, total int) string {
	if i == total-1 && total > 1 {
		return "last_frame"
	}
	return "first_frame"
}

func (openrouterVideoProvider) Create(ctx context.Context, p Provider, m Model, r GenerateRequest) (string, error) {
	if err := validateOpenRouterConstraints(r.Params); err != nil {
		return "", err
	}
	payload, err := openrouterVideoProvider{}.createPayload(m, r)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	u := strings.TrimRight(p.BaseURL, "/") + "/api/v1/videos"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := &http.Client{Transport: sharedTransport, Timeout: time.Duration(p.RequestTimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("%w: invalid create response", ErrProvider)
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("%w: no job id in create response", ErrProvider)
	}
	return parsed.ID, nil
}

func (openrouterVideoProvider) Poll(ctx context.Context, p Provider, videoID, model string) (pollResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	u := base + "/api/v1/videos/" + url.PathEscape(videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return pollResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	// A status GET should be quick; cap it well below the per-task context
	// budget (p.RequestTimeoutSeconds), which also covers the result download.
	client := &http.Client{Transport: sharedTransport, Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return pollResult{}, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode == http.StatusNotFound {
		return pollResult{Status: StatusFailed, Progress: 0}, fmt.Errorf("%w: job not found", ErrProvider)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return pollResult{Status: StatusQueued}, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	var parsed struct {
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return pollResult{Status: StatusQueued}, fmt.Errorf("%w: invalid poll response", ErrProvider)
	}
	switch statusFromOpenRouter(parsed.Status) {
	case StatusCompleted:
		return pollResult{Status: StatusCompleted, Progress: 100, VideoURL: base + "/api/v1/videos/" + url.PathEscape(videoID) + "/content"}, nil
	case StatusFailed:
		msg := parsed.Error.Message
		if msg == "" {
			msg = "job " + parsed.Status
		}
		return pollResult{Status: StatusFailed, Progress: 0}, fmt.Errorf("%w: %s", ErrProvider, msg)
	case StatusProgress:
		return pollResult{Status: StatusProgress, Progress: 0}, nil
	default:
		return pollResult{Status: StatusQueued, Progress: 0}, nil
	}
}

// statusFromOpenRouter maps OpenRouter's job lifecycle onto the gallery status
// vocabulary. cancelled / expired are permanent terminal failures.
func statusFromOpenRouter(status string) VideoStatus {
	switch status {
	case "completed":
		return StatusCompleted
	case "failed", "cancelled", "expired":
		return StatusFailed
	case "in_progress":
		return StatusProgress
	default: // pending and any unknown future state stay queued
		return StatusQueued
	}
}

// Fetch downloads the completed video from the provider's content endpoint with
// the same Bearer credential used for create/poll.
func (openrouterVideoProvider) Fetch(ctx context.Context, p Provider, res pollResult) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, res.VideoURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := &http.Client{Transport: sharedTransport, Timeout: time.Duration(p.RequestTimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		return nil, "", fmt.Errorf("%w: content status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 500<<20))
	if err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	return data, contentType, nil
}

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
	"strconv"
	"strings"
	"time"
)

// ErrProvider marks failures that originated upstream rather than in the
// caller's request, so handlers can answer 502 instead of 400.
var ErrProvider = errors.New("video provider request failed")

type agnesProvider struct{}

func (agnesProvider) createPayload(m Model, r GenerateRequest) map[string]any {
	payload := map[string]any{"model": m.ID, "prompt": r.Prompt}
	// When keyframe URLs are supplied without an explicit keyframes mode,
	// treat the request as keyframes: Agnes requires the pairing, and leaving
	// mode on ti2vid while passing extra_body.image is ambiguous upstream.
	mode, _ := r.Params["mode"].(string)
	if len(stringSlice(r.Params["extra_body.image"])) > 0 && mode != "keyframes" {
		mode = "keyframes"
	}
	// Agnes understands "ti2vid"/"keyframes" as top-level mode values. Other
	// values (e.g. "image_to_video") are implied by the presence of an image
	// field, so we never forward them as an unsupported top-level mode.
	if mode == "ti2vid" || mode == "keyframes" {
		payload["mode"] = mode
	}
	for _, key := range []string{"width", "height", "num_frames", "frame_rate", "num_inference_steps", "seed", "negative_prompt"} {
		if v, ok := r.Params[key]; ok && !isEmpty(v) {
			payload[key] = v
		}
	}
	// image-to-video: a single public URL at the top level.
	if imgs := stringSlice(r.Params["image"]); len(imgs) > 0 {
		payload["image"] = imgs[0]
	}
	// keyframes: an array of public URLs inside extra_body, with an optional mode.
	keyframes := stringSlice(r.Params["extra_body.image"])
	if len(keyframes) > 0 {
		extra := map[string]any{"image": keyframes}
		if mode != "" {
			extra["mode"] = mode
		}
		payload["extra_body"] = extra
	} else if mode == "keyframes" {
		payload["extra_body"] = map[string]any{"mode": "keyframes"}
	}
	return payload
}

func (agnesProvider) Create(ctx context.Context, p Provider, m Model, r GenerateRequest) (string, error) {
	// Constraints are checked against the model's configured spec (bounds from
	// the config) plus the documented 441 hard ceiling; note this may promote
	// r.Params["mode"] to keyframes in place.
	if err := validateAgnesConstraintsWithSpecs(r.Params, m.Parameters); err != nil {
		return "", err
	}
	payload := agnesProvider{}.createPayload(m, r)
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	u := strings.TrimRight(p.BaseURL, "/") + "/v1/videos"
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
		VideoID string `json:"video_id"`
		TaskID  string `json:"task_id"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("%w: invalid create response", ErrProvider)
	}
	id := parsed.VideoID
	if id == "" {
		id = parsed.TaskID
	}
	if id == "" {
		return "", fmt.Errorf("%w: no video_id in create response", ErrProvider)
	}
	return id, nil
}

func (agnesProvider) Poll(ctx context.Context, p Provider, videoID, model string) (pollResult, error) {
	u := strings.TrimRight(p.BaseURL, "/") + "/agnesapi?video_id=" + url.QueryEscape(videoID)
	// model_name is optional but recommended: it pins result retrieval when the
	// video_id is an upstream ID or the model is not the default.
	if model != "" {
		u += "&model_name=" + url.QueryEscape(model)
	}
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
		return pollResult{Status: StatusFailed, Progress: 0}, fmt.Errorf("%w: task not found", ErrProvider)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return pollResult{Status: StatusQueued}, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, providerMessage(body))
	}
	var parsed struct {
		Status   string          `json:"status"`
		Progress int             `json:"progress"`
		Seconds  json.RawMessage `json:"seconds"`
		Size     string          `json:"size"`
		// Agnes has shipped the completed video URL under different fields:
		// metadata.url (current docs) and, observed in real responses,
		// remixed_from_video_id / video_url / url at the top level. Accept any.
		VideoURL string `json:"remixed_from_video_id"`
		URL      string `json:"url"`
		Metadata struct {
			URL string `json:"url"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return pollResult{Status: StatusQueued}, fmt.Errorf("%w: invalid poll response", ErrProvider)
	}
	seconds, _ := parseSeconds(parsed.Seconds)
	switch parsed.Status {
	case "completed":
		videoURL := parsed.Metadata.URL
		if videoURL == "" {
			videoURL = parsed.VideoURL
		}
		if videoURL == "" {
			videoURL = parsed.URL
		}
		if videoURL == "" {
			return pollResult{Status: StatusFailed, Progress: parsed.Progress}, fmt.Errorf("%w: completed without url", ErrProvider)
		}
		return pollResult{Status: StatusCompleted, Progress: 100, VideoURL: videoURL, Size: parsed.Size, Seconds: seconds}, nil
	case "failed":
		return pollResult{Status: StatusFailed, Progress: parsed.Progress}, fmt.Errorf("%w: task failed", ErrProvider)
	case "in_progress":
		return pollResult{Status: StatusProgress, Progress: parsed.Progress}, nil
	default:
		return pollResult{Status: StatusQueued, Progress: parsed.Progress}, nil
	}
}

// parseSeconds tolerates Agnes returning "seconds" as either a JSON number or a
// string (e.g. "10.0").
func parseSeconds(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return n, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

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

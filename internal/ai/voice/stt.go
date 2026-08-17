package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrProvider marks failures that originated upstream rather than in the
// caller's request, so callers can surface appropriate HTTP status codes.
var ErrProvider = errors.New("stt provider request failed")

// STTRequest describes an audio transcription request.
type STTRequest struct {
	AudioBytes    []byte
	AudioFormat   string // "wav", "mp3", "flac", etc.
	ProviderID    string
	ModelID       string
	Language      string // ISO-639-1 or empty for auto-detect
	Temperature   float64
	ProviderOpts  map[string]any
}

// STTResponse holds the result of a transcription.
type STTResponse struct {
	Text       string         `json:"text"`
	Usage      map[string]any `json:"usage,omitempty"`
	ModelID    string         `json:"model_id,omitempty"`
	ProviderID string         `json:"provider_id,omitempty"`
}

// Transcribe sends audio to an OpenRouter STT provider and returns the
// transcript. The cfg parameter must be non-nil and enabled.
func Transcribe(ctx context.Context, cfg *Config, req STTRequest) (*STTResponse, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, errors.New("voice typing is disabled")
	}
	providerID := req.ProviderID
	if providerID == "" {
		providerID = cfg.DefaultProvider
	}
	p, ok := cfg.Providers[providerID]
	if !ok || !p.Enabled || !p.IsOpenRouterSTT() {
		return nil, errors.New("invalid voice provider")
	}
	modelID := req.ModelID
	if modelID == "" {
		for _, m := range p.Models {
			if m.Default {
				modelID = m.ID
				break
			}
		}
		if modelID == "" && len(p.Models) > 0 {
			modelID = p.Models[0].ID
		}
	}
	if modelID == "" {
		return nil, errors.New("no model specified")
	}
	found := false
	for _, m := range p.Models {
		if m.ID == modelID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown model %q", modelID)
	}
	payload := map[string]any{
		"model": modelID,
		"input_audio": map[string]string{
			"data":   base64.StdEncoding.EncodeToString(req.AudioBytes),
			"format": req.AudioFormat,
		},
	}
	if req.Language != "" && req.Language != "unknown" {
		payload["language"] = OpenRouterLanguageCode(req.Language)
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if len(req.ProviderOpts) > 0 {
		payload["provider"] = map[string]any{"options": req.ProviderOpts}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(p.RequestTimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	u := trimRightSlash(p.BaseURL) + "/audio/transcriptions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("HTTP-Referer", openRouterReferer)
	httpReq.Header.Set("Referer", openRouterReferer)
	httpReq.Header.Set("X-Title", openRouterTitle)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d: %s", ErrProvider, resp.StatusCode, sttErrorMessage(respBody))
	}
	var result struct {
		Text  string         `json:"text"`
		Usage map[string]any `json:"usage,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("%w: invalid response: %v", ErrProvider, err)
	}
	return &STTResponse{Text: result.Text, Usage: result.Usage, ModelID: modelID, ProviderID: providerID}, nil
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func sttErrorMessage(b []byte) string {
	var v struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &v) == nil && v.Error.Message != "" {
		return v.Error.Message
	}
	s := string(b)
	if len(s) > 300 {
		s = s[:300]
	}
	if s == "" {
		return "no response body"
	}
	return s
}



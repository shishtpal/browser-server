package modelrefresh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// geminiProvider implements Provider for Google's Gemini Models List API. It
// returns only chat-capable models (those supporting generateContent) and
// skips embeddings, image generation, and grounding models.
type geminiProvider struct{}

func (geminiProvider) Name() string { return "gemini" }

// geminiModelsURL is the public Gemini Models List endpoint. It stays in sync
// with the default base_url used by the gemini_interactions provider client;
// it is a var (not a const) so tests can point it at a local server.
var geminiModelsURL = "https://generativelanguage.googleapis.com/v1beta/models"

func (geminiProvider) FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, geminiModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// The Gemini API authenticates with the x-goog-api-key header rather than
	// the Authorization: Bearer convention used by OpenAI-compatible catalogs.
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", geminiModelsURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", geminiModelsURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", geminiModelsURL, resp.StatusCode, truncate(string(body), 256))
	}

	var parsed struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			OutputTokenLimit           int      `json:"outputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			SupportedActions           []string `json:"supportedActions"`
			InputModalities            []string `json:"inputModalities"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse JSON from %s: %w", geminiModelsURL, err)
	}

	models := make([]ModelInfo, 0, len(parsed.Models))
	for _, item := range parsed.Models {
		actions := make(map[string]bool, len(item.SupportedActions)+len(item.SupportedGenerationMethods))
		for _, a := range item.SupportedActions {
			actions[a] = true
		}
		for _, m := range item.SupportedGenerationMethods {
			actions[m] = true
		}
		// Only generation models make sense for the chat service.
		isChat := actions["generateContent"] || actions["generateContentStream"]
		if !isChat {
			continue
		}
		id := strings.TrimSpace(item.Name)
		if id == "" {
			continue
		}
		label := item.DisplayName
		if label == "" {
			label = id
		}
		maxTokens := item.OutputTokenLimit
		if maxTokens <= 0 {
			maxTokens = 4096
		}
		// Vision is detected from the catalog when it declares input
		// modalities; otherwise fall back to the model family, because every
		// current gemini-* chat model accepts image input while text-only
		// families (such as gemma) do not.
		supportsVision := false
		for _, modality := range item.InputModalities {
			if modality == "IMAGE" {
				supportsVision = true
				break
			}
		}
		if !supportsVision && strings.HasPrefix(id, "models/gemini") && !strings.Contains(strings.ToLower(id), "tts") {
			supportsVision = true
		}
		// Every chat-capable model in this catalog exposes tool use, so
		// SupportsTools keys off generateContent as well as a dedicated
		// functionCalling action when the catalog advertises one.
		supportsTools := actions["functionCalling"] || isChat
		models = append(models, ModelInfo{
			ID:              id,
			Label:           label,
			SupportsTools:   supportsTools,
			SupportsVision:  supportsVision,
			MaxOutputTokens: maxTokens,
		})
	}
	return models, nil
}

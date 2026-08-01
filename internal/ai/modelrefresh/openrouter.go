package modelrefresh

import (
	"context"
	"strings"
)

// openRouterProvider implements Provider for OpenRouter's public model list.
type openRouterProvider struct{}

func (openRouterProvider) Name() string { return "openrouter.ai" }

const openRouterModelsURL = "https://openrouter.ai/api/v1/models"

func (openRouterProvider) FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	return fetchOpenAICompatibleModels(ctx, openRouterModelsURL, apiKey)
}

// modelsResponse is the OpenAI-compatible /v1/models response shape shared by
// the catalog endpoints used here.
type modelsResponse struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// fetchOpenAICompatibleModels fetches an OpenAI-compatible /v1/models catalog
// and maps it into ModelInfo entries, skipping blank ids.
func fetchOpenAICompatibleModels(ctx context.Context, url, apiKey string) ([]ModelInfo, error) {
	var resp modelsResponse
	if err := fetchJSON(ctx, url, apiKey, &resp); err != nil {
		return nil, err
	}
	models := make([]ModelInfo, 0, len(resp.Data))
	for _, item := range resp.Data {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		label := item.Name
		if label == "" {
			label = item.ID
		}
		models = append(models, ModelInfo{
			ID:              item.ID,
			Label:           label,
			SupportsTools:   true,
			MaxOutputTokens: 4096,
		})
	}
	return models, nil
}

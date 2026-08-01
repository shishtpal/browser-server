package modelrefresh

import "context"

// openCodeProvider implements Provider for OpenCode Zen's model list.
type openCodeProvider struct{}

func (openCodeProvider) Name() string { return "opencode.ai" }

const openCodeModelsURL = "https://opencode.ai/zen/v1/models"

func (openCodeProvider) FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	return fetchOpenAICompatibleModels(ctx, openCodeModelsURL, apiKey)
}

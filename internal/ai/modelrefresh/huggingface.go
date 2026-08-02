package modelrefresh

import "context"

// huggingfaceProvider implements Provider for Hugging Face's inference
// provider router (router.huggingface.co).
type huggingfaceProvider struct{}

func (huggingfaceProvider) Name() string { return "huggingface.co" }

const huggingfaceModelsURL = "https://router.huggingface.co/v1/models"

func (huggingfaceProvider) FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	return fetchOpenAICompatibleModels(ctx, huggingfaceModelsURL, apiKey)
}

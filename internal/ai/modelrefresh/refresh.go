package modelrefresh

import (
	"context"
	"fmt"

	"browser-server/internal/ai/config"
)

// Result reports the outcome of a refresh for one provider.
type Result struct {
	// Fetched is the number of models the provider catalog returned.
	Fetched int
	// Existing is the number of models already configured before the merge.
	Existing int
	// Added is the number of models appended by the merge.
	Added int
	// Models is the merged model list ready to be persisted.
	Models []config.ModelConfig
}

// Refresh fetches the provider's model catalog and merges it into the
// provider's existing model list. cfg.APIKey must already be resolved (env
// references expanded by the caller); it is only used for the HTTP fetch and
// is never written back by callers that persist Result.Models.
func Refresh(ctx context.Context, providerName string, cfg config.ProviderConfig) (Result, error) {
	p, ok := GetProvider(providerName)
	if !ok {
		return Result{}, unknownProviderError(providerName)
	}
	fetched, err := p.FetchModels(ctx, cfg.APIKey)
	if err != nil {
		return Result{}, fmt.Errorf("fetch %s models: %w", providerName, err)
	}
	merged := Merge(cfg.Models, fetched)
	return Result{
		Fetched:  len(fetched),
		Existing: len(cfg.Models),
		Added:    len(merged) - len(cfg.Models),
		Models:   merged,
	}, nil
}

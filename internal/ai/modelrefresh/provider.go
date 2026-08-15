// Package modelrefresh fetches provider model catalogs and merges them into
// the server's bs-ai-models.json, preserving existing entries and appending
// only new models. It only reads and writes JSON; it has no SQLite/DB
// dependencies.
package modelrefresh

import (
	"context"
	"fmt"
	"sort"
)

// ModelInfo describes a model advertised by a provider's catalog endpoint.
type ModelInfo struct {
	ID              string
	Label           string
	SupportsTools   bool
	SupportsVision  bool
	MaxOutputTokens int
}

// Provider fetches a provider's public model list. Implementations must be
// registered via Register so the CLI core stays provider-agnostic.
type Provider interface {
	Name() string
	FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error)
}

// registry holds the registered provider implementations.
var registry = map[string]Provider{}

// Register adds a provider implementation to the registry. Registering the
// same name twice replaces the previous implementation.
func Register(p Provider) {
	registry[p.Name()] = p
}

// GetProvider returns the registered provider with the given name.
func GetProvider(name string) (Provider, bool) {
	p, ok := registry[name]
	return p, ok
}

// ProviderNames returns the names of all registered providers, sorted.
func ProviderNames() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	Register(openRouterProvider{})
	Register(openCodeProvider{})
	Register(huggingfaceProvider{})
	Register(geminiProvider{})
}

func unknownProviderError(name string) error {
	return fmt.Errorf("unknown provider %q (registered: %v)", name, ProviderNames())
}

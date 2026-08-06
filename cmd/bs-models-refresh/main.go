// Command bs-models-refresh refreshes the model catalog in bs-ai-models.json
// from a supported provider (openrouter.ai, opencode.ai, huggingface.co), preserving existing
// entries and appending only new models.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/modelrefresh"
)

const defaultModelsFile = "bs-ai-models.json"

// modelsFileDoc mirrors the top-level shape of bs-ai-models.json.
type modelsFileDoc struct {
	Providers map[string]config.ProviderConfig `json:"providers"`
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		providerName string
		configPath   string
		modelsPath   string
	)
	flag.StringVar(&providerName, "provider", "", "provider name to refresh (openrouter.ai, opencode.ai, huggingface.co)")
	flag.StringVar(&configPath, "config", "", "path to bs-ai-config.json")
	flag.StringVar(&modelsPath, "models", "", "path to bs-ai-models.json")
	flag.Usage = usage
	flag.Parse()

	if strings.TrimSpace(providerName) == "" {
		fmt.Fprintln(os.Stderr, "Error: --provider is required")
		usage()
		return 1
	}

	modelsPath, err := resolveModelsPath(modelsPath, configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	doc, err := loadModelsFile(modelsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	provider, ok := doc.Providers[providerName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: provider %q is not configured in %s (configured: %s)\n",
			providerName, modelsPath, providerNames(doc.Providers))
		return 1
	}

	// Resolve env:VAR references for the fetch, but keep the original value so
	// the file is written back without ever leaking a plaintext key.
	resolvedKey, err := resolveAPIKey(providerName, provider.APIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	originalKey := provider.APIKey
	provider.APIKey = resolvedKey

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := modelrefresh.Refresh(ctx, providerName, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	provider.APIKey = originalKey
	provider.Models = result.Models
	doc.Providers[providerName] = provider

	if err := writeModelsFile(modelsPath, doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Refreshed %q: fetched=%d existing=%d added=%d output=%s\n",
		providerName, result.Fetched, result.Existing, result.Added, modelsPath)
	return 0
}

// resolveModelsPath resolves the models file path using, in order:
//  1. --models flag
//  2. BS_AI_MODELS_PATH env var
//  3. --config flag → sibling bs-ai-models.json
//  4. BS_AI_CONFIG_PATH env var → sibling bs-ai-models.json
//  5. bs-ai-models.json in the current working directory
func resolveModelsPath(modelsFlag, configFlag string) (string, error) {
	if modelsFlag != "" {
		return modelsFlag, nil
	}
	if env := os.Getenv("BS_AI_MODELS_PATH"); env != "" {
		return env, nil
	}
	configPath := configFlag
	if configPath == "" {
		configPath = os.Getenv("BS_AI_CONFIG_PATH")
	}
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), defaultModelsFile), nil
	}
	exeDir, err := config.ExecutableDir()
	if err != nil {
		return "", fmt.Errorf("resolve executable directory: %w", err)
	}
	return filepath.Join(exeDir, defaultModelsFile), nil
}

func loadModelsFile(path string) (*modelsFileDoc, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read models file %s: %w", path, err)
	}
	doc := &modelsFileDoc{}
	if err := json.Unmarshal(content, doc); err != nil {
		return nil, fmt.Errorf("parse models file %s: %w", path, err)
	}
	if doc.Providers == nil {
		doc.Providers = map[string]config.ProviderConfig{}
	}
	return doc, nil
}

func writeModelsFile(path string, doc *modelsFileDoc) error {
	content, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return fmt.Errorf("encode models file: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write models file %s: %w", path, err)
	}
	return nil
}

// resolveAPIKey expands an env:VAR api_key reference. Non-env values are
// returned as-is (empty values are rejected).
func resolveAPIKey(providerName, value string) (string, error) {
	if !strings.HasPrefix(value, "env:") {
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("provider %q api_key is empty", providerName)
		}
		return value, nil
	}
	envName := strings.TrimSpace(strings.TrimPrefix(value, "env:"))
	if envName == "" {
		return "", fmt.Errorf("provider %q api_key env reference is empty", providerName)
	}
	resolved := os.Getenv(envName)
	if resolved == "" {
		return "", fmt.Errorf("provider %q api_key references unset environment variable %q", providerName, envName)
	}
	return resolved, nil
}

func providerNames(providers map[string]config.ProviderConfig) string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: bs-models-refresh --provider <name> [options]

Refreshes the model catalog for a provider in bs-ai-models.json, preserving
existing entries and appending only new models.

Options:
  --provider <name>   Provider to refresh (required): %s
  --config <path>     Path to bs-ai-config.json (optional)
  --models <path>     Path to bs-ai-models.json (optional)

Path resolution for the models file (first match wins):
  1. --models flag
  2. BS_AI_MODELS_PATH environment variable
  3. --config flag, sibling bs-ai-models.json
  4. BS_AI_CONFIG_PATH environment variable, sibling bs-ai-models.json
  5. bs-ai-models.json next to the compiled binary
`, strings.Join(modelrefresh.ProviderNames(), ", "))
}

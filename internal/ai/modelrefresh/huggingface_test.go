package modelrefresh

import (
	"slices"
	"testing"
)

func TestHuggingfaceProviderRegistered(t *testing.T) {
	if _, ok := GetProvider("huggingface.co"); !ok {
		t.Fatal("huggingface.co provider is not registered")
	}
	if !slices.Contains(ProviderNames(), "huggingface.co") {
		t.Fatalf("ProviderNames() = %v, want huggingface.co included", ProviderNames())
	}
}

func TestHuggingfaceModelsURL(t *testing.T) {
	if huggingfaceModelsURL != "https://router.huggingface.co/v1/models" {
		t.Fatalf("huggingfaceModelsURL = %q", huggingfaceModelsURL)
	}
}

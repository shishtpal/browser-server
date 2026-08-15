package modelrefresh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestGeminiProviderRegistered(t *testing.T) {
	if _, ok := GetProvider("gemini"); !ok {
		t.Fatal("gemini provider is not registered")
	}
	if !slices.Contains(ProviderNames(), "gemini") {
		t.Fatalf("ProviderNames() = %v, want gemini included", ProviderNames())
	}
}

func TestGeminiModelsURL(t *testing.T) {
	if geminiModelsURL != "https://generativelanguage.googleapis.com/v1beta/models" {
		t.Fatalf("geminiModelsURL = %q", geminiModelsURL)
	}
}

func TestGeminiFetchModelsFiltersToChatModels(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "secret" {
			t.Errorf("x-goog-api-key = %q", r.Header.Get("x-goog-api-key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{"name": "models/gemini-3-flash", "displayName": "Gemini 3 Flash", "outputTokenLimit": 8192, "supportedActions": ["generateContent", "functionCalling"], "inputModalities": ["TEXT", "IMAGE"]},
				{"name": "models/gemini-3-pro", "displayName": "Gemini 3 Pro", "outputTokenLimit": 65536, "supportedGenerationMethods": ["generateContent", "generateContentStream"]},
				{"name": "models/text-embedding-005", "displayName": "Text Embedding", "supportedGenerationMethods": ["embedContent"]},
				{"name": "models/imagen-4.0", "displayName": "Imagen 4.0", "supportedGenerationMethods": ["imagen"]},
				{"name": "models/gemma-4-31b-it", "displayName": "Gemma 4 31B IT", "outputTokenLimit": 8192, "supportedGenerationMethods": ["generateContent"]}
			]
		}`))
	}))
	defer s.Close()

	originalURL := geminiModelsURL
	geminiModelsURL = s.URL
	t.Cleanup(func() { geminiModelsURL = originalURL })

	models, err := geminiProvider{}.FetchModels(context.Background(), "secret")
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("models = %+v, want the three chat-capable models only", models)
	}
	// models/gemini-3-flash: explicit IMAGE input modality + functionCalling.
	if models[0].ID != "models/gemini-3-flash" || models[0].Label != "Gemini 3 Flash" || models[0].MaxOutputTokens != 8192 || !models[0].SupportsTools || !models[0].SupportsVision {
		t.Errorf("models[0] = %+v", models[0])
	}
	// models/gemini-3-pro: no declared modalities, but the gemini family
	// fallback still marks it vision-capable.
	if models[1].ID != "models/gemini-3-pro" || models[1].MaxOutputTokens != 65536 || !models[1].SupportsTools || !models[1].SupportsVision {
		t.Errorf("models[1] = %+v", models[1])
	}
	// models/gemma-4-31b-it: chat-capable and tool-capable, but text-only.
	if models[2].ID != "models/gemma-4-31b-it" || !models[2].SupportsTools || models[2].SupportsVision {
		t.Errorf("models[2] = %+v", models[2])
	}
}

func TestGeminiFetchModelsHTTPError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "API key not valid", http.StatusForbidden)
	}))
	defer s.Close()

	originalURL := geminiModelsURL
	geminiModelsURL = s.URL
	t.Cleanup(func() { geminiModelsURL = originalURL })

	if _, err := (geminiProvider{}).FetchModels(context.Background(), "bad"); err == nil {
		t.Fatal("expected error for HTTP 403")
	}
}

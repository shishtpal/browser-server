package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func schemaObj(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := configSchema(name)
	if err != nil {
		t.Fatalf("configSchema(%q): %v", name, err)
	}
	if strings.TrimSpace(string(raw)) == "null" {
		t.Fatalf("configSchema(%q) returned null", name)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("configSchema(%q) produced invalid JSON: %v", name, err)
	}
	return root
}

func propNode(t *testing.T, node map[string]any, key string) map[string]any {
	t.Helper()
	props, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("node has no properties")
	}
	child, ok := props[key].(map[string]any)
	if !ok {
		t.Fatalf("property %q missing or not an object", key)
	}
	return child
}

func additionalNode(t *testing.T, node map[string]any) map[string]any {
	t.Helper()
	ap, ok := node["additionalProperties"].(map[string]any)
	if !ok {
		t.Fatalf("node has no object additionalProperties")
	}
	return ap
}

func enumOf(t *testing.T, node map[string]any) []any {
	t.Helper()
	e, ok := node["enum"].([]any)
	if !ok {
		t.Fatalf("node has no enum")
	}
	return e
}

func enumContains(vals []any, want string) bool {
	for _, v := range vals {
		if v == want {
			return true
		}
	}
	return false
}

func TestConfigSchemaModelsFile(t *testing.T) {
	root := schemaObj(t, "bs-ai-models.json")
	providers := additionalNode(t, propNode(t, root, "providers"))
	enum := enumOf(t, propNode(t, providers, "type"))
	for _, want := range []string{"openai_compatible", "gemini_interactions"} {
		if !enumContains(enum, want) {
			t.Errorf("providers.*.type enum missing %q (got %v)", want, enum)
		}
	}
	if items, ok := propNode(t, providers, "models")["items"].(map[string]any); !ok || items["type"] != "object" {
		t.Error("providers.models should be an array of objects")
	}
	if typ := propNode(t, providers, "request_timeout_seconds")["type"]; typ != "integer" {
		t.Errorf("request_timeout_seconds should be integer, got %v", typ)
	}
}

func TestConfigSchemaTTSFile(t *testing.T) {
	root := schemaObj(t, "bs-ai-tts.json")
	providers := additionalNode(t, propNode(t, root, "providers"))
	enum := enumOf(t, propNode(t, providers, "type"))
	for _, want := range []string{"openrouter_speech", "fish_audio"} {
		if !enumContains(enum, want) {
			t.Errorf("providers.*.type enum missing %q (got %v)", want, enum)
		}
	}
	if typ := propNode(t, providers, "base_url")["type"]; typ != "string" {
		t.Errorf("base_url should be string, got %v", typ)
	}
	if desc, ok := propNode(t, root, "default_provider")["description"].(string); !ok || desc == "" {
		t.Error("default_provider should carry a description")
	}
}

func TestConfigSchemaNullForUnknownFile(t *testing.T) {
	raw, err := configSchema("bs-not-a-file.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != "null" {
		t.Fatalf("unknown file should return null, got %s", raw)
	}
}

func TestConfigSchemaStripsRequiredRecursively(t *testing.T) {
	raw, err := configSchema("bs-ai-models.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"required"`) {
		t.Fatal("schema still contains a required field")
	}
}

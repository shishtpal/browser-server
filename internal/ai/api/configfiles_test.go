package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigMetaRejectsTraversalAndUnknownNames(t *testing.T) {
	for _, name := range []string{"../bs-ai-config.json", `..\\bs-ai-config.json`, "/tmp/bs-ai-config.json", "other.json"} {
		if _, _, ok := configMeta(name); ok {
			t.Fatalf("configMeta(%q) accepted an unsafe or unknown name", name)
		}
	}
	if meta, _, ok := configMeta("bs-ai-config.json"); !ok || meta.Class != "core" {
		t.Fatal("whitelisted config was rejected")
	}
}

func TestRedactAndRestoreMaskedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bs-ai-models.json")
	original := []byte(`{"providers":{"literal":{"api_key":"super-secret"},"reference":{"api_key":"env:SAFE_KEY"}}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	redacted, err := redactJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(redacted, []byte("super-secret")) || !bytes.Contains(redacted, []byte(keepSecretSentinel)) || !bytes.Contains(redacted, []byte("env:SAFE_KEY")) {
		t.Fatalf("unexpected redaction: %s", redacted)
	}

	var edited map[string]any
	if err := json.Unmarshal(redacted, &edited); err != nil {
		t.Fatal(err)
	}
	providers := edited["providers"].(map[string]any)
	providers["literal"].(map[string]any)["label"] = "changed"
	candidate, err := json.Marshal(edited)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restoreMaskedJSON(candidate, path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte("super-secret")) || !bytes.Contains(restored, []byte("changed")) {
		t.Fatalf("masked value was not restored: %s", restored)
	}
}

func TestRestoreMaskedJSONRequiresExistingValue(t *testing.T) {
	_, err := restoreMaskedJSON([]byte(`{"api_key":"__KEEP__"}`), filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("missing masked value was accepted")
	}
}

func TestWriteAtomicReplacesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q, want new", content)
	}
}

package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_MissingDirectory(t *testing.T) {
	reg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if len(reg.List()) != 0 {
		t.Fatalf("expected empty list, got %d profiles", len(reg.List()))
	}
}

func TestLoad_EmptyDirectory(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, profileDir), 0755); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(base)
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
	if len(reg.List()) != 0 {
		t.Fatalf("expected empty list, got %d profiles", len(reg.List()))
	}
}

func TestLoad_ValidProfiles(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, profileDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create sample profiles
	os.WriteFile(filepath.Join(dir, "Architect.md"), []byte("You are a software architect."), 0644)
	os.WriteFile(filepath.Join(dir, "Codex.md"), []byte("You are a coding assistant."), 0644)
	os.WriteFile(filepath.Join(dir, "README.txt"), []byte("not a profile"), 0644) // should be skipped

	reg, err := Load(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles := reg.List()
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}

	// Check Architect profile
	content, ok := reg.Get("architect")
	if !ok {
		t.Fatal("expected to find 'architect' profile")
	}
	if content != "You are a software architect." {
		t.Fatalf("unexpected content: %s", content)
	}

	// Check case-insensitive lookup
	content2, ok := reg.Get("ARCHITECT")
	if !ok || content2 != content {
		t.Fatal("case-insensitive lookup failed")
	}

	// Check label preserves case
	for _, p := range profiles {
		if p.Name == "architect" && p.Label != "Architect" {
			t.Fatalf("expected label 'Architect', got '%s'", p.Label)
		}
	}
}

func TestLoad_OversizedFile(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, profileDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file that exceeds maxProfileSize
	bigContent := strings.Repeat("x", maxProfileSize+1)
	os.WriteFile(filepath.Join(dir, "Big.md"), []byte(bigContent), 0644)
	os.WriteFile(filepath.Join(dir, "Small.md"), []byte("small"), 0644)

	reg, err := Load(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles := reg.List()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile (oversized skipped), got %d", len(profiles))
	}
	if profiles[0].Name != "small" {
		t.Fatalf("expected 'small' profile, got '%s'", profiles[0].Name)
	}
}

func TestLoad_SubdirectoriesSkipped(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, profileDir)
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "Valid.md"), []byte("valid"), 0644)

	reg, err := Load(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.List()) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(reg.List()))
	}
}

func TestGet_NilRegistry(t *testing.T) {
	var reg *Registry
	content, ok := reg.Get("anything")
	if ok || content != "" {
		t.Fatal("expected empty result from nil registry")
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := &Registry{byName: map[string]string{"existing": "content"}}
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

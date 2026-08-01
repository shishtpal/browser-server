package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePathsEmptyOk(t *testing.T) {
	cfg := &Config{Path: filepath.Join(t.TempDir(), "bs-ai-config.json")}
	if err := validatePaths(PathsConfig{}, cfg); err != nil {
		t.Fatalf("empty paths should validate, got %v", err)
	}
}

func TestValidatePathsAdditionalDirMissing(t *testing.T) {
	cfg := &Config{Path: filepath.Join(t.TempDir(), "bs-ai-config.json")}
	paths := PathsConfig{AdditionalDirs: []string{"/nonexistent/path/that/does/not/exist"}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "additional_dirs") {
		t.Fatalf("expected additional_dirs error, got %v", err)
	}
}

func TestValidatePathsAdditionalDirNotADir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Path: filepath.Join(dir, "bs-ai-config.json")}
	paths := PathsConfig{AdditionalDirs: []string{file}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestValidatePathsAdditionalDirRelativeResolved(t *testing.T) {
	base := t.TempDir()
	subdir := filepath.Join(base, "bindir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Path: filepath.Join(base, "bs-ai-config.json")}
	paths := PathsConfig{AdditionalDirs: []string{"bindir"}}
	if err := validatePaths(paths, cfg); err != nil {
		t.Fatalf("relative dir should resolve against config dir, got %v", err)
	}
}

func TestValidatePathsAdditionalDirEmptyEntry(t *testing.T) {
	cfg := &Config{Path: filepath.Join(t.TempDir(), "bs-ai-config.json")}
	paths := PathsConfig{AdditionalDirs: []string{""}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "empty entry") {
		t.Fatalf("expected empty entry error, got %v", err)
	}
}

func TestValidatePathsAdditionalDirLimit(t *testing.T) {
	base := t.TempDir()
	dirs := make([]string, maxPathsAdditionalDirs+1)
	for i := range dirs {
		d := filepath.Join(base, "d"+strings.Repeat("x", i))
		_ = os.Mkdir(d, 0755)
		dirs[i] = d
	}
	cfg := &Config{Path: filepath.Join(base, "bs-ai-config.json")}
	paths := PathsConfig{AdditionalDirs: dirs}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected exceeds error, got %v", err)
	}
}

func TestValidatePathsBinaryMissing(t *testing.T) {
	cfg := &Config{Path: filepath.Join(t.TempDir(), "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"git": "/nonexistent/git"}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "binaries") {
		t.Fatalf("expected binaries error, got %v", err)
	}
}

func TestValidatePathsBinaryIsDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Path: filepath.Join(dir, "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"git": dir}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func TestValidatePathsBinaryNameHasSlash(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git")
	_ = os.WriteFile(file, []byte("x"), 0755)
	cfg := &Config{Path: filepath.Join(dir, "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"sub/git": file}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "path separators") {
		t.Fatalf("expected path separators error, got %v", err)
	}
}

func TestValidatePathsBinaryEmptyName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git")
	_ = os.WriteFile(file, []byte("x"), 0755)
	cfg := &Config{Path: filepath.Join(dir, "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"": file}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Fatalf("expected empty name error, got %v", err)
	}
}

func TestValidatePathsBinaryEmptyPath(t *testing.T) {
	cfg := &Config{Path: filepath.Join(t.TempDir(), "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"git": ""}}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}

func TestValidatePathsBinaryRelativeResolved(t *testing.T) {
	base := t.TempDir()
	bin := filepath.Join(base, "git")
	_ = os.WriteFile(bin, []byte("x"), 0755)
	cfg := &Config{Path: filepath.Join(base, "bs-ai-config.json")}
	paths := PathsConfig{Binaries: map[string]string{"git": "git"}}
	if err := validatePaths(paths, cfg); err != nil {
		t.Fatalf("relative binary path should resolve against config dir, got %v", err)
	}
}

func TestValidatePathsBinariesLimit(t *testing.T) {
	base := t.TempDir()
	binaries := make(map[string]string, maxPathsBinaries+1)
	for i := 0; i <= maxPathsBinaries; i++ {
		name := "b" + strings.Repeat("x", i)
		binaries[name] = filepath.Join(base, name)
		_ = os.WriteFile(binaries[name], []byte("x"), 0755)
	}
	cfg := &Config{Path: filepath.Join(base, "bs-ai-config.json")}
	paths := PathsConfig{Binaries: binaries}
	err := validatePaths(paths, cfg)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected exceeds error, got %v", err)
	}
}

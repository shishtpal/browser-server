package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

// execLookPath wraps exec.LookPath for testability.
func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func TestResolveBinaryExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "mygit")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	paths := config.PathsConfig{Binaries: map[string]string{"git": bin}}
	got := resolveBinary("git", paths)
	if got != bin {
		t.Fatalf("resolveBinary = %q, want %q", got, bin)
	}
}

func TestResolveBinaryExplicitOverrideWithoutExecExt(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "uv")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	paths := config.PathsConfig{Binaries: map[string]string{"uv": bin}}
	got := resolveBinary("uv.exe", paths)
	if got != bin {
		t.Fatalf("resolveBinary(\"uv.exe\") = %q, want %q", got, bin)
	}
}

func TestResolveBinaryExplicitOverridePriority(t *testing.T) {
	dir := t.TempDir()
	// Put a fake "git" in additional_dirs
	fake := filepath.Join(dir, "git")
	_ = os.WriteFile(fake, []byte("fake"), 0755)
	// And a different one as explicit override
	explicit := filepath.Join(dir, "real-git")
	_ = os.WriteFile(explicit, []byte("real"), 0755)
	paths := config.PathsConfig{
		AdditionalDirs: []string{dir},
		Binaries:       map[string]string{"git": explicit},
	}
	got := resolveBinary("git", paths)
	if got != explicit {
		t.Fatalf("resolveBinary = %q, want explicit %q", got, explicit)
	}
}

func TestResolveBinaryAdditionalDir(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "mytool")
	_ = os.WriteFile(bin, []byte("x"), 0755)
	paths := config.PathsConfig{AdditionalDirs: []string{dir}}
	got := resolveBinary("mytool", paths)
	if got != bin {
		t.Fatalf("resolveBinary = %q, want %q", got, bin)
	}
}

func TestResolveBinaryAdditionalDirFirstMatchWins(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	first := filepath.Join(dir1, "tool")
	_ = os.WriteFile(first, []byte("1"), 0755)
	second := filepath.Join(dir2, "tool")
	_ = os.WriteFile(second, []byte("2"), 0755)
	paths := config.PathsConfig{AdditionalDirs: []string{dir1, dir2}}
	got := resolveBinary("tool", paths)
	if got != first {
		t.Fatalf("resolveBinary = %q, want first match %q", got, first)
	}
}

func TestResolveBinaryFallsBackToSystemPath(t *testing.T) {
	// "go" is expected to be on PATH in CI; if not, skip.
	if _, err := execLookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	paths := config.PathsConfig{}
	got := resolveBinary("go", paths)
	if got == "go" {
		t.Fatalf("resolveBinary should have resolved to full path, got bare name")
	}
}

func TestResolveBinaryUnknownReturnsBareName(t *testing.T) {
	paths := config.PathsConfig{}
	got := resolveBinary("this-binary-does-not-exist-xyz123", paths)
	if got != "this-binary-does-not-exist-xyz123" {
		t.Fatalf("resolveBinary = %q, want bare name", got)
	}
}

func TestChildEnvNilWhenEmpty(t *testing.T) {
	paths := config.PathsConfig{}
	if env := childEnv(paths); env != nil {
		t.Fatalf("childEnv should return nil when no additional dirs, got %v", env)
	}
}

func TestChildEnvPrependsAdditionalDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	paths := config.PathsConfig{AdditionalDirs: []string{dir1, dir2}}
	env := childEnv(paths)
	if env == nil {
		t.Fatal("childEnv should not be nil when additional dirs are set")
	}
	var pathEntry string
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			pathEntry = e
			break
		}
	}
	if pathEntry == "" {
		t.Fatal("PATH entry not found in child env")
	}
	if !strings.HasPrefix(pathEntry[len("PATH="):], dir1) {
		t.Fatalf("PATH should start with first additional dir, got %q", pathEntry)
	}
	if !strings.Contains(pathEntry, dir2) {
		t.Fatalf("PATH should contain second additional dir, got %q", pathEntry)
	}
}

func TestChildEnvPreservesExistingPath(t *testing.T) {
	dir := t.TempDir()
	paths := config.PathsConfig{AdditionalDirs: []string{dir}}
	env := childEnv(paths)
	if env == nil {
		t.Fatal("childEnv should not be nil")
	}
	originalPath := os.Getenv("PATH")
	if originalPath == "" {
		t.Skip("PATH not set in environment")
	}
	var pathEntry string
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			pathEntry = e
			break
		}
	}
	if !strings.Contains(pathEntry, originalPath) {
		t.Fatalf("PATH should preserve original PATH, got %q", pathEntry)
	}
}

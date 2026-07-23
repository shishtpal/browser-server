package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileAppliesMultipleHunks(t *testing.T) {
	path := writeEditFixture(t, "one\ntwo\nthree\nfour\nfive\n")
	patch := "--- a/test.txt\n+++ b/test.txt\n" +
		"@@ -1,2 +1,2 @@\n one\n-two\n+TWO\n" +
		"@@ -4,2 +4,3 @@\n four\n+four-half\n five\n"

	result, err := editFile(context.Background(), editArgs(t, path, patch, false))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["hunks_applied"] != 2 || response["lines_added"] != 2 || response["lines_removed"] != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
	assertEditContent(t, path, "one\nTWO\nthree\nfour\nfour-half\nfive\n")
}

func TestEditFileSupportsInsertRemoveAndFuzz(t *testing.T) {
	path := writeEditFixture(t, "zero\none\ntwo\nthree\n")
	// The declared start is two lines later than the matching context.
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -3,2 +3,2 @@\n one\n-two\n+inserted\n"

	if _, err := editFile(context.Background(), editArgs(t, path, patch, false)); err != nil {
		t.Fatal(err)
	}
	assertEditContent(t, path, "zero\none\ninserted\nthree\n")
}

func TestEditFileDryRunReturnsPreviewWithoutWriting(t *testing.T) {
	path := writeEditFixture(t, "before\n")
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-before\n+after\n"

	result, err := editFile(context.Background(), editArgs(t, path, patch, true))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["dry_run"] != true || response["preview"] != "after\n" {
		t.Fatalf("unexpected dry-run response: %#v", response)
	}
	assertEditContent(t, path, "before\n")
}

func TestEditFileCanEditEmptyFile(t *testing.T) {
	path := writeEditFixture(t, "")
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -0,0 +1,2 @@\n+first\n+second\n"

	if _, err := editFile(context.Background(), editArgs(t, path, patch, false)); err != nil {
		t.Fatal(err)
	}
	assertEditContent(t, path, "first\nsecond\n")
}

func TestEditFileCanInsertWithoutContext(t *testing.T) {
	path := writeEditFixture(t, "first\nthird\n")
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -1,0 +2 @@\n+second\n"

	if _, err := editFile(context.Background(), editArgs(t, path, patch, false)); err != nil {
		t.Fatal(err)
	}
	assertEditContent(t, path, "first\nsecond\nthird\n")
}

func TestEditFilePreservesCRLFLineEndings(t *testing.T) {
	path := writeEditFixture(t, "before\r\nafter\r\n")
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -1,2 +1,2 @@\n-before\n+changed\n after\n"

	if _, err := editFile(context.Background(), editArgs(t, path, patch, false)); err != nil {
		t.Fatal(err)
	}
	assertEditContent(t, path, "changed\r\nafter\r\n")
}

func TestEditFileRejectsInvalidPatches(t *testing.T) {
	path := writeEditFixture(t, "one\ntwo\n")
	tests := map[string]string{
		"missing headers":  "@@ -1 +1 @@\n-one\n+ONE\n",
		"count mismatch":   "--- a/test.txt\n+++ b/test.txt\n@@ -1,2 +1,2 @@\n-one\n+ONE\n",
		"context mismatch": "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-other\n+ONE\n",
	}
	for name, patch := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := editFile(context.Background(), editArgs(t, path, patch, false)); err == nil {
				t.Fatal("expected patch rejection")
			}
			assertEditContent(t, path, "one\ntwo\n")
		})
	}
}

func TestEditFileRejectsMissingFileAndSymlink(t *testing.T) {
	dir := t.TempDir()
	patch := "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-one\n+ONE\n"
	if _, err := editFile(context.Background(), editArgs(t, filepath.Join(dir, "missing.txt"), patch, false)); err == nil {
		t.Fatal("expected missing file rejection")
	}

	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := editFile(context.Background(), editArgs(t, link, patch, false)); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic link rejection, got %v", err)
	}
}

func writeEditFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func editArgs(t *testing.T, path, patch string, dryRun bool) json.RawMessage {
	t.Helper()
	args, err := json.Marshal(map[string]any{"path": path, "patch": patch, "dry_run": dryRun})
	if err != nil {
		t.Fatal(err)
	}
	return args
}

func assertEditContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("file content = %q, want %q", content, want)
	}
}

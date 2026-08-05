package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestBuildPromptFilesFirstPromptLast(t *testing.T) {
	fileA := writeTempFile(t, "main.go", "package main\n")
	fileB := writeTempFile(t, "notes.txt", "hello world\n")

	got, err := buildPrompt([]string{fileA, fileB}, "what does this do?", 64*1024)
	if err != nil {
		t.Fatalf("buildPrompt: %v", err)
	}

	idxA := strings.Index(got, "<file path=")
	idxB := strings.Index(got, "notes.txt")
	idxPrompt := strings.Index(got, "what does this do?")
	if idxA < 0 || idxB < 0 || idxPrompt < 0 {
		t.Fatalf("missing parts in output: %q", got)
	}
	if !(idxA < idxB && idxB < idxPrompt) {
		t.Fatalf("files must come before the prompt: fileA=%d fileB=%d prompt=%d", idxA, idxB, idxPrompt)
	}
	if !strings.Contains(got, "package main") || !strings.Contains(got, "hello world") {
		t.Fatalf("file contents missing: %q", got)
	}
}

func TestBuildPromptNoFilesReturnsPrompt(t *testing.T) {
	got, err := buildPrompt(nil, "just a prompt", 64*1024)
	if err != nil {
		t.Fatalf("buildPrompt: %v", err)
	}
	if got != "just a prompt" {
		t.Fatalf("got %q, want prompt unchanged", got)
	}
}

func TestBuildPromptFileOnly(t *testing.T) {
	file := writeTempFile(t, "data.csv", "a,b\n1,2\n")
	got, err := buildPrompt([]string{file}, "", 64*1024)
	if err != nil {
		t.Fatalf("buildPrompt: %v", err)
	}
	if !strings.Contains(got, "<file path=") || !strings.Contains(got, "a,b") {
		t.Fatalf("file-only prompt missing content: %q", got)
	}
}

func TestInlineFilesRejectsNonUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary.bin")
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 0x00, 0x01}, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := inlineFiles([]string{path}, 64*1024)
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("expected UTF-8 error, got %v", err)
	}
}

func TestInlineFilesRejectsOversize(t *testing.T) {
	path := writeTempFile(t, "big.txt", strings.Repeat("x", 100))
	_, err := inlineFiles([]string{path}, 50)
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("expected size-limit error, got %v", err)
	}
}

func TestInlineFilesMissingFile(t *testing.T) {
	_, err := inlineFiles([]string{filepath.Join(t.TempDir(), "missing.txt")}, 64*1024)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSplitList(t *testing.T) {
	got := splitList("a, b,,c ,")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestConversationTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello world", "hello world"},
		{"  spaced   out  ", "spaced out"},
		{strings.Repeat("a", 100), strings.Repeat("a", 60)},
		{"", "New chat"},
	}
	for _, c := range cases {
		if got := conversationTitle(c.in); got != c.want {
			t.Errorf("conversationTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

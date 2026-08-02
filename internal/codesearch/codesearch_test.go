package codesearch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateSetLiteral(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello world\nfoo bar\n"), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := CandidateSet(context.Background(), Options{
		Root:    root,
		Pattern: "hello",
		Type:    "literal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(set.Candidates))
	}
	m := set.Candidates[0].Value
	if m.Line != 1 || m.Column != 1 || m.Match != "hello" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestCandidateSetExcludes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "skip.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := CandidateSet(context.Background(), Options{
		Root:    root,
		Pattern: "hello",
		Type:    "literal",
		Exclude: []string{"skip.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(set.Candidates))
	}
	if !strings.HasSuffix(set.Candidates[0].Value.File, "keep.txt") {
		t.Fatalf("expected keep.txt, got %s", set.Candidates[0].Value.File)
	}
}

func TestCandidateSetContext(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\nb\nc\nd\ne\n"), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := CandidateSet(context.Background(), Options{
		Root:         root,
		Pattern:      "c",
		Type:         "literal",
		ContextLines: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := set.Candidates[0].Value
	if len(m.ContextBefore) != 1 || m.ContextBefore[0] != "b" {
		t.Fatalf("unexpected context before: %v", m.ContextBefore)
	}
	if len(m.ContextAfter) != 1 || m.ContextAfter[0] != "d" {
		t.Fatalf("unexpected context after: %v", m.ContextAfter)
	}
}

func TestCandidateSetGlobInclude(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := CandidateSet(context.Background(), Options{
		Root:    root,
		Pattern: "hello",
		Type:    "literal",
		Include: []string{"*.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(set.Candidates))
	}
}

func TestCandidateSetCapAndPerMatchScores(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("match matching xmatch\nmatch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := CandidateSet(context.Background(), Options{
		Root: root, Pattern: "match", Type: "literal", MaxCandidates: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 2 || !set.Truncated {
		t.Fatalf("expected two capped candidates and truncation, got %+v", set)
	}
	if set.Candidates[0].BaseScore <= set.Candidates[1].BaseScore {
		t.Fatalf("whole-token match should outrank prefix: %+v", set.Candidates)
	}
}

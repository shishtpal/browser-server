package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

func TestRawSearchCodeFormatterNoMatches(t *testing.T) {
	env := makeSearchCodeEnvelope("foo", 1, 10, 0, 0, false, false, 0, nil)
	raw, ok := rawSearchCodeFormatter(env)
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	want := "QUERY foo | no matches\n"
	if string(raw) != want {
		t.Fatalf("got %q, want %q", raw, want)
	}
}

func TestRawSearchCodeFormatterOneFileWithContext(t *testing.T) {
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("alpha", 1, 10, 1, 1, false, false, 0, []map[string]any{
		{
			"file":           "main.go",
			"line":           2,
			"column":         1,
			"match":          "alpha",
			"score":          0.9,
			"context_before": []string{"package main"},
			"context_after":  []string{"func main() {}", ""},
		},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	got := string(raw)
	if !strings.Contains(got, "QUERY alpha | matches=1 page=1/10 score=0.9") {
		t.Fatalf("missing header: %s", got)
	}
	if !strings.Contains(got, "FILE main.go") {
		t.Fatalf("missing file: %s", got)
	}
	if !strings.Contains(got, "2:1") {
		t.Fatalf("missing position: %s", got)
	}
	if !strings.Contains(got, "  > alpha") {
		t.Fatalf("missing match line: %s", got)
	}
	if !strings.Contains(got, "  -1 package main") {
		t.Fatalf("missing context before: %s", got)
	}
	if !strings.Contains(got, "  +1 func main() {}") {
		t.Fatalf("missing context after: %s", got)
	}
	if strings.Contains(got, "+2") {
		t.Fatalf("blank context after should be dropped: %s", got)
	}
}

func TestRawSearchCodeFormatterMultipleFiles(t *testing.T) {
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("foo", 1, 10, 3, 3, false, false, 0, []map[string]any{
		{"file": "a.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
		{"file": "b.go", "line": 5, "column": 3, "match": "foo", "score": 0.9},
		{"file": "a.go", "line": 10, "column": 1, "match": "foo", "score": 0.9},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	got := string(raw)
	if !strings.Contains(got, "FILE a.go") || !strings.Contains(got, "FILE b.go") {
		t.Fatalf("missing file headers: %s", got)
	}
	// a.go should have both hits grouped; b.go should have one.
	if strings.Count(got, "FILE a.go") != 1 {
		t.Fatalf("a.go should appear once: %s", got)
	}
	if strings.Count(got, "FILE b.go") != 1 {
		t.Fatalf("b.go should appear once: %s", got)
	}
	if strings.Count(got, "1:1") != 1 || strings.Count(got, "10:1") != 1 || strings.Count(got, "5:3") != 1 {
		t.Fatalf("expected each position exactly once: %s", got)
	}
}

func TestRawSearchCodeFormatterVariableScores(t *testing.T) {
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("foo", 1, 10, 2, 2, false, false, 0, []map[string]any{
		{"file": "a.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
		{"file": "b.go", "line": 1, "column": 1, "match": "foo", "score": 0.7},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	got := string(raw)
	if strings.Contains(got, "| matches=2 page=1/10 score=") {
		t.Fatalf("constant score should not appear in header: %s", got)
	}
	if !strings.Contains(got, "1:1 score=0.9") || !strings.Contains(got, "1:1 score=0.7") {
		t.Fatalf("per-hit scores missing: %s", got)
	}
}

func TestRawSearchCodeFormatterPaginationFlags(t *testing.T) {
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("foo", 2, 5, 12, 12, true, true, 0, []map[string]any{
		{"file": "a.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	got := string(raw)
	if !strings.Contains(got, "QUERY foo | matches=1 page=2/5 score=0.9 has_more=true truncated=true") {
		t.Fatalf("header missing flags: %s", got)
	}
}

func TestRawSearchCodeFormatterEmbeddedNewline(t *testing.T) {
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("foo", 1, 10, 1, 1, false, false, 0, []map[string]any{
		{"file": "a.go", "line": 1, "column": 1, "match": "foo\nbar", "score": 0.9},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	got := string(raw)
	if !strings.Contains(got, "  > foo⏎bar") {
		t.Fatalf("embedded newline not escaped: %s", got)
	}
	if strings.Contains(got, "\nbar") {
		t.Fatalf("embedded newline broke line invariant: %s", got)
	}
}

func TestRawSearchCodeFormatterRejectsNonEnvelope(t *testing.T) {
	if _, ok := rawSearchCodeFormatter(map[string]any{}); ok {
		t.Fatal("formatter should reject non-envelope value")
	}
}

func TestRawSearchCodeFormatterPreservesFileOrder(t *testing.T) {
	// b.go is seen first, then a.go; raw output should preserve first-seen order.
	raw, ok := rawSearchCodeFormatter(makeSearchCodeEnvelope("foo", 1, 10, 2, 2, false, false, 0, []map[string]any{
		{"file": "b.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
		{"file": "a.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
	}))
	if !ok {
		t.Fatal("expected formatter to accept envelope")
	}
	idxB := strings.Index(string(raw), "FILE b.go")
	idxA := strings.Index(string(raw), "FILE a.go")
	if idxB < 0 || idxA < 0 || idxA < idxB {
		t.Fatalf("expected b.go before a.go: %s", raw)
	}
}

func TestExecuteSearchCodeRawJSONOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.go"), []byte("package main\nfunc hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []byte(fmt.Sprintf(`{"pattern":"hello","path":%q,"type":"literal"}`, root))

	// Default registry (search_code not in raw allowlist) -> JSON envelope.
	r := New()
	out, err := r.Execute(context.Background(), "search_code", args)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("default output should be JSON: %s", out)
	}
	if !bytes.Contains(out, []byte(`"results"`)) {
		t.Fatalf("JSON envelope missing results key: %s", out)
	}

	// Force raw via context override -> compact text.
	raw := true
	ctx := WithRawOutputOverride(context.Background(), &raw)
	out, err = r.Execute(ctx, "search_code", args)
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(out) {
		t.Fatalf("forced raw output should not be JSON: %s", out)
	}
	if !bytes.Contains(out, []byte("QUERY hello")) {
		t.Fatalf("raw output missing query header: %s", out)
	}
	if !bytes.Contains(out, []byte("FILE")) {
		t.Fatalf("raw output missing file header: %s", out)
	}

	// Config allowlist default raw, force JSON override -> JSON.
	r2 := New(Options{Tools: config.ToolsConfig{RawOutput: []string{"search_code"}}})
	forceJSON := false
	out, err = r2.Execute(WithRawOutputOverride(context.Background(), &forceJSON), "search_code", args)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("forced JSON output should be JSON: %s", out)
	}

	// Config allowlist default raw, no override -> raw.
	out, err = r2.Execute(context.Background(), "search_code", args)
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(out) {
		t.Fatalf("config raw output should not be JSON: %s", out)
	}
}

func TestExecuteSearchCodeRawOutputLimit(t *testing.T) {
	r := New(Options{Tools: config.ToolsConfig{MaxOutput: 64, RawOutput: []string{"search_code"}}})
	// search_code result is naturally small; force a huge match text to exceed 64 bytes.
	root := t.TempDir()
	big := strings.Repeat("x", 200)
	content := fmt.Sprintf("package main\nfunc %s() {}\n", big)
	if err := os.WriteFile(filepath.Join(root, "x.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	args := []byte(fmt.Sprintf(`{"pattern":%q,"path":%q,"type":"literal"}`, big, root))
	_, err := r.Execute(context.Background(), "search_code", args)
	if err == nil || err.Error() != "tool output exceeds limit" {
		t.Fatalf("expected output limit error, got %v", err)
	}
}

func TestExecuteSearchCodeJSONFallbackWithoutRawFunc(t *testing.T) {
	// A search_code stub that returns a plain map should still render as JSON
	// even when raw is forced, because only the typed envelope triggers raw formatting.
	r := New()
	r.add(Tool{
		Name: "search_code_stub",
		Execute: func(context.Context, json.RawMessage) (any, error) {
			return makeSearchCodeEnvelope("foo", 1, 10, 1, 1, false, false, 0, []map[string]any{
				{"file": "a.go", "line": 1, "column": 1, "match": "foo", "score": 0.9},
			}), nil
		},
		RawContentFunc: rawSearchCodeFormatter,
	})
	raw := true
	out, err := r.Execute(WithRawOutputOverride(context.Background(), &raw), "search_code_stub", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(out) {
		t.Fatalf("expected raw output for typed envelope: %s", out)
	}
	if !bytes.Contains(out, []byte("QUERY foo")) {
		t.Fatalf("expected raw query header: %s", out)
	}
}

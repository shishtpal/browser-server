package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"browser-server/internal/ai/config"
)

func TestStrictToolArguments(t *testing.T) {
	r := New()
	if _, err := r.Execute(context.Background(), "get_current_time", []byte(`{"unknown":1}`)); err == nil {
		t.Fatal("expected unknown argument rejection")
	}
	if _, err := r.Execute(context.Background(), "list_directory", []byte(`null`)); err == nil {
		t.Fatal("expected null argument rejection")
	}
	if _, err := r.Execute(context.Background(), "missing", []byte(`{}`)); err == nil {
		t.Fatal("expected unknown tool rejection")
	}
}

func TestFilesystemToolsAreRegistered(t *testing.T) {
	r := New()
	names := []string{"read_file", "write_file", "edit_file", "multi_edit", "list_directory", "delete_file", "move_file", "copy_file"}
	if specs := r.Specs(names); len(specs) != len(names) {
		t.Fatalf("got %d filesystem tool specs, want %d", len(specs), len(names))
	}
}

func TestSearchToolValidatesArguments(t *testing.T) {
	r := New(Options{Allowed: []string{SearchToolName, "read_file"}})
	for _, raw := range []string{`{}`, `{"query":""}`, `{"query":"file","limit":6}`, `{"query":"file","unknown":true}`} {
		if _, err := r.Search([]byte(raw)); err == nil {
			t.Fatalf("Search(%s) succeeded, want validation error", raw)
		}
	}
}

func TestSearchToolRanksFiltersAndLimitsMatches(t *testing.T) {
	allowed := []string{
		SearchToolName,
		"write_file",
		"read_file",
		"edit_file",
		"copy_file",
		"move_file",
		"delete_file",
		"missing_tool",
	}
	r := New(Options{Allowed: allowed})
	result, err := r.Search([]byte(`{"query":"read_file"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) == 0 || result.Matches[0].Name != "read_file" {
		t.Fatalf("matches = %#v, want exact name first", result.Matches)
	}
	for _, match := range result.Matches {
		if match.Name == SearchToolName || match.Name == "missing_tool" {
			t.Fatalf("unexpected match %#v", match)
		}
	}

	result, err = r.Search([]byte(`{"query":"file"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 5 {
		t.Fatalf("default match count = %d, want 5", len(result.Matches))
	}
	limited, err := r.Search([]byte(`{"query":"file","limit":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Matches) != 2 {
		t.Fatalf("limited match count = %d, want 2", len(limited.Matches))
	}
}

func TestWriteFileRequiresContent(t *testing.T) {
	r := New()
	path := filepath.Join(t.TempDir(), "empty.txt")
	args := []byte(`{"path":` + quoted(path) + `}`)
	if _, err := r.Execute(context.Background(), "write_file", args); err == nil {
		t.Fatal("expected missing content rejection")
	}
}

func TestCopyFileRejectsExistingDestinationWithoutTruncatingIt(t *testing.T) {
	r := New()
	path := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(path, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []byte(`{"source":` + quoted(path) + `,"destination":` + quoted(path) + `}`)
	if _, err := r.Execute(context.Background(), "copy_file", args); err == nil {
		t.Fatal("expected same-file copy rejection")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep me" {
		t.Fatalf("source content = %q, want %q", content, "keep me")
	}
}

func TestDeleteFileDoesNotTrimApprovedPath(t *testing.T) {
	r := New()
	dir := t.TempDir()
	spacedPath := filepath.Join(dir, " report ")
	trimmedPath := filepath.Join(dir, "report")
	if err := os.WriteFile(spacedPath, []byte("spaced"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trimmedPath, []byte("trimmed"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []byte(`{"path":` + quoted(spacedPath) + `}`)
	if _, err := r.Execute(context.Background(), "delete_file", args); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(trimmedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "trimmed" {
		t.Fatalf("trimmed-path content = %q, want %q", content, "trimmed")
	}
}

func TestExecuteRawOutputOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("hello raw"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []byte(`{"path":` + quoted(path) + `}`)

	// Default (no config allowlist, no override): JSON envelope.
	r := New()
	out, err := r.Execute(context.Background(), "read_file", args)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("default read_file output = %q, want JSON", out)
	}

	// Force raw via context override -> bare content, no JSON envelope.
	raw := true
	ctx := WithRawOutputOverride(context.Background(), &raw)
	out, err = r.Execute(ctx, "read_file", args)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello raw" {
		t.Fatalf("forced raw read_file output = %q, want %q", out, "hello raw")
	}

	// Force JSON via context override on a config-raw-enabled registry.
	r2 := New(Options{Tools: config.ToolsConfig{RawOutput: []string{"read_file"}}})
	forceJSON := false
	out, err = r2.Execute(WithRawOutputOverride(context.Background(), &forceJSON), "read_file", args)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("forced JSON read_file output = %q, want JSON envelope", out)
	}

	// Config allowlist still applies when no override is present.
	out, err = r2.Execute(context.Background(), "read_file", args)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello raw" {
		t.Fatalf("config raw read_file output = %q, want %q", out, "hello raw")
	}

	// Tools without RawContentFunc stay JSON even when raw is forced.
	out, err = r.Execute(ctx, "get_current_time", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("get_current_time output = %q, want JSON even in forced raw mode", out)
	}
}

func quoted(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

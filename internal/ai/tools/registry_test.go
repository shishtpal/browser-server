package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	names := []string{"read_file", "write_file", "edit_file", "list_directory", "delete_file", "move_file", "copy_file"}
	if specs := r.Specs(names); len(specs) != len(names) {
		t.Fatalf("got %d filesystem tool specs, want %d", len(specs), len(names))
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

func quoted(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

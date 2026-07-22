package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func registerListDirectory(r *Registry) {
	r.add(Tool{
		Name:        "list_directory",
		Category:    "File Operations",
		Description: "List the immediate contents of a directory on the server filesystem",
		Schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path on the server; defaults to the server working directory"}},"additionalProperties":false}`),
		Execute:     listDirectory,
	})
}

func listDirectory(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true}); err != nil {
		return nil, err
	}
	if a.Path == "" {
		a.Path = "."
	} else if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path cannot contain only whitespace")
	}
	dir, err := os.Open(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}
	defer dir.Close()
	result := []map[string]any{}
	outputBytes := len(a.Path) + 128
	truncated := false
	for {
		entries, err := dir.ReadDir(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}
		entry := entries[0]
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to inspect %q: %w", entry.Name(), err)
		}
		item := map[string]any{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mod_time": info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		}
		encoded, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to encode directory entry: %w", err)
		}
		if outputBytes+len(encoded)+1 > maxOutput-512 {
			truncated = true
			break
		}
		result = append(result, item)
		outputBytes += len(encoded) + 1
	}
	return map[string]any{"path": a.Path, "entries": result, "truncated": truncated}, nil
}

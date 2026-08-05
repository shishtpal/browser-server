package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

//go:embed schemas/list_directory.json
var listDirectorySchema []byte

func registerListDirectory(r *Registry) {
	r.add(Tool{
		Name:           "list_directory",
		Category:       "File Operations",
		Description:    "List the immediate contents of a directory on the server filesystem",
		Schema:         json.RawMessage(listDirectorySchema),
		Execute:        listDirectory,
		RawContentFunc: rawListDirectoryResult,
	})
}

func listDirectory(ctx context.Context, raw json.RawMessage) (any, error) {
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
		if outputBytes+len(encoded)+1 > outputBudget(ctx) {
			truncated = true
			break
		}
		result = append(result, item)
		outputBytes += len(encoded) + 1
	}
	return map[string]any{"path": a.Path, "entries": result, "truncated": truncated}, nil
}

// rawListDirectoryResult removes per-entry JSON metadata in raw-output mode.
// Names grouped by type retain the information needed to navigate while using
// substantially fewer tokens than size and timestamp fields for every entry.
func rawListDirectoryResult(value any) ([]byte, bool) {
	result, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	path, ok := result["path"].(string)
	if !ok {
		return nil, false
	}
	truncated, ok := result["truncated"].(bool)
	if !ok {
		return nil, false
	}
	entries, ok := result["entries"].([]map[string]any)
	if !ok {
		return nil, false
	}

	dirs := make([]string, 0, len(entries))
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, nameOK := entry["name"].(string)
		isDir, dirOK := entry["is_dir"].(bool)
		if !nameOK || !dirOK {
			return nil, false
		}
		if isDir {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}

	var output strings.Builder
	fmt.Fprintf(&output, "path=%s\ntruncated=%t", path, truncated)
	if len(dirs) > 0 {
		output.WriteString("\ndirs=")
		output.WriteString(strings.Join(dirs, ","))
	}
	if len(files) > 0 {
		output.WriteString("\nfiles=")
		output.WriteString(strings.Join(files, ","))
	}
	return []byte(output.String()), true
}

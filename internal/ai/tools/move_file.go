package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func registerMoveFile(r *Registry) {
	r.add(Tool{
		Name:        "move_file",
		Category:    "File Operations",
		Description: "Move or rename a file on the server filesystem without overwriting an existing destination, creating parent directories as needed",
		Schema:      json.RawMessage(`{"type":"object","properties":{"source":{"type":"string","description":"Existing source file path"},"destination":{"type":"string","description":"New destination file path"}},"required":["source","destination"],"additionalProperties":false}`),
		Execute:     moveFile,
	})
}

func moveFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Source string `json:"source"`
		Dest   string `json:"destination"`
	}
	if err := strict(raw, &a, map[string]bool{"source": true, "destination": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Source) == "" || strings.TrimSpace(a.Dest) == "" {
		return nil, fmt.Errorf("source and destination paths are required")
	}
	info, err := os.Lstat(a.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect source file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source must be a regular file")
	}
	if _, err := os.Lstat(a.Dest); err == nil {
		return nil, fmt.Errorf("destination already exists")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect destination file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(a.Dest), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	if err := os.Rename(a.Source, a.Dest); err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}
	return map[string]any{"source": a.Source, "destination": a.Dest, "success": true}, nil
}

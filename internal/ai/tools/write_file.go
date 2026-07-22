package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func registerWriteFile(r *Registry) {
	r.add(Tool{
		Name:        "write_file",
		Category:    "File Operations",
		Description: "Create or overwrite a UTF-8 text file on the server filesystem, creating parent directories as needed",
		Schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Destination path on the server"},"content":{"type":"string","description":"Complete file content"}},"required":["path","content"],"additionalProperties":false}`),
		Execute:     writeFile,
	})
}

func writeFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path    string  `json:"path"`
		Content *string `json:"content"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "content": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if a.Content == nil {
		return nil, fmt.Errorf("content is required")
	}
	if info, err := os.Lstat(a.Path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("destination cannot be a symbolic link")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect destination file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(a.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(a.Path, []byte(*a.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	return map[string]any{"path": a.Path, "success": true}, nil
}

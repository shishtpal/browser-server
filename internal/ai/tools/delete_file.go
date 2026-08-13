package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

//go:embed schemas/delete_file.json
var deleteFileSchemaFS []byte

func registerDeleteFile(r *Registry) {
	r.add(Tool{
		Name:           "delete_file",
		Category:       "File Operations",
		Description:    "Delete a file from the server filesystem",
		Schema:         json.RawMessage(deleteFileSchemaFS),
		Execute:        deleteFile,
		RawContentFunc: rawTrue,
	})
}

func deleteFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	info, err := os.Lstat(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path must be a file")
	}
	if err := os.Remove(a.Path); err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}
	return map[string]any{"path": a.Path, "success": true}, nil
}

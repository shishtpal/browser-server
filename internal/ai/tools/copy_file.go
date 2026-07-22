package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func registerCopyFile(r *Registry) {
	r.add(Tool{
		Name:        "copy_file",
		Category:    "File Operations",
		Description: "Copy a file on the server filesystem without overwriting an existing destination, creating parent directories as needed",
		Schema:      json.RawMessage(`{"type":"object","properties":{"source":{"type":"string","description":"Existing source file path"},"destination":{"type":"string","description":"New destination file path"}},"required":["source","destination"],"additionalProperties":false}`),
		Execute:     copyFile,
	})
}

func copyFile(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := strict(raw, &a, map[string]bool{"source": true, "destination": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Source) == "" || strings.TrimSpace(a.Destination) == "" {
		return nil, fmt.Errorf("source and destination paths are required")
	}
	if _, err := os.Lstat(a.Destination); err == nil {
		return nil, fmt.Errorf("destination already exists")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect destination file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(a.Destination), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	sourceFile, err := os.Open(a.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect source file: %w", err)
	}
	if !sourceInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("source must be a regular file")
	}
	destFile, err := os.OpenFile(a.Destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, sourceInfo.Mode().Perm())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	complete := false
	defer func() {
		destFile.Close()
		if !complete {
			os.Remove(a.Destination)
		}
	}()
	if _, err := io.Copy(destFile, contextReader{ctx: ctx, reader: sourceFile}); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}
	if err := destFile.Chmod(sourceInfo.Mode()); err != nil {
		return nil, fmt.Errorf("failed to preserve file permissions: %w", err)
	}
	if err := destFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to finish destination file: %w", err)
	}
	complete = true
	return map[string]any{"source": a.Source, "destination": a.Destination, "success": true}, nil
}

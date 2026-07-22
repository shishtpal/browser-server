package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

func registerReadFile(r *Registry) {
	r.add(Tool{
		Name:        "read_file",
		Category:    "File Operations",
		Description: "Read a UTF-8 text file from the server filesystem (maximum 32 KiB)",
		Schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path to the file on the server"}},"required":["path"],"additionalProperties":false}`),
		Execute:     readFile,
	})
}

func readFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	file, err := os.Open(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxOutput+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if len(data) > maxOutput {
		return nil, fmt.Errorf("file exceeds %d-byte read limit", maxOutput)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("file is not valid UTF-8 text")
	}
	return map[string]any{"content": string(data), "path": a.Path}, nil
}

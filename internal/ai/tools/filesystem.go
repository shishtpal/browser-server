package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

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

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}

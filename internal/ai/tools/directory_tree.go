package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed schemas/directory_tree.json
var directoryTreeSchema []byte

func registerDirectoryTree(r *Registry) {
	r.add(Tool{
		Name:        "directory_tree",
		Category:    "File Operations",
		Description: "Generate a tree-style directory listing showing the hierarchical structure of files and folders. Ignores .git and node_modules by default.",
		Schema:      json.RawMessage(directoryTreeSchema),
		Execute:     directoryTree,
	})
}

func directoryTree(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path           string   `json:"path"`
		MaxDepth       int      `json:"max_depth"`
		IgnorePatterns []string `json:"ignore_patterns"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "max_depth": true, "ignore_patterns": true}); err != nil {
		return nil, err
	}
	if a.Path == "" {
		a.Path = "."
	} else if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path cannot contain only whitespace")
	}
	if a.MaxDepth <= 0 {
		a.MaxDepth = 5
	}
	if a.MaxDepth > 20 {
		a.MaxDepth = 20
	}

	defaultIgnore := []string{".git", "node_modules"}
	ignoreSet := make(map[string]bool, len(defaultIgnore)+len(a.IgnorePatterns))
	for _, p := range defaultIgnore {
		ignoreSet[p] = true
	}
	for _, p := range a.IgnorePatterns {
		p = strings.TrimSpace(p)
		if p != "" {
			ignoreSet[p] = true
		}
	}

	info, err := os.Stat(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	var builder strings.Builder
	absPath, _ := filepath.Abs(a.Path)
	builder.WriteString(filepath.Base(absPath))
	builder.WriteString("/\n")

	truncated := buildTree(&builder, a.Path, a.MaxDepth, 0, ignoreSet, outputBudget(ctx))

	return map[string]any{
		"path":      a.Path,
		"tree":      builder.String(),
		"truncated": truncated,
	}, nil
}

// buildTree uses indentation-based output (2 spaces per level) which is
// significantly more token-efficient than box-drawing connectors while
// remaining fully parseable by both humans and LLMs.
func buildTree(builder *strings.Builder, dir string, maxDepth, currentDepth int, ignoreSet map[string]bool, budget int) bool {
	if currentDepth >= maxDepth {
		return false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if shouldIgnore(entry.Name(), ignoreSet) {
			continue
		}
		filtered = append(filtered, entry)
	}

	indent := strings.Repeat("  ", currentDepth+1)

	for _, entry := range filtered {
		if builder.Len() > budget {
			builder.WriteString(indent)
			builder.WriteString("[truncated]\n")
			return true
		}

		builder.WriteString(indent)
		builder.WriteString(entry.Name())

		if entry.IsDir() {
			builder.WriteString("/\n")
			if truncated := buildTree(builder, filepath.Join(dir, entry.Name()), maxDepth, currentDepth+1, ignoreSet, budget); truncated {
				return true
			}
		} else {
			builder.WriteByte('\n')
		}
	}
	return false
}

// shouldIgnore checks if a name matches any ignore pattern.
func shouldIgnore(name string, ignoreSet map[string]bool) bool {
	if ignoreSet[name] {
		return true
	}
	for pattern := range ignoreSet {
		if strings.ContainsAny(pattern, "*?[") {
			if matched, _ := filepath.Match(pattern, name); matched {
				return true
			}
		}
	}
	return false
}

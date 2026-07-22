package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func registerGitDiff(r *Registry) {
	r.add(Tool{
		Name:        "git_diff",
		Category:    "Git Operations",
		Description: "View git diff output (working tree, staged, or between commits)",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"cached":{"type":"boolean","description":"Show staged changes (--cached)"},"commit1":{"type":"string","description":"Base ref"},"commit2":{"type":"string","description":"Target ref"},"path":{"type":"string","description":"Limit diff to a specific path"}},"additionalProperties":false}`),
		Execute:     gitDiff,
	})
}

func gitDiff(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Cached     bool   `json:"cached"`
		Commit1    string `json:"commit1"`
		Commit2    string `json:"commit2"`
		Path       string `json:"path"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "cached": true, "commit1": true, "commit2": true, "path": true,
	}); err != nil {
		return nil, err
	}

	args := []string{"diff"}
	if a.Cached {
		args = append(args, "--cached")
	}
	if a.Commit1 != "" {
		if err := validateRef(a.Commit1); err != nil {
			return nil, err
		}
		args = append(args, a.Commit1)
	}
	if a.Commit2 != "" {
		if a.Commit1 == "" {
			return nil, fmt.Errorf("commit1 is required when commit2 is provided")
		}
		if err := validateRef(a.Commit2); err != nil {
			return nil, err
		}
		args = append(args, a.Commit2)
	}
	if a.Path != "" {
		args = append(args, "--", a.Path)
	}

	diff, err := runGit(ctx, a.WorkingDir, args...)
	if err != nil {
		return nil, err
	}
	return map[string]any{"diff": diff}, nil
}

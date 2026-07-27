package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schemas/git_diff.json
var gitDiffSchema []byte

func registerGitDiff(r *Registry) {
	r.add(Tool{
		Name:        "git_diff",
		Category:    "Git Operations",
		Description: "View git diff output (working tree, staged, or between commits)",
		Schema:      json.RawMessage(gitDiffSchema),
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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func registerGitMerge(r *Registry) {
	r.add(Tool{
		Name:        "git_merge",
		Category:    "Git Operations",
		Description: "Merge a branch into the current branch",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"branch":{"type":"string","description":"Branch to merge"},"no_ff":{"type":"boolean","description":"Create merge commit even if fast-forward possible"},"squash":{"type":"boolean","description":"Squash commits"},"no_commit":{"type":"boolean","description":"Merge without auto-commit"},"message":{"type":"string","description":"Custom merge commit message"}},"required":["branch"],"additionalProperties":false}`),
		Execute:     gitMerge,
	})
}

func gitMerge(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Branch     string `json:"branch"`
		NoFF       bool   `json:"no_ff"`
		Squash     bool   `json:"squash"`
		NoCommit   bool   `json:"no_commit"`
		Message    string `json:"message"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "branch": true, "no_ff": true,
		"squash": true, "no_commit": true, "message": true,
	}); err != nil {
		return nil, err
	}
	if a.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if err := validateRef(a.Branch); err != nil {
		return nil, err
	}

	args := []string{"merge"}
	if a.NoFF {
		args = append(args, "--no-ff")
	}
	if a.Squash {
		args = append(args, "--squash")
	}
	if a.NoCommit {
		args = append(args, "--no-commit")
	}
	if a.Message != "" {
		args = append(args, "-m", a.Message)
	}
	args = append(args, a.Branch)

	if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "message": fmt.Sprintf("merged '%s' successfully", a.Branch)}, nil
}

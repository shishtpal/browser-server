package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schemas/git_pull.json
var gitPullSchema []byte

func registerGitPull(r *Registry) {
	r.add(Tool{
		Name:        "git_pull",
		Category:    "Git Operations",
		Description: "Pull changes from a remote repository",
		Schema:      json.RawMessage(gitPullSchema),
		Execute:     gitPull,
	})
}

func gitPull(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Remote     string `json:"remote"`
		Branch     string `json:"branch"`
		Rebase     bool   `json:"rebase"`
		FFOnly     bool   `json:"ff_only"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "remote": true, "branch": true,
		"rebase": true, "ff_only": true,
	}); err != nil {
		return nil, err
	}
	if a.Remote == "" {
		a.Remote = "origin"
	}
	if err := validateRef(a.Remote); err != nil {
		return nil, err
	}

	args := []string{"pull"}
	if a.Rebase {
		args = append(args, "--rebase")
	}
	if a.FFOnly {
		args = append(args, "--ff-only")
	}
	args = append(args, a.Remote)
	if a.Branch != "" {
		if err := validateRef(a.Branch); err != nil {
			return nil, err
		}
		args = append(args, a.Branch)
	}

	if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "message": fmt.Sprintf("pulled from %s", a.Remote)}, nil
}

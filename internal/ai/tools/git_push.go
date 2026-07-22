package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func registerGitPush(r *Registry) {
	r.add(Tool{
		Name:        "git_push",
		Category:    "Git Operations",
		Description: "Push commits to a remote repository",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"remote":{"type":"string","description":"Remote name (default: origin)"},"branch":{"type":"string","description":"Branch to push"},"set_upstream":{"type":"boolean","description":"Set upstream tracking (-u)"},"force":{"type":"boolean","description":"Force push (uses --force-with-lease)"},"tags":{"type":"boolean","description":"Push tags"}},"additionalProperties":false}`),
		Execute:     gitPush,
	})
}

func gitPush(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir  string `json:"working_dir"`
		Remote      string `json:"remote"`
		Branch      string `json:"branch"`
		SetUpstream bool   `json:"set_upstream"`
		Force       bool   `json:"force"`
		Tags        bool   `json:"tags"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "remote": true, "branch": true,
		"set_upstream": true, "force": true, "tags": true,
	}); err != nil {
		return nil, err
	}
	if a.Remote == "" {
		a.Remote = "origin"
	}
	if err := validateRef(a.Remote); err != nil {
		return nil, err
	}

	args := []string{"push"}
	if a.SetUpstream {
		args = append(args, "-u")
	}
	if a.Force {
		args = append(args, "--force-with-lease")
	}
	if a.Tags {
		args = append(args, "--tags")
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
	msg := fmt.Sprintf("pushed to %s", a.Remote)
	if a.Branch != "" {
		msg = fmt.Sprintf("pushed %s to %s", a.Branch, a.Remote)
	}
	return map[string]any{"success": true, "message": msg}, nil
}

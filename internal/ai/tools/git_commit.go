package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func registerGitCommit(r *Registry) {
	r.add(Tool{
		Name:        "git_commit",
		Category:    "Git Operations",
		Description: "Create a git commit, optionally staging files first",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"message":{"type":"string","description":"Commit message (required unless amend)"},"add":{"type":"array","items":{"type":"string"},"description":"Files to stage before committing"},"all":{"type":"boolean","description":"Stage all tracked changes"},"amend":{"type":"boolean","description":"Amend the previous commit"},"allow_empty":{"type":"boolean","description":"Allow empty commit"}},"additionalProperties":false}`),
		Execute:     gitCommit,
	})
}

func gitCommit(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string   `json:"working_dir"`
		Message    string   `json:"message"`
		Add        []string `json:"add"`
		All        bool     `json:"all"`
		Amend      bool     `json:"amend"`
		AllowEmpty bool     `json:"allow_empty"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "message": true, "add": true,
		"all": true, "amend": true, "allow_empty": true,
	}); err != nil {
		return nil, err
	}
	if a.Message == "" && !a.Amend {
		return nil, fmt.Errorf("message is required (unless amending)")
	}

	// Stage files if requested
	if a.All {
		if _, err := runGit(ctx, a.WorkingDir, "add", "-A"); err != nil {
			return nil, fmt.Errorf("failed to stage: %w", err)
		}
	} else if len(a.Add) > 0 {
		args := append([]string{"add", "--"}, a.Add...)
		if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
			return nil, fmt.Errorf("failed to stage: %w", err)
		}
	}

	// Build commit command
	args := []string{"commit"}
	if a.Message != "" {
		args = append(args, "-m", a.Message)
	}
	if a.Amend {
		args = append(args, "--amend")
		if a.Message == "" {
			args = append(args, "--no-edit")
		}
	}
	if a.AllowEmpty {
		args = append(args, "--allow-empty")
	}

	if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
		return nil, err
	}

	// Read back HEAD info
	info, err := runGit(ctx, a.WorkingDir, "log", "-1", "--pretty=format:%H%x00%s%x00%an%x00%aI")
	if err != nil {
		return map[string]any{"success": true, "message": "committed"}, nil
	}
	parts := strings.SplitN(strings.TrimSpace(info), "\x00", 4)
	if len(parts) < 4 {
		return map[string]any{"success": true, "message": "committed"}, nil
	}
	return map[string]any{
		"sha":     parts[0],
		"message": parts[1],
		"author":  parts[2],
		"date":    parts[3],
	}, nil
}

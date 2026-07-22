package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func registerGitCheckout(r *Registry) {
	r.add(Tool{
		Name:        "git_checkout",
		Category:    "Git Operations",
		Description: "Switch to a branch or create and switch to a new branch",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"branch":{"type":"string","description":"Branch to switch to or create"},"create":{"type":"boolean","description":"Create new branch (-b)"},"force":{"type":"boolean","description":"Force checkout"}},"required":["branch"],"additionalProperties":false}`),
		Execute:     gitCheckout,
	})
}

func gitCheckout(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Branch     string `json:"branch"`
		Create     bool   `json:"create"`
		Force      bool   `json:"force"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "branch": true, "create": true, "force": true,
	}); err != nil {
		return nil, err
	}
	if a.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if err := validateRef(a.Branch); err != nil {
		return nil, err
	}

	args := []string{"checkout"}
	if a.Create && a.Force {
		args = append(args, "-B", a.Branch)
	} else if a.Create {
		args = append(args, "-b", a.Branch)
	} else {
		if a.Force {
			args = append(args, "-f")
		}
		args = append(args, a.Branch)
	}

	if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "message": fmt.Sprintf("switched to '%s'", a.Branch)}, nil
}

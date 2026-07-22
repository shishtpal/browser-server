package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func registerGitBranch(r *Registry) {
	r.add(Tool{
		Name:        "git_branch",
		Category:    "Git Operations",
		Description: "Manage git branches: list, create, delete, or rename",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"operation":{"type":"string","enum":["list","create","delete","rename"],"description":"Branch operation (default: list)"},"name":{"type":"string","description":"Branch name (required for create/delete/rename)"},"new_name":{"type":"string","description":"New name (required for rename)"},"start_point":{"type":"string","description":"Start point for create"},"force":{"type":"boolean","description":"Force delete (-D)"},"all":{"type":"boolean","description":"Include remote branches in list"}},"required":["operation"],"additionalProperties":false}`),
		Execute:     gitBranch,
	})
}

func gitBranch(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Operation  string `json:"operation"`
		Name       string `json:"name"`
		NewName    string `json:"new_name"`
		StartPoint string `json:"start_point"`
		Force      bool   `json:"force"`
		All        bool   `json:"all"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "operation": true, "name": true, "new_name": true,
		"start_point": true, "force": true, "all": true,
	}); err != nil {
		return nil, err
	}

	switch a.Operation {
	case "list", "":
		args := []string{"branch", "--format=%(HEAD)|%(refname:short)|%(upstream:short)"}
		if a.All {
			args = append(args, "-a")
		}
		output, err := runGit(ctx, a.WorkingDir, args...)
		if err != nil {
			return nil, err
		}

		type branchInfo struct {
			Name     string `json:"name"`
			Current  bool   `json:"current"`
			IsRemote bool   `json:"is_remote"`
			Tracking string `json:"tracking,omitempty"`
		}
		var branches []branchInfo
		for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 3)
			if len(parts) < 3 {
				continue
			}
			branches = append(branches, branchInfo{
				Name:     parts[1],
				Current:  parts[0] == "*",
				IsRemote: strings.HasPrefix(parts[1], "remotes/"),
				Tracking: parts[2],
			})
		}
		return map[string]any{"branches": branches}, nil

	case "create":
		if a.Name == "" {
			return nil, fmt.Errorf("name is required for create")
		}
		if err := validateRef(a.Name); err != nil {
			return nil, err
		}
		args := []string{"branch", a.Name}
		if a.StartPoint != "" {
			if err := validateRef(a.StartPoint); err != nil {
				return nil, err
			}
			args = append(args, a.StartPoint)
		}
		if _, err := runGit(ctx, a.WorkingDir, args...); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "message": fmt.Sprintf("branch '%s' created", a.Name)}, nil

	case "delete":
		if a.Name == "" {
			return nil, fmt.Errorf("name is required for delete")
		}
		if err := validateRef(a.Name); err != nil {
			return nil, err
		}
		flag := "-d"
		if a.Force {
			flag = "-D"
		}
		if _, err := runGit(ctx, a.WorkingDir, "branch", flag, a.Name); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "message": fmt.Sprintf("branch '%s' deleted", a.Name)}, nil

	case "rename":
		if a.Name == "" || a.NewName == "" {
			return nil, fmt.Errorf("name and new_name are required for rename")
		}
		if err := validateRef(a.Name); err != nil {
			return nil, err
		}
		if err := validateRef(a.NewName); err != nil {
			return nil, err
		}
		if _, err := runGit(ctx, a.WorkingDir, "branch", "-m", a.Name, a.NewName); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "message": fmt.Sprintf("branch renamed from '%s' to '%s'", a.Name, a.NewName)}, nil

	default:
		return nil, fmt.Errorf("unknown operation %q; use list, create, delete, or rename", a.Operation)
	}
}

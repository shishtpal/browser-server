package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func registerGitLog(r *Registry) {
	r.add(Tool{
		Name:        "git_log",
		Category:    "Git Operations",
		Description: "View git commit history with optional filtering by branch, path, date range, or author",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":50,"description":"Max commits (default 20)"},"branch":{"type":"string"},"path":{"type":"string"},"since":{"type":"string","description":"ISO date"},"until":{"type":"string","description":"ISO date"},"author":{"type":"string"}},"additionalProperties":false}`),
		Execute:     gitLog,
	})
}

func gitLog(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
		Limit      int    `json:"limit"`
		Branch     string `json:"branch"`
		Path       string `json:"path"`
		Since      string `json:"since"`
		Until      string `json:"until"`
		Author     string `json:"author"`
	}
	if err := strict(raw, &a, map[string]bool{
		"working_dir": true, "limit": true, "branch": true, "path": true,
		"since": true, "until": true, "author": true,
	}); err != nil {
		return nil, err
	}

	if a.Limit == 0 {
		a.Limit = 20
	}
	if a.Limit < 1 || a.Limit > 50 {
		return nil, fmt.Errorf("limit must be between 1 and 50")
	}

	args := []string{"log",
		fmt.Sprintf("--max-count=%d", a.Limit),
		"--pretty=format:%H%x00%s%x00%an%x00%ae%x00%aI",
	}
	if a.Since != "" {
		args = append(args, "--since="+a.Since)
	}
	if a.Until != "" {
		args = append(args, "--until="+a.Until)
	}
	if a.Author != "" {
		args = append(args, "--author="+a.Author)
	}
	if a.Branch != "" {
		if err := validateRef(a.Branch); err != nil {
			return nil, err
		}
		args = append(args, a.Branch)
	}
	if a.Path != "" {
		args = append(args, "--", a.Path)
	}

	output, err := runGit(ctx, a.WorkingDir, args...)
	if err != nil {
		return nil, err
	}

	type commitEntry struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		Email   string `json:"email"`
		Date    string `json:"date"`
	}

	var commits []commitEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 5)
		if len(parts) < 5 {
			continue
		}
		commits = append(commits, commitEntry{
			SHA:     parts[0],
			Message: parts[1],
			Author:  parts[2],
			Email:   parts[3],
			Date:    parts[4],
		})
	}
	return map[string]any{"commits": commits}, nil
}

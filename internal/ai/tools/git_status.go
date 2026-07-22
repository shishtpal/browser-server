package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func registerGitStatus(r *Registry) {
	r.add(Tool{
		Name:        "git_status",
		Category:    "Git Operations",
		Description: "Check the git repository status: current branch, staged/unstaged changes, untracked files, ahead/behind remote",
		Schema:      json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string","description":"Repository path. Defaults to the server binary directory."}},"additionalProperties":false}`),
		Execute:     gitStatus,
	})
}

func gitStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
	}
	if err := strict(raw, &a, map[string]bool{"working_dir": true}); err != nil {
		return nil, err
	}

	branch, _ := runGit(ctx, a.WorkingDir, "branch", "--show-current")
	branch = strings.TrimSpace(branch)
	if branch == "" {
		head, _ := runGit(ctx, a.WorkingDir, "rev-parse", "--short", "HEAD")
		head = strings.TrimSpace(head)
		if head != "" {
			branch = "HEAD detached at " + head
		}
	}

	output, err := runGit(ctx, a.WorkingDir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	type fileChange struct {
		Path   string `json:"path"`
		Status string `json:"status"`
	}

	var staged, unstaged []fileChange
	var untracked []string
	isDirty := false

	for _, line := range strings.Split(output, "\n") {
		if len(line) < 3 {
			continue
		}
		x, y := line[0], line[1]
		file := strings.TrimSpace(line[3:])

		if x == '?' && y == '?' {
			untracked = append(untracked, file)
			isDirty = true
			continue
		}
		if x != ' ' && x != '?' {
			staged = append(staged, fileChange{Path: file, Status: gitStatusChar(x)})
			isDirty = true
		}
		if y != ' ' && y != '?' {
			unstaged = append(unstaged, fileChange{Path: file, Status: gitStatusChar(y)})
			isDirty = true
		}
	}

	aheadBy, behindBy := 0, 0
	ab, err := runGit(ctx, a.WorkingDir, "rev-list", "--count", "--left-right", "@{u}...HEAD")
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(ab))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &behindBy)
			fmt.Sscanf(parts[1], "%d", &aheadBy)
		}
	}

	return map[string]any{
		"branch":    branch,
		"is_dirty":  isDirty,
		"staged":    staged,
		"unstaged":  unstaged,
		"untracked": untracked,
		"ahead_by":  aheadBy,
		"behind_by": behindBy,
	}, nil
}

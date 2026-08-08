package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/ai/config"
)

//go:embed schemas/git_status.json
var gitStatusSchema []byte

type gitStatusResult struct {
	Branch    string       `json:"branch"`
	IsDirty   bool         `json:"is_dirty"`
	Staged    []fileChange `json:"staged"`
	Unstaged  []fileChange `json:"unstaged"`
	Untracked []string     `json:"untracked"`
	AheadBy   int          `json:"ahead_by"`
	BehindBy  int          `json:"behind_by"`
	Raw       string       `json:"-"`
}

type fileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func registerGitStatus(r *Registry, paths config.PathsConfig) {
	r.add(Tool{
		Name:        "git_status",
		Category:    "Git Operations",
		Description: "Check the git repository status: current branch, staged/unstaged changes, untracked files, ahead/behind remote",
		Schema:      json.RawMessage(gitStatusSchema),
		Execute:     gitStatus(paths),
		RawContentFunc: func(value any) ([]byte, bool) {
			result, ok := value.(gitStatusResult)
			return []byte(result.Raw), ok
		},
	})
}

func gitStatus(paths config.PathsConfig) func(ctx context.Context, raw json.RawMessage) (any, error) {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var a struct {
			WorkingDir string `json:"working_dir"`
		}
		if err := strict(raw, &a, map[string]bool{"working_dir": true}); err != nil {
			return nil, err
		}

		branch, _ := runGit(ctx, a.WorkingDir, paths, "branch", "--show-current")
		branch = strings.TrimSpace(branch)
		if branch == "" {
			head, _ := runGit(ctx, a.WorkingDir, paths, "rev-parse", "--short", "HEAD")
			head = strings.TrimSpace(head)
			if head != "" {
				branch = "HEAD detached at " + head
			}
		}

		output, err := runGit(ctx, a.WorkingDir, paths, "status", "--porcelain=v1", "--branch")
		if err != nil {
			return nil, err
		}

		var staged, unstaged []fileChange
		var untracked []string
		isDirty := false

		for _, line := range strings.Split(output, "\n") {
			if len(line) < 3 || strings.HasPrefix(line, "## ") {
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
		ab, err := runGit(ctx, a.WorkingDir, paths, "rev-list", "--count", "--left-right", "@{u}...HEAD")
		if err == nil {
			parts := strings.Fields(strings.TrimSpace(ab))
			if len(parts) == 2 {
				fmt.Sscanf(parts[0], "%d", &behindBy)
				fmt.Sscanf(parts[1], "%d", &aheadBy)
			}
		}

		return gitStatusResult{
			Branch:    branch,
			IsDirty:   isDirty,
			Staged:    staged,
			Unstaged:  unstaged,
			Untracked: untracked,
			AheadBy:   aheadBy,
			BehindBy:  behindBy,
			Raw:       output,
		}, nil
	}
}

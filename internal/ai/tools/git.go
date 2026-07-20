package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitTimeout = 30 * time.Second

// validateRef rejects ref names that start with '-' to prevent them being
// interpreted as git flags.
func validateRef(ref string) error {
	if strings.HasPrefix(strings.TrimSpace(ref), "-") {
		return fmt.Errorf("ref name %q cannot start with '-'", ref)
	}
	return nil
}

// runGit executes a git command with discrete arguments (safe from injection).
// If dir is empty, defaults to the server binary's directory.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if dir == "" {
		ex, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("cannot determine working directory: %w", err)
		}
		dir = filepath.Dir(ex)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out after %v", gitTimeout)
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	out := stdout.String()
	if len(out) > maxOutput {
		out = out[:maxOutput] + "\n... (output truncated)"
	}
	return out, nil
}

// --- git_status ---

func gitStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		WorkingDir string `json:"working_dir"`
	}
	if err := strict(raw, &a, map[string]bool{"working_dir": true}); err != nil {
		return nil, err
	}

	// Current branch (empty on detached HEAD)
	branch, _ := runGit(ctx, a.WorkingDir, "branch", "--show-current")
	branch = strings.TrimSpace(branch)
	if branch == "" {
		// Detached HEAD — show the short SHA
		head, _ := runGit(ctx, a.WorkingDir, "rev-parse", "--short", "HEAD")
		head = strings.TrimSpace(head)
		if head != "" {
			branch = "HEAD detached at " + head
		}
	}

	// Porcelain v1 status
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

	// Ahead/behind (may fail if no upstream — that's fine)
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

func gitStatusChar(c byte) string {
	switch c {
	case 'A':
		return "added"
	case 'M':
		return "modified"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	default:
		return "unknown"
	}
}

// --- git_diff ---

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

// --- git_log ---

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

	// NUL-separated format avoids ambiguity with pipes/delimiters in messages
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

// --- git_branch ---

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

// --- git_checkout ---

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

// --- git_commit ---

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

// --- git_push ---

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

// --- git_pull ---

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

// --- git_merge ---

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

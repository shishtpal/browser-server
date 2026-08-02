package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"browser-server/internal/ai/config"
)

// truncMarker is appended when git output is cut at the byte limit.
const truncMarker = "\n... (output truncated)"

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
func runGit(ctx context.Context, dir string, paths config.PathsConfig, args ...string) (string, error) {
	return runGitWithLimit(ctx, dir, limitsFrom(ctx).gitMaxOutput, paths, args...)
}

// runGitWithLimit executes a git command and truncates the output to the
// provided byte limit (0 means no truncation). This lets git_diff use a
// diff-specific limit while other git tools share the general max_output.
func runGitWithLimit(ctx context.Context, dir string, limit int, paths config.PathsConfig, args ...string) (string, error) {
	if dir == "" {
		ex, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("cannot determine working directory: %w", err)
		}
		dir = filepath.Dir(ex)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, limitsFrom(ctx).gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, resolveBinary("git", paths), args...)
	cmd.Dir = dir
	if env := childEnv(paths); env != nil {
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out after %v", limitsFrom(ctx).gitTimeout)
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	out := stdout.String()
	if limit > 0 && len(out) > limit {
		// Reserve room for the marker so the returned output stays within the
		// configured byte limit instead of exceeding it by the marker length.
		budget := limit - len(truncMarker)
		if budget < 0 {
			budget = 0
		}
		out = truncateUTF8(out, budget) + truncMarker
	}
	return out, nil
}

// gitStatusChar converts a git status byte to a human-readable string.
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

package tools

import (
	"bytes"
	"context"
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

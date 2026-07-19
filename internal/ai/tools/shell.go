package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	maxCommandTimeout = 30 * time.Second
	maxOutputBytes    = 64 * 1024
)

// ShellInfo describes the shell the server is running under so that the AI
// model can generate commands appropriate for the user's active terminal.
type ShellInfo struct {
	Name     string `json:"name"`     // e.g. "powershell", "bash", "cmd", "zsh"
	Platform string `json:"platform"` // runtime.GOOS
}

// DetectShell identifies the parent shell that launched the server process.
// It checks the SHELL and ComSpec environment variables and the parent process
// name. The result is baked into the tool description so the LLM knows which
// shell syntax to use.
func DetectShell() ShellInfo {
	info := ShellInfo{Platform: runtime.GOOS}

	// On Windows, check PSModulePath (set inside PowerShell sessions) first,
	// then fall back to ComSpec.
	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			info.Name = "powershell"
			return info
		}
		comspec := os.Getenv("ComSpec")
		if comspec != "" {
			lower := strings.ToLower(comspec)
			if strings.Contains(lower, "cmd.exe") {
				info.Name = "cmd"
			} else {
				info.Name = "powershell"
			}
			return info
		}
		info.Name = "powershell"
		return info
	}

	// Unix: prefer SHELL env
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := strings.ToLower(shell)
		switch {
		case strings.Contains(base, "zsh"):
			info.Name = "zsh"
		case strings.Contains(base, "fish"):
			info.Name = "fish"
		case strings.Contains(base, "bash"):
			info.Name = "bash"
		default:
			info.Name = "bash"
		}
		return info
	}

	info.Name = "bash"
	return info
}

func executeCommand(shell ShellInfo) func(context.Context, json.RawMessage) (any, error) {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var a struct {
			Command    string `json:"command"`
			WorkingDir string `json:"working_dir"`
			TimeoutSec int    `json:"timeout_seconds"`
		}
		if err := strict(raw, &a, map[string]bool{"command": true, "working_dir": true, "timeout_seconds": true}); err != nil {
			return nil, err
		}
		a.Command = strings.TrimSpace(a.Command)
		if a.Command == "" {
			return nil, fmt.Errorf("command is required")
		}
		if len(a.Command) > 4096 {
			return nil, fmt.Errorf("command exceeds 4096 characters")
		}

		timeout := 10 * time.Second
		if a.TimeoutSec > 0 {
			timeout = time.Duration(a.TimeoutSec) * time.Second
		}
		if timeout > maxCommandTimeout {
			timeout = maxCommandTimeout
		}

		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var cmd *exec.Cmd
		switch shell.Name {
		case "powershell":
			cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", a.Command)
		case "cmd":
			cmd = exec.CommandContext(cmdCtx, "cmd", "/C", a.Command)
		default:
			// bash, zsh, fish, etc.
			cmd = exec.CommandContext(cmdCtx, shell.Name, "-c", a.Command)
		}

		if a.WorkingDir != "" {
			cmd.Dir = a.WorkingDir
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		outBytes := stdout.Bytes()
		errBytes := stderr.Bytes()

		// Truncate output if too large
		stdoutTruncated := false
		stderrTruncated := false
		if len(outBytes) > maxOutputBytes {
			outBytes = outBytes[:maxOutputBytes]
			stdoutTruncated = true
		}
		if len(errBytes) > maxOutputBytes {
			errBytes = errBytes[:maxOutputBytes]
			stderrTruncated = true
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else if cmdCtx.Err() == context.DeadlineExceeded {
				return map[string]any{
					"exit_code": -1,
					"stdout":    string(outBytes),
					"stderr":    string(errBytes),
					"error":     fmt.Sprintf("command timed out after %v", timeout),
					"timed_out": true,
				}, nil
			} else {
				return nil, fmt.Errorf("failed to execute command: %v", err)
			}
		}

		result := map[string]any{
			"exit_code": exitCode,
			"stdout":    string(outBytes),
			"stderr":    string(errBytes),
		}
		if stdoutTruncated {
			result["stdout_truncated"] = true
		}
		if stderrTruncated {
			result["stderr_truncated"] = true
		}
		return result, nil
	}
}

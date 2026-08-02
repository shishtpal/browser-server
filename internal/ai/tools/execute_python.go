package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"browser-server/internal/ai/config"
)

//go:embed schemas/execute_python.json
var executePythonSchema []byte

const (
	maxPythonCodeLen     = 32768
	maxPythonPackages    = 20
	maxPythonTimeout     = 120 * time.Second
	defaultPythonTimeout = 30 * time.Second
	pythonStdoutCap      = 32 * 1024
	pythonStderrCap      = 16 * 1024
)

func registerExecutePython(r *Registry, paths config.PathsConfig) {
	r.add(Tool{
		Name:           "execute_python",
		Category:       "Process Management",
		Description:    "Execute Python 3 code via uv and return stdout, stderr, and exit code. Use for math, data processing, parsing, or file inspection. Rules: state is NOT preserved between calls; print() anything you need to see; put third-party packages in \"packages\" (installed on the fly, cached between runs); finish within the timeout.",
		Schema:         json.RawMessage(executePythonSchema),
		Execute:        executePython(paths),
		RawContentFunc: rawPythonResult,
	})
}

// limitWriter keeps consuming output (so the child never blocks on a full
// pipe) but stores only the first max bytes.
type limitWriter struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (w *limitWriter) Write(p []byte) (int, error) {
	n := len(p)
	remaining := w.max - w.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			w.buf.Write(p[:remaining])
		} else {
			w.buf.Write(p)
		}
	}
	if n > remaining {
		w.truncated = true
	}
	return n, nil
}

func (w *limitWriter) String() string {
	s := w.buf.String()
	if w.truncated {
		s += "\n...[output truncated]"
	}
	return s
}

// pythonResult is the JSON-serializable result of a Python execution.
type pythonResult struct {
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	ExitCode        int    `json:"exit_code"`
	TimedOut        bool   `json:"timed_out"`
	DurationMs      int64  `json:"duration_ms"`
	StdoutTruncated bool   `json:"stdout_truncated,omitempty"`
	StderrTruncated bool   `json:"stderr_truncated,omitempty"`
}

func executePython(paths config.PathsConfig) func(ctx context.Context, raw json.RawMessage) (any, error) {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var a struct {
			Code        string   `json:"code"`
			Packages    []string `json:"packages"`
			WorkingDir  string   `json:"working_dir"`
			TimeoutSecs int      `json:"timeout_seconds"`
		}
		if err := strict(raw, &a, map[string]bool{
			"code": true, "packages": true,
			"working_dir": true, "timeout_seconds": true,
		}); err != nil {
			return nil, err
		}

		a.Code = strings.TrimSpace(a.Code)
		if a.Code == "" {
			return nil, fmt.Errorf("code is required")
		}
		if len(a.Code) > maxPythonCodeLen {
			return nil, fmt.Errorf("code exceeds %d characters", maxPythonCodeLen)
		}
		if len(a.Packages) > maxPythonPackages {
			return nil, fmt.Errorf("packages exceeds %d entries", maxPythonPackages)
		}
		for _, p := range a.Packages {
			if strings.TrimSpace(p) == "" {
				return nil, fmt.Errorf("package names must not be empty")
			}
		}

		timeout := defaultPythonTimeout
		if a.TimeoutSecs > 0 {
			timeout = time.Duration(a.TimeoutSecs) * time.Second
		}
		if timeout > maxPythonTimeout {
			timeout = maxPythonTimeout
		}

		workDir := a.WorkingDir
		if workDir == "" {
			workDir = os.TempDir()
		}
		if abs, err := filepath.Abs(workDir); err == nil {
			workDir = abs
		}
		if st, err := os.Stat(workDir); err != nil || !st.IsDir() {
			return nil, fmt.Errorf("working_dir does not exist or is not a directory")
		}

		// Build: uv run --no-project --quiet [--with pkg]... -- python -I -c <code>
		//
		//   --no-project  don't pick up a pyproject.toml from workDir
		//   --quiet       keep uv's resolver chatter out of stderr
		//   --with        ephemeral deps, resolved and cached by uv
		//   --            everything after this goes to python, not uv
		//   -I            python isolated mode (ignores PYTHON* env vars, user site-packages)
		args := []string{"run", "--no-project", "--quiet"}
		for _, p := range a.Packages {
			args = append(args, "--with", p)
		}
		args = append(args, "--", "python", "-I", "-c", a.Code)

		// On Windows, uv.exe is expected on PATH; on Unix, uv.
		uvBinary := "uv"
		if runtime.GOOS == "windows" {
			uvBinary = "uv.exe"
		}
		uvBinary = resolveBinary(uvBinary, paths)

		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		cmd := exec.CommandContext(runCtx, uvBinary, args...)
		cmd.Dir = workDir
		if env := childEnv(paths); env != nil {
			cmd.Env = env
		}

		stdout := &limitWriter{max: pythonStdoutCap}
		stderr := &limitWriter{max: pythonStderrCap}
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		start := time.Now()
		err := cmd.Run()

		res := pythonResult{
			Stdout:          stdout.String(),
			Stderr:          stderr.String(),
			DurationMs:      time.Since(start).Milliseconds(),
			StdoutTruncated: stdout.truncated,
			StderrTruncated: stderr.truncated,
		}

		switch {
		case runCtx.Err() == context.DeadlineExceeded:
			res.TimedOut = true
			res.ExitCode = -1
			res.Stderr += fmt.Sprintf("\nkilled after %s (timeout)", timeout)
		case err == nil:
			res.ExitCode = 0
		default:
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				res.ExitCode = exitErr.ExitCode() // Python exception → non-zero
			} else {
				res.ExitCode = -1
				res.Stderr += "\nfailed to launch uv: " + err.Error()
			}
		}
		return res, nil
	}
}

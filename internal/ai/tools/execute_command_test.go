package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

// TestExecuteCommandFindsConfiguredBinaryInChildPath is a regression test for
// the gap where a binary mapped in paths.binaries (and absent from the
// inherited PATH) was invisible to shell commands run by execute_command.
// childEnv now prepends the mapped binary's parent directory to the child
// PATH, so the plain basename command resolves without rewriting the command
// string.
func TestExecuteCommandFindsConfiguredBinaryInChildPath(t *testing.T) {
	dir := t.TempDir()
	marker := "configured-binary-marker-xyz"

	fixture := filepath.Join(dir, "configured-tool")
	switch runtime.GOOS {
	case "windows":
		fixture += ".cmd"
		if err := os.WriteFile(fixture, []byte("@echo "+marker+"\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	default:
		if err := os.WriteFile(fixture, []byte("#!/bin/sh\necho "+marker+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		// os.WriteFile honors only the permission bits; ensure it is executable
		// on Unix even if umask stripped owner-execute.
		if err := os.Chmod(fixture, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	shell := DetectShell()
	shellProbe := shell.Name
	if runtime.GOOS == "windows" {
		shellProbe = "powershell.exe"
		if shell.Name == "cmd" {
			shellProbe = "cmd.exe"
		}
	}
	if _, err := exec.LookPath(shellProbe); err != nil {
		t.Skipf("shell %q unavailable: %v", shell.Name, err)
	}

	paths := config.PathsConfig{
		Binaries: map[string]string{"configured-tool": fixture},
	}

	res, err := executeCommand(shell, paths)(context.Background(), json.RawMessage(`{"command":"configured-tool"}`))
	if err != nil {
		t.Fatalf("executeCommand returned error: %v", err)
	}
	result, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", res)
	}
	code, _ := result["exit_code"].(int)
	stdout, _ := result["stdout"].(string)
	stderr, _ := result["stderr"].(string)
	if code != 0 {
		t.Fatalf("exit_code = %d, want 0 (stdout=%q stderr=%q)", code, stdout, stderr)
	}
	if !strings.Contains(stdout, marker) {
		t.Fatalf("stdout %q missing marker %q", stdout, marker)
	}
	lowerStderr := strings.ToLower(stderr)
	if strings.Contains(lowerStderr, "not recognized") || strings.Contains(lowerStderr, "command not found") {
		t.Fatalf("stderr indicates executable lookup failure: %q", stderr)
	}
}

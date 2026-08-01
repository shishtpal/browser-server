package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// uvAvailable reports whether uv is installed and on PATH.
func uvAvailable() bool {
	if _, err := exec.LookPath("uv"); err == nil {
		return true
	}
	if err := exec.Command("uv", "--version").Run(); err == nil {
		return true
	}
	if err := exec.Command("uv.exe", "--version").Run(); err == nil {
		return true
	}
	return false
}

func TestExecutePythonRejectsEmptyCode(t *testing.T) {
	r := New()
	for _, raw := range []string{`{}`, `{"code":""}`, `{"code":"   "}`} {
		if _, err := r.Execute(context.Background(), "execute_python", []byte(raw)); err == nil {
			t.Fatalf("execute_python(%s) succeeded, want error for empty code", raw)
		}
	}
}

func TestExecutePythonRejectsUnknownArgs(t *testing.T) {
	r := New()
	if _, err := r.Execute(context.Background(), "execute_python", []byte(`{"code":"1+1","unknown":true}`)); err == nil {
		t.Fatal("expected unknown argument rejection")
	}
}

func TestExecutePythonRejectsTooManyPackages(t *testing.T) {
	r := New()
	pkgs := make([]string, maxPythonPackages+1)
	for i := range pkgs {
		pkgs[i] = "pkg"
	}
	raw, _ := json.Marshal(map[string]any{"code": "print(1)", "packages": pkgs})
	if _, err := r.Execute(context.Background(), "execute_python", raw); err == nil {
		t.Fatal("expected too many packages rejection")
	}
}

func TestExecutePythonRunsSimpleCode(t *testing.T) {
	if !uvAvailable() {
		t.Skip("uv not installed, skipping integration test")
	}
	r := New()
	result, err := r.Execute(context.Background(), "execute_python", []byte(`{"code":"print(6 * 7)","timeout_seconds":60}`))
	if err != nil {
		t.Fatal(err)
	}
	var out pythonResult
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Stdout, "42") {
		t.Fatalf("stdout = %q, want it to contain 42", out.Stdout)
	}
	if out.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0; stderr=%s", out.ExitCode, out.Stderr)
	}
}

func TestExecutePythonReportsException(t *testing.T) {
	if !uvAvailable() {
		t.Skip("uv not installed, skipping integration test")
	}
	r := New()
	result, err := r.Execute(context.Background(), "execute_python", []byte(`{"code":"raise ValueError('boom')","timeout_seconds":60}`))
	if err != nil {
		t.Fatal(err)
	}
	var out pythonResult
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatal(err)
	}
	if out.ExitCode == 0 {
		t.Fatal("expected non-zero exit code for exception")
	}
	if !strings.Contains(out.Stderr, "ValueError") {
		t.Fatalf("stderr = %q, want it to mention ValueError", out.Stderr)
	}
}

func TestExecutePythonInvalidWorkingDir(t *testing.T) {
	r := New()
	_, err := r.Execute(context.Background(), "execute_python", []byte(`{"code":"print(1)","working_dir":"/nonexistent/path/that/does/not/exist"}`))
	if err == nil {
		t.Fatal("expected error for nonexistent working_dir")
	}
}

func TestExecutePythonIsRegistered(t *testing.T) {
	r := New()
	specs := r.Specs([]string{"execute_python"})
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec for execute_python, got %d", len(specs))
	}
}

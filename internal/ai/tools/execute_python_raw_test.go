package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRawPythonResultSuccess(t *testing.T) {
	res := pythonResult{
		Stdout:     "hello world\n",
		Stderr:     "",
		ExitCode:   0,
		TimedOut:   false,
		DurationMs: 179,
	}
	raw, ok := rawPythonResult(res)
	if !ok {
		t.Fatal("rawPythonResult returned ok=false for pythonResult")
	}

	want := "# stderr: \"\"\n# exit_code: 0\n# timed_out: false\n# duration_ms: 179\n\nhello world\n"
	if string(raw) != want {
		t.Fatalf("raw output mismatch\nwant:\n%s\ngot:\n%s", want, string(raw))
	}
}

func TestRawPythonResultEscapesStderr(t *testing.T) {
	res := pythonResult{
		Stdout:     "out",
		Stderr:     "line1\nline2\t\"quoted\"\r\nend",
		ExitCode:   0,
		TimedOut:   false,
		DurationMs: 50,
	}
	raw, ok := rawPythonResult(res)
	if !ok {
		t.Fatal("rawPythonResult returned ok=false")
	}

	lines := strings.Split(string(raw), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "# stderr: ") {
		t.Fatalf("expected first line to be stderr comment, got %q", lines[0])
	}
	// strings.Split removes the trailing newline from the first element; verify
	// the stderr comment itself contains no literal newline characters.
	if strings.Contains(lines[0], "\n") {
		t.Fatalf("stderr line should not contain embedded newline: %q", lines[0])
	}
	// Ensure newlines inside stderr were escaped as \n, not literal newlines.
	if strings.Count(lines[0], "line1") != 1 {
		t.Fatalf("unexpected escaped stderr line: %q", lines[0])
	}
	wantSubstr := `# stderr: "line1\nline2\t\"quoted\"\r\nend"`
	if lines[0] != wantSubstr {
		t.Fatalf("stderr comment mismatch\nwant: %q\ngot:  %q", wantSubstr, lines[0])
	}
}

func TestRawPythonResultFailureAndTruncation(t *testing.T) {
	res := pythonResult{
		Stdout:          "partial out",
		Stderr:          "partial err",
		ExitCode:        1,
		TimedOut:        true,
		DurationMs:      999,
		StdoutTruncated: true,
		StderrTruncated: true,
	}
	raw, ok := rawPythonResult(res)
	if !ok {
		t.Fatal("rawPythonResult returned ok=false")
	}

	wantHeader := []string{
		"# stderr: \"partial err\"",
		"# exit_code: 1",
		"# timed_out: true",
		"# duration_ms: 999",
		"# stdout_truncated: true",
		"# stderr_truncated: true",
		"",
	}
	got := strings.Split(string(raw), "\n")
	if len(got) < len(wantHeader)+1 {
		t.Fatalf("got %d lines, want at least %d", len(got), len(wantHeader)+1)
	}
	for i, want := range wantHeader {
		if got[i] != want {
			t.Fatalf("line %d mismatch: want %q, got %q", i, want, got[i])
		}
	}
}

func TestRawPythonResultUnsupportedValue(t *testing.T) {
	_, ok := rawPythonResult(map[string]any{"stdout": "x"})
	if ok {
		t.Fatal("expected ok=false for unsupported type")
	}
}

func TestRawPythonResultPreservesStdoutLineEndings(t *testing.T) {
	res := pythonResult{
		Stdout:     "line1\r\nline2\r\n",
		Stderr:     "",
		ExitCode:   0,
		TimedOut:   false,
		DurationMs: 10,
	}
	raw, ok := rawPythonResult(res)
	if !ok {
		t.Fatal("rawPythonResult returned ok=false")
	}
	// Header separator is a single bare LF; the stdout payload keeps CRLF.
	if !bytes.HasSuffix(raw, []byte("\r\n")) {
		t.Fatalf("expected stdout to end with CRLF, got %q", string(raw))
	}
	wantTail := []byte("\n\nline1\r\nline2\r\n")
	if !bytes.HasSuffix(raw, wantTail) {
		t.Fatalf("expected tail %q, got %q", wantTail, string(raw[len(raw)-len(wantTail):]))
	}
}

func TestRegistryExecutePythonRawOverride(t *testing.T) {
	// Use a fake/stub execute_python that returns a deterministic result.
	stubResult := pythonResult{
		Stdout:     "42\n",
		Stderr:     "",
		ExitCode:   0,
		TimedOut:   false,
		DurationMs: 7,
	}
	stubExecute := func(ctx context.Context, raw json.RawMessage) (any, error) {
		return stubResult, nil
	}

	r := New()
	// Replace the registered execute_python with our stub to avoid needing uv.
	stubTool := r.tools["execute_python"]
	stubTool.Execute = stubExecute
	r.tools["execute_python"] = stubTool

	ctx := context.Background()
	trueBool := true
	falseBool := false

	// JSON by default.
	jsonOut, err := r.Execute(ctx, "execute_python", []byte(`{"code":"print(42)"}`))
	if err != nil {
		t.Fatalf("default execute failed: %v", err)
	}
	if !strings.HasPrefix(string(jsonOut), "{") {
		t.Fatalf("expected JSON output by default, got %q", string(jsonOut))
	}
	var parsed pythonResult
	if err := json.Unmarshal(jsonOut, &parsed); err != nil {
		t.Fatalf("default output is not valid JSON: %v", err)
	}
	if parsed.Stdout != "42\n" {
		t.Fatalf("default JSON stdout mismatch: got %q", parsed.Stdout)
	}

	// Forced raw.
	rawCtx := WithRawOutputOverride(ctx, &trueBool)
	rawOut, err := r.Execute(rawCtx, "execute_python", []byte(`{"code":"print(42)"}`))
	if err != nil {
		t.Fatalf("raw execute failed: %v", err)
	}
	if strings.HasPrefix(string(rawOut), "{") {
		t.Fatalf("expected raw output in raw mode, got %q", string(rawOut))
	}
	if !strings.HasPrefix(string(rawOut), "# stderr: \"\"\n") {
		t.Fatalf("expected raw header, got %q", string(rawOut))
	}

	// Forced JSON override should disable raw even for a tool with RawContentFunc.
	jsonCtx := WithRawOutputOverride(ctx, &falseBool)
	jsonOut2, err := r.Execute(jsonCtx, "execute_python", []byte(`{"code":"print(42)"}`))
	if err != nil {
		t.Fatalf("forced JSON execute failed: %v", err)
	}
	if !strings.HasPrefix(string(jsonOut2), "{") {
		t.Fatalf("expected JSON output when override=false, got %q", string(jsonOut2))
	}
}

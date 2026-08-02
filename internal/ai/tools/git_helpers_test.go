package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

// skipIfNoGit skips the test when git is not available on PATH.
func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
}

// TestRunGitWithLimitKeepsOutputWithinLimit verifies that truncated git output
// stays within the byte limit (the truncation marker is reserved for) and that
// the marker is not appended when the output already fits.
func TestRunGitWithLimitKeepsOutputWithinLimit(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	runIn := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	runIn("init", "-q")
	runIn("config", "user.email", "test@example.com")
	runIn("config", "user.name", "Test")

	file := filepath.Join(dir, "file.txt")
	const lines = 5000
	orig := make([]string, lines)
	mod := make([]string, lines)
	for i := 0; i < lines; i++ {
		orig[i] = fmt.Sprintf("line %d original\n", i)
		mod[i] = fmt.Sprintf("line %d modified\n", i)
	}
	content1 := strings.Join(orig, "")
	content2 := strings.Join(mod, "")
	if err := os.WriteFile(file, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}
	runIn("add", ".")
	runIn("commit", "-q", "-m", "initial")
	if err := os.WriteFile(file, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	// Limit smaller than the diff -> output truncated to <= limit with marker.
	limit := 1024
	out, err := runGitWithLimit(context.Background(), dir, limit, config.PathsConfig{}, "diff")
	if err != nil {
		t.Fatalf("runGitWithLimit: %v", err)
	}
	if len(out) > limit {
		t.Fatalf("truncated output length = %d, want <= %d", len(out), limit)
	}
	if !strings.Contains(out, truncMarker) {
		t.Fatalf("truncated output missing marker %q", truncMarker)
	}

	// Large limit -> output returned unchanged without the marker.
	out2, err := runGitWithLimit(context.Background(), dir, 100*1024*1024, config.PathsConfig{}, "diff")
	if err != nil {
		t.Fatalf("runGitWithLimit (large limit): %v", err)
	}
	if strings.Contains(out2, truncMarker) {
		t.Fatal("untruncated output must not contain the truncation marker")
	}
	if len(out2) <= len(content1) {
		t.Fatalf("untruncated output length = %d, want larger diff", len(out2))
	}
}

// TestExecuteGitDiffEndToEndHonorsDiffLimit reproduces the reported bug: a
// diff larger than tools.max_output but smaller than tools.max_diff_output was
// rejected with "tool output exceeds limit" because Registry.Execute validated
// the final result against max_output. The diff must now be returned up to
// max_diff_output and truncated (not rejected) beyond it.
func TestExecuteGitDiffEndToEndHonorsDiffLimit(t *testing.T) {
	skipIfNoGit(t)
	dir := t.TempDir()
	runIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runIn("init", "-q")
	runIn("config", "user.email", "test@example.com")
	runIn("config", "user.name", "Test")

	file := filepath.Join(dir, "file.txt")
	write := func(lines []string) {
		t.Helper()
		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	const n = 700 // ~30 KB diff, above max_output, below max_diff_output
	orig := make([]string, n)
	mod := make([]string, n)
	for i := 0; i < n; i++ {
		orig[i] = fmt.Sprintf("line %d original text", i)
		mod[i] = fmt.Sprintf("line %d modified text", i)
	}
	write(orig)
	runIn("add", ".")
	runIn("commit", "-q", "-m", "initial")
	write(mod)

	args := []byte(`{"working_dir":` + quoted(dir) + `}`)

	// Diff within max_diff_output: must succeed with full raw content.
	r := New(Options{
		Tools: config.ToolsConfig{
			MaxOutput:     10240,
			MaxDiffOutput: 51200,
			RawOutput:     []string{"git_diff"},
		},
	})
	out, err := r.Execute(context.Background(), "git_diff", args)
	if err != nil {
		t.Fatalf("git_diff e2e within max_diff_output must succeed, got: %v", err)
	}
	if len(out) <= 10240 {
		t.Fatalf("diff output length = %d, want > max_output to prove the diff limit is honored", len(out))
	}
	if !strings.Contains(string(out), "diff --git") {
		t.Fatalf("output does not look like a diff: %.80q", out)
	}

	// Diff beyond max_diff_output: truncated to the cap, not an error. Rewrite
	// every line of a 3000-line file so the diff is well past 51200 bytes.
	const bigN = 3000
	bigOrig := make([]string, bigN)
	bigMod := make([]string, bigN)
	for i := 0; i < bigN; i++ {
		bigOrig[i] = fmt.Sprintf("big original line %d", i)
		bigMod[i] = fmt.Sprintf("big modified line %d", i)
	}
	write(bigOrig)
	runIn("add", ".")
	runIn("commit", "-q", "-m", "big")
	write(bigMod)
	r2 := New(Options{
		Tools: config.ToolsConfig{
			MaxOutput:     10240,
			MaxDiffOutput: 51200,
			RawOutput:     []string{"git_diff"},
		},
	})
	out2, err := r2.Execute(context.Background(), "git_diff", args)
	if err != nil {
		t.Fatalf("git_diff e2e over max_diff_output must truncate, not error, got: %v", err)
	}
	if len(out2) > 51200 {
		t.Fatalf("truncated diff output length = %d, want <= 51200", len(out2))
	}
	if !strings.Contains(string(out2), truncMarker) {
		t.Fatalf("truncated diff output missing marker %q", truncMarker)
	}
}

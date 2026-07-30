package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// defaultTestTool returns a fileReadTool with default settings for testing.
func defaultTestTool() *fileReadTool {
	return &fileReadTool{
		maxReadBytes:      32 * 1024,
		maxLineReadBytes:  64 * 1024,
		maxLineCount:      5000,
		maxFileSizeWarnMB: 100,
	}
}

func TestReadFileBasic(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "basic.txt")
	if err := os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "line one\nline two\nline three" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
	if m["line_count"] != 3 {
		t.Fatalf("line_count = %v, want 3", m["line_count"])
	}
}

func TestReadFileWithOffsetAndLimit(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "offset.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\ne\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path, "offset": 2, "limit": 2})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "b\nc" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
	if m["line_count"] != 2 {
		t.Fatalf("line_count = %v, want 2", m["line_count"])
	}
}

func TestReadFileWithRanges(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "ranges.txt")
	content := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{
		"path": path,
		"ranges": []map[string]int{
			{"offset": 2, "limit": 3},
			{"offset": 7, "limit": 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "2\n3\n4\n7\n8" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
	if m["line_count"] != 5 {
		t.Fatalf("line_count = %v, want 5", m["line_count"])
	}
}

func TestReadFileWithLineNumbers(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "linenum.txt")
	content := "first\nsecond\nthird\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path, "line_numbers": true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	expected := "1: first\n2: second\n3: third"
	if m["content"] != expected {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], expected)
	}
}

func TestReadFileLineNumbersWithRanges(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "linenum_ranges.txt")
	content := "a\nb\nc\nd\ne\nf\ng\nh\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{
		"path": path,
		"ranges": []map[string]int{
			{"offset": 2, "limit": 2},
			{"offset": 5, "limit": 3},
		},
		"line_numbers": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	expected := "2: b\n3: c\n5: e\n6: f\n7: g"
	if m["content"] != expected {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], expected)
	}
}

func TestReadFileRangesMutuallyExclusiveWithOffsetLimit(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "excl.txt")
	os.WriteFile(path, []byte("x\ny\nz\n"), 0644)
	args, err := json.Marshal(map[string]any{"path": path, "ranges": []map[string]int{{"offset": 1, "limit": 1}}, "offset": 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error when ranges and offset are both specified")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadFileRangeOffsetOneBased(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "onebased.txt")
	if err := os.WriteFile(path, []byte("only line\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path, "offset": 1, "limit": 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "only line" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
}

func TestReadFileRangeBeyondFileEnd(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "short.txt")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path, "offset": 5, "limit": 10})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "" {
		t.Fatalf("expected empty content for offset beyond file, got %q", m["content"])
	}
}

func TestReadFileInvalidRange(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "inv.txt")
	os.WriteFile(path, []byte("x\ny\n"), 0644)
	args, err := json.Marshal(map[string]any{"path": path, "ranges": []map[string]int{{"offset": 0, "limit": 1}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for offset < 1")
	}
}

func TestReadFileNoTrailingNewline(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "notrail.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "hello world" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
	if m["line_count"] != 1 {
		t.Fatalf("line_count = %v, want 1", m["line_count"])
	}
}

func TestReadFileLineNumbersNoTrailingNewline(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "nl_notrail.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path, "line_numbers": true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	expected := "1: hello world"
	if m["content"] != expected {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], expected)
	}
}

// --- New tests for the fixes ---

func TestReadFileNotFound(t *testing.T) {
	ft := defaultTestTool()
	args, err := json.Marshal(map[string]any{"path": "C:\\nonexistent\\file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("expected friendly 'file not found' error, got: %v", err)
	}
}

func TestReadFileBlockedPattern(t *testing.T) {
	ft := defaultTestTool()
	ft.blockedPatterns = []string{"**/.env*"}
	path := filepath.Join(t.TempDir(), ".env.local")
	os.WriteFile(path, []byte("SECRET=1\n"), 0644)
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for blocked path pattern")
	}
	if !strings.Contains(err.Error(), "blocked by path pattern") {
		t.Fatalf("expected 'blocked by path pattern' error, got: %v", err)
	}
}

func TestReadFileAllowedExtensions(t *testing.T) {
	ft := defaultTestTool()
	ft.allowedExts = []string{".go", ".txt"}
	path := filepath.Join(t.TempDir(), "test.exe")
	os.WriteFile(path, []byte("binary??\n"), 0644)
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for disallowed extension")
	}
	if !strings.Contains(err.Error(), "extension not in allowed list") {
		t.Fatalf("expected 'extension not in allowed list' error, got: %v", err)
	}
}

func TestReadFileExceedsFullReadLimit(t *testing.T) {
	ft := defaultTestTool()
	ft.maxReadBytes = 100
	path := filepath.Join(t.TempDir(), "large.txt")
	data := make([]byte, 200)
	for i := range data {
		data[i] = 'A' + byte(i%26)
	}
	data[199] = '\n'
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ft.readFile(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for file exceeding full-read limit")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected 'exceeds read limit' error, got: %v", err)
	}
}

func TestReadFileWithOffsetBeyondFileLength(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "slim.txt")
	os.WriteFile(path, []byte("only one\n"), 0644)
	args, err := json.Marshal(map[string]any{"path": path, "offset": 10, "limit": 5})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "" {
		t.Fatalf("expected empty content, got %q", m["content"])
	}
	if m["line_count"] != 0 {
		t.Fatalf("expected 0 lines, got %v", m["line_count"])
	}
}

func TestReadFileWindowsPathForwardSlash(t *testing.T) {
	// Simulate a path like /D:/foo/bar — on Windows we strip the leading /
	ft := defaultTestTool()
	// Create a real file and test with forward-slash absolute path
	realPath := filepath.Join(t.TempDir(), "winpath.txt")
	os.WriteFile(realPath, []byte("content\n"), 0644)

	// Build a forward-slash style absolute path like /D:/path/to/file
	abs := filepath.ToSlash(realPath)
	if len(abs) > 0 && abs[0] != '/' {
		abs = "/" + abs
	}
	args, err := json.Marshal(map[string]any{"path": abs})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "content" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
}

func TestReadFileEmpty(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "empty.txt")
	os.WriteFile(path, []byte{}, 0644)
	args, err := json.Marshal(map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["content"] != "" {
		t.Fatalf("expected empty content, got %q", m["content"])
	}
	if m["line_count"] != 0 {
		t.Fatalf("line_count = %v, want 0", m["line_count"])
	}
}

func TestReadFileMultipleRangesOverlap(t *testing.T) {
	ft := defaultTestTool()
	path := filepath.Join(t.TempDir(), "overlap.txt")
	content := "a\nb\nc\nd\ne\n"
	os.WriteFile(path, []byte(content), 0644)
	args, err := json.Marshal(map[string]any{
		"path": path,
		"ranges": []map[string]int{
			{"offset": 1, "limit": 3},
			{"offset": 2, "limit": 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFile(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	// Duplicate line numbers should be deduplicated; order should match first occurrence
	if m["content"] != "a\nb\nc" {
		t.Fatalf("unexpected content: %q", m["content"])
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		val, min, max, def, want int
	}{
		{0, 10, 100, 50, 50},    // zero → default
		{5, 10, 100, 50, 10},    // below min → min
		{200, 10, 100, 50, 100}, // above max → max
		{42, 10, 100, 50, 42},   // in range → as-is
	}
	for _, tc := range tests {
		got := clamp(tc.val, tc.min, tc.max, tc.def)
		if got != tc.want {
			t.Errorf("clamp(%d,%d,%d,%d) = %d, want %d", tc.val, tc.min, tc.max, tc.def, got, tc.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}
	for _, tc := range tests {
		got := formatBytes(tc.bytes)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

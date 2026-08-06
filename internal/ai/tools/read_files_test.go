package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runReadFiles(t *testing.T, ft *fileReadTool, args map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ft.readFiles(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	return result.(map[string]any)
}

func TestReadFilesBasicMultiple(t *testing.T) {
	ft := defaultTestTool()
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.txt")
	p2 := filepath.Join(dir, "b.txt")
	os.WriteFile(p1, []byte("alpha one\nalpha two\n"), 0644)
	os.WriteFile(p2, []byte("beta one\n"), 0644)

	m := runReadFiles(t, ft, map[string]any{"files": []string{p1, p2}})
	want := "# File: " + p1 + "\nalpha one\nalpha two\n\n# File: " + p2 + "\nbeta one"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}
	if m["files_read"] != 2 || m["files_total"] != 2 {
		t.Fatalf("files_read=%v files_total=%v, want 2/2", m["files_read"], m["files_total"])
	}
}

func TestReadFilesOffsetLimitSuffix(t *testing.T) {
	ft := defaultTestTool()
	p := filepath.Join(t.TempDir(), "nums.txt")
	os.WriteFile(p, []byte("1\n2\n3\n4\n5\n"), 0644)

	spec := filepath.ToSlash(p) + ":2:2"
	m := runReadFiles(t, ft, map[string]any{"files": []string{spec}})
	want := "# File: " + filepath.ToSlash(p) + ":2:2\n2\n3"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}
}

func TestReadFilesStartEndSuffix(t *testing.T) {
	ft := defaultTestTool()
	p := filepath.Join(t.TempDir(), "nums.txt")
	os.WriteFile(p, []byte("1\n2\n3\n4\n5\n"), 0644)

	spec := filepath.ToSlash(p) + ":2-4"
	m := runReadFiles(t, ft, map[string]any{"files": []string{spec}})
	want := "# File: " + filepath.ToSlash(p) + ":2-4\n2\n3\n4"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}
}

func TestReadFilesLineNumbersToggle(t *testing.T) {
	ft := defaultTestTool()
	p := filepath.Join(t.TempDir(), "ln.txt")
	os.WriteFile(p, []byte("a\nb\nc\n"), 0644)

	spec := filepath.ToSlash(p) + ":2-3"
	m := runReadFiles(t, ft, map[string]any{"files": []string{spec}, "line_numbers": true})
	want := "# File: " + filepath.ToSlash(p) + ":2-3\n2: b\n3: c"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}

	m = runReadFiles(t, ft, map[string]any{"files": []string{spec}, "line_numbers": false})
	want = "# File: " + filepath.ToSlash(p) + ":2-3\nb\nc"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}
}

func TestReadFilesMissingFileInlineError(t *testing.T) {
	ft := defaultTestTool()
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	os.WriteFile(good, []byte("ok\n"), 0644)
	bad := filepath.Join(dir, "missing.txt")

	m := runReadFiles(t, ft, map[string]any{"files": []string{bad, good}})
	content := m["content"].(string)
	if !strings.Contains(content, "# File: "+bad+"\nerror: file not found") {
		t.Fatalf("expected inline not-found error, got:\n%s", content)
	}
	if !strings.HasSuffix(content, "# File: "+good+"\nok") {
		t.Fatalf("expected good file to still be read, got:\n%s", content)
	}
	if m["files_read"] != 1 || m["files_total"] != 2 {
		t.Fatalf("files_read=%v files_total=%v, want 1/2", m["files_read"], m["files_total"])
	}
}

func TestReadFilesInvalidSpecInlineError(t *testing.T) {
	ft := defaultTestTool()
	p := filepath.Join(t.TempDir(), "x.txt")
	os.WriteFile(p, []byte("x\n"), 0644)

	m := runReadFiles(t, ft, map[string]any{"files": []string{filepath.ToSlash(p) + ":5-2"}})
	content := m["content"].(string)
	if !strings.Contains(content, "error: invalid range") {
		t.Fatalf("expected inline invalid-range error, got:\n%s", content)
	}
	if m["files_read"] != 0 {
		t.Fatalf("files_read=%v, want 0", m["files_read"])
	}
}

func TestReadFilesEmptyFiles(t *testing.T) {
	ft := defaultTestTool()
	raw, _ := json.Marshal(map[string]any{"files": []string{}})
	_, err := ft.readFiles(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected empty-files error, got: %v", err)
	}
}

func TestReadFilesTooMany(t *testing.T) {
	ft := defaultTestTool()
	files := make([]string, 21)
	for i := range files {
		files[i] = "f.txt"
	}
	raw, _ := json.Marshal(map[string]any{"files": files})
	_, err := ft.readFiles(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "maximum 20") {
		t.Fatalf("expected too-many-files error, got: %v", err)
	}
}

func TestReadFilesBlockedPattern(t *testing.T) {
	ft := defaultTestTool()
	ft.blockedPatterns = []string{"**/.env*"}
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("SECRET=1\n"), 0644)

	m := runReadFiles(t, ft, map[string]any{"files": []string{p}})
	if !strings.Contains(m["content"].(string), "blocked by path pattern") {
		t.Fatalf("expected blocked error, got:\n%s", m["content"])
	}
}

func TestReadFilesWindowsDriveLetterNotParsedAsRange(t *testing.T) {
	// A drive-letter path like D:/dir/file.txt must not be treated as having
	// a range suffix.
	spec, err := parseFileSpec(`D:\dir\file.txt`)
	if err != nil {
		t.Fatal(err)
	}
	if spec.hasRng {
		t.Fatalf("drive-letter path parsed as range: %+v", spec)
	}
	if spec.path != `D:\dir\file.txt` {
		t.Fatalf("path = %q", spec.path)
	}
}

func TestParseFileSpecForms(t *testing.T) {
	tests := []struct {
		in       string
		wantPath string
		wantRng  bool
		offset   int
		limit    int
		disp     string
		wantErr  bool
	}{
		{in: "a/b.txt", wantPath: "a/b.txt"},
		{in: "a/b.txt:3:2", wantPath: "a/b.txt", wantRng: true, offset: 3, limit: 2, disp: "3:2"},
		{in: "a/b.txt:3-7", wantPath: "a/b.txt", wantRng: true, offset: 3, limit: 5, disp: "3-7"},
		{in: "a/b.txt:0:5", wantErr: true},
		{in: "a/b.txt:5-2", wantErr: true},
		{in: "a/b.txt:1:2:3", wantErr: true},
		{in: "a/b.txt:abc", wantErr: true},
		{in: ":5:5", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tc := range tests {
		s, err := parseFileSpec(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseFileSpec(%q): expected error, got %+v", tc.in, s)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseFileSpec(%q): %v", tc.in, err)
			continue
		}
		if s.path != tc.wantPath || s.hasRng != tc.wantRng {
			t.Errorf("parseFileSpec(%q) = path %q rng %v, want path %q rng %v", tc.in, s.path, s.hasRng, tc.wantPath, tc.wantRng)
		}
		if tc.wantRng && (s.offset != tc.offset || s.limit != tc.limit || s.rngDisp != tc.disp) {
			t.Errorf("parseFileSpec(%q) = %d:%d disp %q, want %d:%d disp %q", tc.in, s.offset, s.limit, s.rngDisp, tc.offset, tc.limit, tc.disp)
		}
	}
}

func TestReadFilesMixedSpecs(t *testing.T) {
	ft := defaultTestTool()
	dir := t.TempDir()
	p1 := filepath.Join(dir, "full.txt")
	p2 := filepath.Join(dir, "ranged.txt")
	os.WriteFile(p1, []byte("full content\n"), 0644)
	os.WriteFile(p2, []byte("r1\nr2\nr3\n"), 0644)

	slash2 := filepath.ToSlash(p2) + ":2-3"
	m := runReadFiles(t, ft, map[string]any{"files": []string{p1, slash2}})
	want := "# File: " + p1 + "\nfull content\n\n# File: " + filepath.ToSlash(p2) + ":2-3\nr2\nr3"
	if m["content"] != want {
		t.Fatalf("unexpected content:\ngot  %q\nwant %q", m["content"], want)
	}
}

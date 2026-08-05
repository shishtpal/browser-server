package tools

import "testing"

func TestRawListDirectoryResult(t *testing.T) {
	raw, ok := rawListDirectoryResult(map[string]any{
		"path":      `D:\Codings\lang-Go\browser-server\internal\ai`,
		"truncated": false,
		"entries": []map[string]any{
			{"name": "api", "is_dir": true},
			{"name": "tools", "is_dir": true},
			{"name": "README.md", "is_dir": false},
		},
	})
	if !ok {
		t.Fatal("rawListDirectoryResult returned ok=false")
	}
	const want = "path=D:\\Codings\\lang-Go\\browser-server\\internal\\ai\ntruncated=false\ndirs=api,tools\nfiles=README.md"
	if string(raw) != want {
		t.Fatalf("raw output = %q, want %q", raw, want)
	}
}

func TestRawListDirectoryResultOmitsEmptyGroups(t *testing.T) {
	raw, ok := rawListDirectoryResult(map[string]any{
		"path":      ".",
		"truncated": true,
		"entries":   []map[string]any{{"name": "api", "is_dir": true}},
	})
	if !ok || string(raw) != "path=.\ntruncated=true\ndirs=api" {
		t.Fatalf("raw=%q ok=%t", raw, ok)
	}
}

func TestRawListDirectoryResultOmitsEmptyDirs(t *testing.T) {
	raw, ok := rawListDirectoryResult(map[string]any{
		"path":      ".",
		"truncated": false,
		"entries":   []map[string]any{{"name": "README.md", "is_dir": false}},
	})
	if !ok || string(raw) != "path=.\ntruncated=false\nfiles=README.md" {
		t.Fatalf("raw=%q ok=%t", raw, ok)
	}
}

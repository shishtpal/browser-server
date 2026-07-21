package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempCode(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSearchCodeLiteralFiltersAndContext(t *testing.T) {
	dir := t.TempDir()
	writeTempCode(t, dir, "main.go", "before\nAlpha alpha\nafter\n")
	writeTempCode(t, dir, "skip.txt", "alpha\n")
	value, err := searchCode(context.Background(), json.RawMessage(`{"pattern":"alpha","path":`+quoteJSON(dir)+`,"type":"literal","include":["*.go"],"case_sensitive":true,"context_lines":1}`))
	if err != nil {
		t.Fatal(err)
	}
	result := value.(map[string]any)
	if result["total_matches"].(int) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	encoded, _ := json.Marshal(result["matches"])
	var matches []struct {
		Line          int      `json:"line"`
		Column        int      `json:"column"`
		ContextBefore []string `json:"context_before"`
	}
	if err := json.Unmarshal(encoded, &matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Line != 2 || matches[0].Column != 7 || len(matches[0].ContextBefore) != 1 {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}

func TestAnalyzeCodeDirectoryAndPrivateFiltering(t *testing.T) {
	dir := t.TempDir()
	writeTempCode(t, dir, "one.go", "package sample\nimport \"fmt\"\n// Person is exported.\ntype Person struct { Name string `json:\"name\"` }\nfunc Public(v int) string { return fmt.Sprint(v) }\nfunc private() {}\n")
	writeTempCode(t, dir, "sub/two.go", "package sub\nconst Exported = 1\n")
	value, err := analyzeCode(context.Background(), json.RawMessage(`{"path":`+quoteJSON(dir)+`,"max_depth":2}`))
	if err != nil {
		t.Fatal(err)
	}
	result := value.(map[string]any)
	files := result["files"].([]map[string]any)
	if len(files) != 2 {
		t.Fatalf("expected per-file analyses, got %#v", result)
	}
	for _, file := range files {
		for _, symbol := range file["symbols"].([]map[string]any) {
			if symbol["name"] == "private" {
				t.Fatal("private symbol was included")
			}
		}
	}
}

func TestDiagnosticsParsingAndUnsupportedLanguage(t *testing.T) {
	d, ok := parseDiagnostic(`C:\work\main.go:12:7: undefined: value`, "error", "gobuild")
	if !ok || d.File != `C:\work\main.go` || d.Range.Start.Line != 11 || d.Range.Start.Character != 6 {
		t.Fatalf("Windows diagnostic not parsed as zero-based: %#v", d)
	}
	dir := t.TempDir()
	writeTempCode(t, dir, "main.ts", "const x = 1\n")
	if _, err := getDiagnostics(context.Background(), json.RawMessage(`{"path":`+quoteJSON(dir)+`,"language":"typescript"}`)); err == nil {
		t.Fatal("expected unsupported language error")
	}
}

func TestInvalidGlobsAreRejected(t *testing.T) {
	dir := t.TempDir()
	writeTempCode(t, dir, "one.go", "package one\n")
	if _, err := searchCode(context.Background(), json.RawMessage(`{"pattern":"x","path":`+quoteJSON(dir)+`,"include":["["]}`)); err == nil {
		t.Fatal("search accepted invalid glob")
	}
	if _, err := analyzeCode(context.Background(), json.RawMessage(`{"path":`+quoteJSON(dir)+`,"exclude_patterns":["["]}`)); err == nil {
		t.Fatal("analyze accepted invalid glob")
	}
}

func TestAnalyzeMaxResultsStillScansAllFiles(t *testing.T) {
	dir := t.TempDir()
	writeTempCode(t, dir, "one.go", "package sample\nimport \"fmt\"\ntype First struct{}\nfunc One(){fmt.Print()}\n")
	writeTempCode(t, dir, "two.go", "package sample\nimport \"strings\"\ntype Second string\nfunc Two(){_ = strings.TrimSpace(\"\")}\n")
	value, err := analyzeCode(context.Background(), json.RawMessage(`{"path":`+quoteJSON(dir)+`,"max_results":1}`))
	if err != nil {
		t.Fatal(err)
	}
	result := value.(map[string]any)
	summary := result["summary"].(map[string]any)
	if summary["files_analyzed"] != 2 || summary["total_functions"] != 2 || summary["total_types"] != 2 || summary["total_imports"] != 2 {
		t.Fatalf("inaccurate complete summary: %#v", summary)
	}
	if !result["truncated"].(bool) || len(result["files"].([]map[string]any)) != 2 {
		t.Fatalf("expected both file entries and symbol truncation: %#v", result)
	}
}

func TestSeverityThreshold(t *testing.T) {
	want := map[string][]bool{
		"error":   {true, false, false, false},
		"warning": {true, true, false, false},
		"info":    {true, true, true, false},
		"hint":    {true, true, true, true},
		"all":     {true, true, true, true},
	}
	levels := []string{"error", "warning", "info", "hint"}
	for threshold, expected := range want {
		for i, level := range levels {
			if got := severityAllowed(threshold, level); got != expected[i] {
				t.Errorf("%s/%s = %v", threshold, level, got)
			}
		}
	}
}

func TestCappedBufferExactTruncation(t *testing.T) {
	b := &cappedBuffer{limit: 4}
	_, _ = b.Write([]byte("1234"))
	if b.truncated {
		t.Fatal("exactly full buffer marked truncated")
	}
	_, _ = b.Write([]byte("5"))
	if !b.truncated || b.String() != "1234" {
		t.Fatalf("unexpected buffer: %q truncated=%v", b.String(), b.truncated)
	}
}

func TestCodeIntelligenceToolsRegistered(t *testing.T) {
	r := New()
	for _, name := range []string{"search_code", "analyze_code", "get_diagnostics"} {
		if _, ok := r.tools[name]; !ok {
			t.Errorf("%s is not registered", name)
		}
	}
}

func TestCodeToolsRejectUnknownArguments(t *testing.T) {
	if _, err := searchCode(context.Background(), json.RawMessage(`{"pattern":"x","unknown":true}`)); err == nil {
		t.Fatal("expected strict argument error")
	}
}

func quoteJSON(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

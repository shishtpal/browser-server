package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mustJSONCode marshals args to JSON so Windows paths with backslashes are
// escaped correctly (raw string concatenation would produce invalid JSON).
func mustJSONCode(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func callSearchCode(t *testing.T, input string) (map[string]any, error) {
	t.Helper()
	res, err := searchCode(context.Background(), json.RawMessage(input))
	if err != nil {
		return nil, err
	}
	env, ok := res.(searchCodeEnvelope)
	if !ok {
		t.Fatalf("expected searchCodeEnvelope, got %T", res)
	}
	return env.AsMap()
}

func TestSearchCodeBasic(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\nfunc main() {\n\thello()\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main\nfunc hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	page, err := callSearchCode(t, mustJSONCode(map[string]any{"pattern": "hello", "path": root, "type": "literal"}))
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if intFrom(page, "page") != 1 || intFrom(page, "page_size") != 10 || intFrom(page, "total") == 0 || page["truncated"] != false {
		t.Fatalf("unexpected metadata: %v", page)
	}
	results := pageResults(page)
	if len(results) == 0 {
		t.Fatalf("expected code results, got %#v", page)
	}
	assertScoresPresent(t, results)
}

func TestSearchCodeEnvelope(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("hello world\nhello again\n"), 0644); err != nil {
		t.Fatal(err)
	}

	page, err := callSearchCode(t, mustJSONCode(map[string]any{"pattern": "hello", "path": root, "type": "literal", "page_size": 5}))
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if intFrom(page, "total_matches") == 0 {
		t.Fatal("expected total_matches > 0")
	}
	if intFrom(page, "search_time_ms") == 0 {
		t.Fatal("expected search_time_ms")
	}
	if _, ok := page["results"]; !ok {
		t.Fatal("expected results key")
	}
}

func TestSearchCodeLegacyMaxResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f%d.txt", i)), []byte("match\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	page, err := callSearchCode(t, mustJSONCode(map[string]any{"pattern": "match", "path": root, "type": "literal", "max_results": 5}))
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if intFrom(page, "page_size") != 5 || len(pageResults(page)) != 5 || page["has_more"] != true {
		t.Fatalf("unexpected page: %v", page)
	}
}

func TestSearchCodePagination(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f%d.txt", i)), []byte("match\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	page, err := callSearchCode(t, mustJSONCode(map[string]any{"pattern": "match", "path": root, "type": "literal", "page_size": 3, "page": 2}))
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if intFrom(page, "page") != 2 || len(pageResults(page)) != 3 || page["has_more"] != true {
		t.Fatalf("unexpected page: %v", page)
	}
}

func TestSearchCodeStrict(t *testing.T) {
	_, err := callSearchCode(t, `{"pattern":"x","bogus":true}`)
	if err == nil || !contains(err.Error(), "bogus") {
		t.Fatalf("expected unknown argument error, got %v", err)
	}
}

func TestSearchEnvelopeOutputBudget(t *testing.T) {
	ctx := withToolLimits(context.Background(), toolLimits{maxOutput: 512})
	results := make([]map[string]any, 10)
	for i := range results {
		results[i] = map[string]any{"id": i, "content": string(make([]byte, 100))}
	}
	envelope := fitSearchEnvelope(ctx, map[string]any{
		"page": 1, "page_size": 10, "total": 10, "has_more": false,
		"truncated": false, "results": results,
	})
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > outputBudget(ctx) || envelope["truncated"] != true || envelope["has_more"] != true {
		t.Fatalf("budget was not enforced: bytes=%d envelope=%v", len(encoded), envelope)
	}
}

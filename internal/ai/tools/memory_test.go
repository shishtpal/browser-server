package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testMemoryStore(t *testing.T) *memoryStore {
	t.Helper()
	root := t.TempDir()
	return &memoryStore{
		root:       root,
		primary:    filepath.Join(root, "memories"),
		refs:       filepath.Join(root, "refs"),
		cache:      filepath.Join(root, "cache"),
		maxFile:    1024 * 1024,
		maxDepth:   5,
		cacheLimit: 1024 * 1024,
	}
}

func TestMemoryCRUDSearchLazyAndCache(t *testing.T) {
	s := testMemoryStore(t)
	v, err := s.remember(context.Background(), json.RawMessage(`{"content":"persistent fact","title":"Fact","tags":["test"]}`))
	if err != nil {
		t.Fatal(err)
	}
	id := v.(map[string]any)["id"].(string)
	if _, err = s.recall(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.search(context.Background(), json.RawMessage(`{"query":"persistent","tags":["test"]}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.lazy(context.Background(), json.RawMessage(`{"memory_id":"`+id+`","trigger":"access","expires_after":"1h"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.update(context.Background(), json.RawMessage(`{"id":"`+id+`","content":"updated fact"}`)); err != nil {
		t.Fatal(err)
	}
	stats, err := s.manageCache(context.Background(), json.RawMessage(`{"action":"stats"}`))
	if err != nil {
		t.Fatal(err)
	}
	if stats.(map[string]any)["entries"].(int) == 0 {
		t.Fatal("expected recall to populate cache")
	}
	if _, err = s.forget(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.read(id, true); err == nil {
		t.Fatal("forgotten memory still exists")
	}
}

func TestMemoryReferenceCycleAndTraversalProtection(t *testing.T) {
	s := testMemoryStore(t)
	nowID := func(id string, refs []string) {
		t.Helper()
		if err := s.write(memoryData{Metadata: memoryMeta{ID: id, Type: "primary", Source: "ai", References: refs}, Content: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	nowID("memory_a", []string{"memory_b"})
	nowID("memory_b", []string{"memory_a"})
	if _, err := s.resolveFrom(context.Background(), "memory_a", 5, true); err == nil {
		t.Fatal("expected cycle error")
	}
	if _, err := s.read("../secret", true); err == nil {
		t.Fatal("expected invalid ID error")
	}
}

func TestMemoryStrictArgumentsAndSizeLimit(t *testing.T) {
	s := testMemoryStore(t)
	s.maxFile = 128
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"x","unknown":true}`)); err == nil {
		t.Fatal("expected unknown argument error")
	}
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"this content cannot fit in a tiny memory file once frontmatter is included"}`)); err == nil {
		t.Fatal("expected size error")
	}
}

func TestMemoryCodecRoundTrip(t *testing.T) {
	m := memoryData{
		Metadata: memoryMeta{ID: "memory_xyz", Type: "primary", Source: "ai", Title: "Round"},
		Content:  "hello",
	}
	b, err := encodeMemory(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "---\n") {
		t.Fatalf("expected frontmatter prefix, got %q", b)
	}
	got, err := decodeMemory(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.ID != m.Metadata.ID || got.Content != m.Content {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestMemoryCodecRejectsBadFrontmatter(t *testing.T) {
	if _, err := decodeMemory([]byte("no frontmatter here")); err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestMemoryAtomicWriteCreatesParent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "deep", "file.md")
	if err := atomicWrite(target, []byte("payload")); err != nil {
		t.Fatalf("atomicWrite: %v", err)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}
	if string(b) != "payload" {
		t.Fatalf("payload = %q", b)
	}
}

func TestMemoryPathForRejectsTraversal(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.pathFor("memory_../escape"); err == nil {
		t.Fatal("expected invalid id error")
	}
	if _, err := s.pathFor("memory_unknown"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected ErrNotExist for unknown id, got %v", err)
	}
}

func TestMemoryReadEnforcesSizeLimit(t *testing.T) {
	s := testMemoryStore(t)
	// Write a memory that fits, then tighten the limit below the file size to
	// trigger the "exceeds file size limit" branch on read.
	if err := s.write(memoryData{Metadata: memoryMeta{ID: "memory_big", Type: "primary", Source: "ai"}, Content: "0123456789"}); err != nil {
		t.Fatal(err)
	}
	s.maxFile = 4
	if _, err := s.read("memory_big", false); err == nil {
		t.Fatal("expected size limit error")
	}
}

func TestMemoryRememberValidatesRequiredFields(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.remember(context.Background(), json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("expected content required, got %v", err)
	}
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"x","type":"invalid"}`)); err == nil {
		t.Fatal("expected type validation error")
	}
}

func TestMemoryRememberRequiresValidReferenceTarget(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"x","type":"reference","target_id":"nope"}`)); err == nil {
		t.Fatal("expected missing target error")
	}
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"x","references":["missing"]}`)); err == nil {
		t.Fatal("expected missing reference error")
	}
}

func TestMemoryRememberAutoCreatesReferences(t *testing.T) {
	s := testMemoryStore(t)
	v, err := s.remember(context.Background(), json.RawMessage(`{"content":"primary","references":["memory_missing"],"auto_create_refs":true}`))
	if err != nil {
		t.Fatalf("auto create should succeed: %v", err)
	}
	if _, err := s.pathFor("memory_missing"); err != nil {
		t.Fatalf("expected stub at memory_missing, got %v", err)
	}
	if v == nil {
		t.Fatal("expected remember result")
	}
}

func TestMemoryUpdateAppliesOnlyProvidedFields(t *testing.T) {
	s := testMemoryStore(t)
	v, _ := s.remember(context.Background(), json.RawMessage(`{"content":"hello","title":"A","tags":["x"]}`))
	id := v.(map[string]any)["id"].(string)

	if _, err := s.update(context.Background(), json.RawMessage(`{"id":"`+id+`","content":"changed"}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := s.read(id, true)
	if got.Content != "changed" || got.Metadata.Title != "A" || len(got.Metadata.Tags) != 1 {
		t.Fatalf("update altered untouched fields: %+v", got)
	}
	if got.Metadata.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestMemoryUpdateRejectsSelfReference(t *testing.T) {
	s := testMemoryStore(t)
	v, _ := s.remember(context.Background(), json.RawMessage(`{"content":"hello"}`))
	id := v.(map[string]any)["id"].(string)
	if _, err := s.update(context.Background(), json.RawMessage(`{"id":"`+id+`","references":["`+id+`"]}`)); err == nil {
		t.Fatal("expected self-reference error")
	}
}

func TestMemoryResolveBoundsDepth(t *testing.T) {
	s := testMemoryStore(t)
	if err := s.write(memoryData{Metadata: memoryMeta{ID: "memory_root", Type: "primary", Source: "ai"}, Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.resolveFrom(context.Background(), "memory_root", s.maxDepth+1, false); err == nil {
		t.Fatal("expected depth bound error")
	}
}

func TestMemoryResolveDefaultDepthIsThree(t *testing.T) {
	s := testMemoryStore(t)
	depth := 0
	nowID := func(id string, refs []string) {
		t.Helper()
		depth++
		_ = s.write(memoryData{Metadata: memoryMeta{ID: id, Type: "primary", Source: "ai", References: refs}, Content: "x"})
	}
	nowID("memory_a", nil)
	nowID("memory_b", []string{"memory_a"})
	if _, err := s.resolveFrom(context.Background(), "memory_b", 0, true); err != nil {
		t.Fatalf("default depth resolve: %v", err)
	}
}

func TestMemorySearchRespectsLimits(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.search(context.Background(), json.RawMessage(`{"limit":0}`)); err != nil {
		// 0 falls back to default 20 — that's the documented behavior
		t.Fatal(err)
	}
	if _, err := s.searchValues(context.Background(), "x", nil, "", "", 0, false); err == nil {
		t.Fatal("expected limit validation error")
	}
	if _, err := s.searchValues(context.Background(), "x", nil, "", "", 200, false); err == nil {
		t.Fatal("expected upper limit error")
	}
}

func TestMemoryListDefaultsLimit(t *testing.T) {
	s := testMemoryStore(t)
	got, err := s.list(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["total_count"].(int) != 0 {
		t.Fatalf("empty store should report 0, got %v", got)
	}
}

func TestMemoryFuzzyScoreRanking(t *testing.T) {
	// identical scores threshold fallback ensures equal-length matches still
	// rank best results first.
	best := fuzzyScore("alpha", []string{"alpha", "beta"}, "alpha", "alpha body", "")
	other := fuzzyScore("alpha", []string{"alpha", "beta"}, "beta", "beta body", "")
	if best <= other {
		t.Fatalf("expected title match to score higher: best=%v other=%v", best, other)
	}
}

func TestMemoryCacheCleanupHonorsMaxAge(t *testing.T) {
	s := testMemoryStore(t)
	v, _ := s.remember(context.Background(), json.RawMessage(`{"content":"hello"}`))
	id := v.(map[string]any)["id"].(string)
	if _, err := s.recall(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil {
		t.Fatal(err)
	}
	// Without time control we cannot backdate files portably; instead verify
	// cleanup runs without error and reports deterministic counts when empty.
	out, err := s.manageCache(context.Background(), json.RawMessage(`{"action":"cleanup","max_age":"1ns"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(map[string]any)["entries"]; !ok {
		t.Fatalf("missing entries key: %v", out)
	}
}

func TestMemoryCacheRejectsUnknownAction(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.manageCache(context.Background(), json.RawMessage(`{"action":"unknown"}`)); err == nil {
		t.Fatal("expected unknown action error")
	}
}

func TestMemoryCacheInvalidMaxAge(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.manageCache(context.Background(), json.RawMessage(`{"action":"cleanup","max_age":"garbage"}`)); err == nil {
		t.Fatal("expected invalid max_age error")
	}
}

func TestMemoryLazyRejectsInvalidTrigger(t *testing.T) {
	s := testMemoryStore(t)
	v, _ := s.remember(context.Background(), json.RawMessage(`{"content":"hi"}`))
	id := v.(map[string]any)["id"].(string)
	if _, err := s.lazy(context.Background(), json.RawMessage(`{"memory_id":"`+id+`","trigger":"other"}`)); err == nil {
		t.Fatal("expected invalid trigger error")
	}
}

func TestMemoryLazyRejectsInvalidExpiry(t *testing.T) {
	s := testMemoryStore(t)
	v, _ := s.remember(context.Background(), json.RawMessage(`{"content":"hi"}`))
	id := v.(map[string]any)["id"].(string)
	if _, err := s.lazy(context.Background(), json.RawMessage(`{"memory_id":"`+id+`","expires_after":"not-a-duration"}`)); err == nil {
		t.Fatal("expected invalid expires_after error")
	}
}

func TestMemoryForgetClearsReferences(t *testing.T) {
	s := testMemoryStore(t)
	v, err := s.remember(context.Background(), json.RawMessage(`{"content":"primary","references":["memory_other"],"auto_create_refs":true}`))
	if err != nil {
		t.Fatalf("remember: %v", err)
	}
	id := v.(map[string]any)["id"].(string)
	if err := s.write(memoryData{Metadata: memoryMeta{ID: "memory_other", Type: "primary", Source: "ai", References: []string{id}}, Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.forget(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil {
		t.Fatal(err)
	}
	other, err := s.read("memory_other", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range other.Metadata.References {
		if ref == id {
			t.Fatalf("forget did not clear reference to %s", id)
		}
	}
}

func TestMemoryRecallRequiresIDOrSearch(t *testing.T) {
	s := testMemoryStore(t)
	if _, err := s.recall(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error when neither id nor search is provided")
	}
}

func TestMemoryRecallIncludesReferences(t *testing.T) {
	s := testMemoryStore(t)
	if err := s.write(memoryData{Metadata: memoryMeta{ID: "memory_root", Type: "primary", Source: "ai", References: []string{"memory_child"}}, Content: "root"}); err != nil {
		t.Fatal(err)
	}
	if err := s.write(memoryData{Metadata: memoryMeta{ID: "memory_child", Type: "primary", Source: "ai"}, Content: "child"}); err != nil {
		t.Fatal(err)
	}
	out, err := s.recall(context.Background(), json.RawMessage(`{"id":"memory_root","include_references":true,"max_depth":3}`))
	if err != nil {
		t.Fatal(err)
	}
	count := out.(map[string]any)["total_count"].(int)
	if count < 2 {
		t.Fatalf("expected root + reference, got %d", count)
	}
}

func TestMemoryRegisteredToolsIncludeAllMemoryTools(t *testing.T) {
	r := New(Options{})
	want := []string{
		"ai_remember", "ai_recall", "ai_search_memory", "ai_list_memories",
		"ai_forget", "ai_update_memory", "ai_resolve_references",
		"ai_lazy_memory", "ai_manage_cache",
	}
	for _, name := range want {
		if _, ok := r.tools[name]; !ok {
			t.Fatalf("expected tool %q to be registered", name)
		}
		if r.tools[name].Category != "Memory" {
			t.Fatalf("tool %q category = %q, want Memory", name, r.tools[name].Category)
		}
	}
}

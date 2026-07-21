package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func testMemoryStore(t *testing.T) *memoryStore {
	t.Helper()
	root := t.TempDir()
	return &memoryStore{root: root, primary: filepath.Join(root, "memories"), refs: filepath.Join(root, "refs"), cache: filepath.Join(root, "cache"), maxFile: 1024 * 1024, maxDepth: 5, cacheLimit: 1024 * 1024}
}

func TestMemoryCRUDSearchLazyAndCache(t *testing.T) {
	s := testMemoryStore(t)
	v, err := s.remember(context.Background(), json.RawMessage(`{"content":"persistent fact","title":"Fact","tags":["test"]}`))
	if err != nil { t.Fatal(err) }
	id := v.(map[string]any)["id"].(string)
	if _, err = s.recall(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil { t.Fatal(err) }
	if _, err = s.search(context.Background(), json.RawMessage(`{"query":"persistent","tags":["test"]}`)); err != nil { t.Fatal(err) }
	if _, err = s.lazy(context.Background(), json.RawMessage(`{"memory_id":"`+id+`","trigger":"access","expires_after":"1h"}`)); err != nil { t.Fatal(err) }
	if _, err = s.update(context.Background(), json.RawMessage(`{"id":"`+id+`","content":"updated fact"}`)); err != nil { t.Fatal(err) }
	stats, err := s.manageCache(context.Background(), json.RawMessage(`{"action":"stats"}`)); if err != nil { t.Fatal(err) }
	if stats.(map[string]any)["entries"].(int) == 0 { t.Fatal("expected recall to populate cache") }
	if _, err = s.forget(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil { t.Fatal(err) }
	if _, err = s.read(id, true); err == nil { t.Fatal("forgotten memory still exists") }
}

func TestMemoryReferenceCycleAndTraversalProtection(t *testing.T) {
	s := testMemoryStore(t)
	nowID := func(id string, refs []string) { t.Helper(); if err := s.write(memoryData{Metadata: memoryMeta{ID:id, Type:"primary", Source:"ai", References:refs}, Content:"x"}); err != nil { t.Fatal(err) } }
	nowID("memory_a", []string{"memory_b"}); nowID("memory_b", []string{"memory_a"})
	if _, err := s.resolveFrom(context.Background(), "memory_a", 5, true); err == nil { t.Fatal("expected cycle error") }
	if _, err := s.read("../secret", true); err == nil { t.Fatal("expected invalid ID error") }
}

func TestMemoryStrictArgumentsAndSizeLimit(t *testing.T) {
	s := testMemoryStore(t); s.maxFile = 128
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"x","unknown":true}`)); err == nil { t.Fatal("expected unknown argument error") }
	if _, err := s.remember(context.Background(), json.RawMessage(`{"content":"this content cannot fit in a tiny memory file once frontmatter is included"}`)); err == nil { t.Fatal("expected size error") }
}

package memory

import (
	"context"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := config.MemoryConfig{
		Enabled: true, Directory: dir, FragmentsDir: "fragments", ArchiveDir: ".archive",
		MaxBodyKB: 64, MaxOpsPerCall: 20, MaxResultBytes: 8192,
		MaxDepth: 3, DefaultDepth: 1, SpreadFactor: 0.45,
		RetentionDays: 365, AutoCleanup: true, SecretScan: true,
		// Lowered from the 0.82 default: Jaccard character-trigram similarity
		// between paraphrased titles like "Memory system v2" and "Memory
		// System v2 design" is ~0.59, well below the production threshold.
		NearDuplicateThreshold: 0.5,
	}
	s := Open(dir, cfg)
	if !s.Enabled() {
		t.Fatal("expected enabled store")
	}
	return s
}

func TestEnsureRootBootstrapsPersona(t *testing.T) {
	s := testStore(t)
	s.mu.RLock()
	_, ok := s.idx.byID["mem_root"]
	s.mu.RUnlock()
	if !ok {
		t.Fatal("mem_root missing after bootstrap")
	}
	block := s.PersonaBlock(context.Background(), 900, true)
	if block == "" {
		t.Fatal("persona block empty")
	}
	if !strings.Contains(block, "mem_projects") {
		t.Fatalf("persona block missing projects index: %s", block)
	}
}

func TestUpsertCreateAndRecall(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{{
		Op: "upsert", ID: "mem_test_one", Kind: KindDecision, Title: "Test Decision",
		Summary: "A test decision about the memory system", Body: "Full body here.",
		Parent: "mem_projects", Tags: []string{"memory"},
	}}})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	res, err := s.Recall(context.Background(), RecallArgs{Query: "test decision", Limit: 5})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(res.Nodes) == 0 {
		t.Fatal("no nodes returned")
	}
	if res.Nodes[0].ID != "mem_test_one" {
		t.Fatalf("expected mem_test_one, got %s", res.Nodes[0].ID)
	}
	// detail=full loads body
	full, err := s.Recall(context.Background(), RecallArgs{IDs: []string{"mem_test_one"}, Detail: "full"})
	if err != nil {
		t.Fatalf("recall full: %v", err)
	}
	if len(full.Nodes) == 0 || !strings.Contains(full.Nodes[0].Body, "Full body") {
		t.Fatalf("expected body in full recall, got %+v", full.Nodes)
	}
	// reads touch access metadata
	got, ok := s.Get("mem_test_one")
	if !ok {
		t.Fatal("expected mem_test_one to exist")
	}
	if got.AccessCount == 0 || got.Accessed.IsZero() {
		t.Fatalf("expected access tracking after reads, got count=%d accessed=%v", got.AccessCount, got.Accessed)
	}
}

func TestRecallByIDsPreservesOrder(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{
		{Op: "upsert", ID: "mem_ord_a", Title: "A", Summary: "first", Parent: "mem_inbox"},
		{Op: "upsert", ID: "mem_ord_b", Title: "B", Summary: "second", Parent: "mem_projects"},
	}})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Request in reverse order; direct ids mode must honor caller order.
	res, err := s.Recall(context.Background(), RecallArgs{IDs: []string{"mem_ord_b", "mem_ord_a"}, Depth: 0})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(res.Nodes) < 2 || res.Nodes[0].ID != "mem_ord_b" || res.Nodes[1].ID != "mem_ord_a" {
		ids := []string{}
		for _, n := range res.Nodes {
			ids = append(ids, n.ID)
		}
		t.Fatalf("expected [mem_ord_b mem_ord_a] first, got %v", ids)
	}
}

func TestDecodeKeepsBodyLinksHeading(t *testing.T) {
	// A body that legitimately ends with a "## Links" section must survive
	// the round-trip when there are no frontmatter links to render.
	f := &Fragment{
		ID: "mem_bodylinks", Kind: KindNote, Title: "Body links",
		Summary: "s", Status: StatusActive, Salience: 1.0, Source: "ai",
		Body: "Notes.\n\n## Links\n- https://example.com\n",
	}
	b, err := encodeFragment(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := decodeFragment(b)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(got.Body, "## Links") {
		t.Fatalf("body lost its own ## Links section: %q", got.Body)
	}
}

func TestUpsertDedupMerge(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{{
		Op: "upsert", ID: "mem_x", Title: "Memory system v2", Summary: "Collapsed tools", Parent: "mem_inbox",
	}}})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	// near-duplicate title -> merged, not created
	res, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{{
		Op: "upsert", Title: "Memory System v2 design", Summary: "Collapsed tools into two", Parent: "mem_inbox",
	}}})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if res.Results[0].Created {
		t.Fatalf("expected merge, got created: %+v", res.Results[0])
	}
}

func TestWriteAtomicityInvalidBatch(t *testing.T) {
	s := testStore(t)
	// op 3 invalid (bad rel) -> whole batch rejected, nothing written
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{
		{Op: "upsert", ID: "mem_a", Title: "A", Parent: "mem_inbox"},
		{Op: "upsert", ID: "mem_b", Title: "B", Parent: "mem_inbox"},
		{Op: "link", From: "mem_a", To: "mem_b", Rel: "not_a_real_rel"},
	}})
	if err == nil {
		t.Fatal("expected batch error for invalid rel")
	}
	s.mu.RLock()
	_, hasA := s.idx.byID["mem_a"]
	_, hasB := s.idx.byID["mem_b"]
	s.mu.RUnlock()
	if hasA || hasB {
		t.Fatal("atomicity violated: partial writes persisted")
	}
}

func TestSecretScanRejects(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{{
		Op: "upsert", ID: "mem_secret", Title: "creds", Summary: "leak", Parent: "mem_inbox",
		Body: "aws key AKIAIOSFODNN7EXAMPLE here",
	}}})
	if err == nil {
		t.Fatal("expected secret scan rejection")
	}
}

func TestMoveRejectsCycle(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{
		{Op: "upsert", ID: "mem_parent", Title: "Parent", Parent: "mem_root"},
		{Op: "upsert", ID: "mem_child", Title: "Child", Parent: "mem_parent"},
	}})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	// moving parent under child would create a cycle
	_, err = s.Write(context.Background(), WriteArgs{Ops: []WriteOp{{
		Op: "move", ID: "mem_parent", Parent: "mem_child",
	}}})
	if err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestSymmetricEdgeMirror(t *testing.T) {
	s := testStore(t)
	_, err := s.Write(context.Background(), WriteArgs{Ops: []WriteOp{
		{Op: "upsert", ID: "mem_a", Title: "A", Parent: "mem_inbox"},
		{Op: "upsert", ID: "mem_b", Title: "B", Parent: "mem_inbox"},
		{Op: "link", From: "mem_a", To: "mem_b", Rel: RelRelates},
	}})
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	s.mu.RLock()
	a := s.idx.byID["mem_a"]
	b := s.idx.byID["mem_b"]
	s.mu.RUnlock()
	if !hasRelTo(a.Links, RelRelates, "mem_b") {
		t.Fatal("a lacks relates->b")
	}
	if !hasRelTo(b.Links, RelRelates, "mem_a") {
		t.Fatal("b lacks mirrored relates->a")
	}
}

func TestPersonaInjectionBudget(t *testing.T) {
	s := testStore(t)
	small := s.PersonaBlock(context.Background(), 200, false)
	if small == "" {
		t.Fatal("persona block empty")
	}
}

func hasRelTo(links []Link, rel Rel, to string) bool {
	for _, l := range links {
		if l.Rel == rel && l.To == to {
			return true
		}
	}
	return false
}

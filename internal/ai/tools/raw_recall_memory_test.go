package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"browser-server/internal/ai/memory"
)

func TestRawRecallMemoryNoMatches(t *testing.T) {
	raw, ok := rawRecallMemoryFormatter(memory.RecallResult{Returned: 0, TotalMatches: 0})
	if !ok {
		t.Fatal("expected formatter to accept result")
	}
	want := "QUERY | no matches\n"
	if string(raw) != want {
		t.Fatalf("got %q, want %q", raw, want)
	}
}

func TestRawRecallMemoryNodes(t *testing.T) {
	raw, ok := rawRecallMemoryFormatter(memory.RecallResult{
		Returned: 2, TotalMatches: 2,
		Nodes: []memory.RecallNode{
			{ID: "mem_a", Kind: "decision", Title: "Use raw output", Summary: "Raw output is compact", Score: 1.5, Parent: "mem_root", Tags: []string{"tools"}},
			{ID: "mem_b", Kind: "note", Title: "Second", Summary: "More text", Score: 0.5},
		},
		Edges: []memory.Edge{
			{From: "mem_a", Rel: "relates", To: "mem_b", Note: "see also"},
		},
	})
	if !ok {
		t.Fatal("expected formatter to accept result")
	}
	got := string(raw)
	if !strings.Contains(got, "QUERY | nodes=2/2") {
		t.Fatalf("missing header: %s", got)
	}
	if !strings.Contains(got, "NODE mem_a [decision] score=1.5 parent=mem_root tags=tools") {
		t.Fatalf("missing node a: %s", got)
	}
	if !strings.Contains(got, "  TITLE Use raw output") {
		t.Fatalf("missing title: %s", got)
	}
	if !strings.Contains(got, "  SUMMARY Raw output is compact") {
		t.Fatalf("missing summary: %s", got)
	}
	if !strings.Contains(got, "EDGE mem_a relates mem_b note=see also") {
		t.Fatalf("missing edge: %s", got)
	}
}

func TestRawRecallMemoryTruncatedAndHint(t *testing.T) {
	raw, ok := rawRecallMemoryFormatter(memory.RecallResult{
		Returned: 1, TotalMatches: 5, Truncated: true, Hint: "5 matches; narrow with tags/kind or raise limit",
		Nodes: []memory.RecallNode{{ID: "mem_a", Kind: "note", Title: "T", Summary: "S", Score: 1}},
	})
	if !ok {
		t.Fatal("expected formatter to accept result")
	}
	got := string(raw)
	if !strings.Contains(got, "nodes=1/5 truncated=true") {
		t.Fatalf("missing truncated flag: %s", got)
	}
	if !strings.Contains(got, "HINT 5 matches") {
		t.Fatalf("missing hint: %s", got)
	}
}

func TestRawRecallMemoryFullBody(t *testing.T) {
	raw, ok := rawRecallMemoryFormatter(memory.RecallResult{
		Returned: 1, TotalMatches: 1,
		Nodes: []memory.RecallNode{{ID: "mem_a", Kind: "note", Title: "T", Summary: "S", Body: "line1\nline2", Score: 1}},
	})
	if !ok {
		t.Fatal("expected formatter to accept result")
	}
	got := string(raw)
	if !strings.Contains(got, "  BODY line1⏎line2") {
		t.Fatalf("body newlines not sanitized: %s", got)
	}
	if strings.Contains(got, "\nline2") {
		t.Fatalf("body broke line invariant: %s", got)
	}
}

func TestRawRecallMemorySynthesized(t *testing.T) {
	raw, ok := rawRecallMemoryFormatter(memory.RecallResult{
		Synthesized: &memory.SynthesisResult{
			Synthesized: true,
			Answer:      "We decided to drop lazy loading.",
			Confidence:  0.9,
			Sources:     []string{"mem_x"},
			Gaps:        []string{"missing deploy date"},
		},
		Hint: "why did we drop lazy loading",
	})
	if !ok {
		t.Fatal("expected formatter to accept result")
	}
	got := string(raw)
	if !strings.Contains(got, "QUERY why did we drop lazy loading | SYNTH conf=0.9") {
		t.Fatalf("missing synth header: %s", got)
	}
	if !strings.Contains(got, "  ANSWER We decided to drop lazy loading.") {
		t.Fatalf("missing answer: %s", got)
	}
	if !strings.Contains(got, "  SOURCE mem_x") {
		t.Fatalf("missing source: %s", got)
	}
	if !strings.Contains(got, "  GAP missing deploy date") {
		t.Fatalf("missing gap: %s", got)
	}
}

func TestRawRecallMemoryRejectsNonResult(t *testing.T) {
	if _, ok := rawRecallMemoryFormatter(map[string]any{}); ok {
		t.Fatal("formatter should reject non-recall value")
	}
}

func TestRecallMemoryRawJSONOverride(t *testing.T) {
	// recall_memory should return JSON by default and compact text when raw is forced.
	r := New()
	args := []byte(`{"query":"test"}`)
	out, err := r.Execute(context.Background(), "recall_memory", args)
	if err != nil {
		// Memory may be disabled in default registry; skip execution assertions.
		t.Skipf("recall_memory not executable in default registry: %v", err)
	}
	if !json.Valid(out) {
		t.Fatalf("default output should be JSON: %s", out)
	}

	raw := true
	ctx := WithRawOutputOverride(context.Background(), &raw)
	out, err = r.Execute(ctx, "recall_memory", args)
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(out) {
		t.Fatalf("forced raw output should not be JSON: %s", out)
	}
}

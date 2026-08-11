package tools

import (
	"context"
	"encoding/json"
	"testing"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/memory"
)

func TestRecallMemoryToolSynthesizes(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled: true, Directory: t.TempDir(), FragmentsDir: "fragments", ArchiveDir: ".archive",
		MaxBodyKB: 64, MaxOpsPerCall: 20, MaxResultBytes: 8192, MaxDepth: 3, DefaultDepth: 1,
		RetentionDays: 365, MaintenanceInterval: "6h", SalienceDecayPerWeek: 0.985,
		Synthesizer: config.SynthesizerConfig{Enabled: true, FallbackOnError: false, MaxOutputTokens: 64, TimeoutMS: 1_000},
	}
	mem := memory.Open(cfg.Directory, cfg)
	mem.SetCompleter(memory.CompleterFunc(func(_ context.Context, _ memory.CompletionRequest) (memory.CompletionResponse, error) {
		return memory.CompletionResponse{Content: `{"answer":"D:/Codings/lang-Go/browser-server","confidence":1,"sources":["mem_working_dir"]}`}, nil
	}))
	if _, err := mem.Write(context.Background(), memory.WriteArgs{Ops: []memory.WriteOp{{
		Op: "upsert", ID: "mem_working_dir", Kind: memory.KindFact, Title: "Browser Server working directory",
		Summary: "D:/Codings/lang-Go/browser-server", Parent: "mem_projects",
	}}}); err != nil {
		t.Fatalf("setup memory: %v", err)
	}

	r := New(Options{Memory: cfg, MemoryStore: mem, Allowed: []string{"recall_memory"}})
	out, err := r.Execute(context.Background(), "recall_memory", json.RawMessage(`{
		"query":"browser-server project working directory",
		"synthesize":true
	}`))
	if err != nil {
		t.Fatalf("execute recall_memory: %v", err)
	}
	var result memory.RecallResult
	if err := json.Unmarshal(out, &result); err != nil || result.Synthesized == nil {
		t.Fatalf("expected synthesized RecallResult, got %s (error: %v)", out, err)
	}
	if result.Synthesized.Answer != "D:/Codings/lang-Go/browser-server" {
		t.Fatalf("unexpected answer: %q", result.Synthesized.Answer)
	}
}

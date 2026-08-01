package modelrefresh

import (
	"reflect"
	"testing"

	"browser-server/internal/ai/config"
)

func TestMergeAddsAllFetchedWhenEmpty(t *testing.T) {
	fetched := []ModelInfo{
		{ID: "a", Label: "Model A", SupportsTools: true, MaxOutputTokens: 4096},
		{ID: "b", Label: "Model B", SupportsTools: true, MaxOutputTokens: 8192},
	}
	got := Merge(nil, fetched)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("unexpected order: %+v", got)
	}
	if !got[0].Default {
		t.Fatalf("expected first entry to become default, got %+v", got[0])
	}
	if got[1].Default {
		t.Fatalf("expected only one default, got %+v", got[1])
	}
}

func TestMergePreservesExistingUnchanged(t *testing.T) {
	existing := []config.ModelConfig{
		{ID: "a", Label: "Custom A", SupportsTools: false, Default: true, MaxOutputTokens: 2048},
		{ID: "b", Label: "Custom B", SupportsTools: true, Default: false, MaxOutputTokens: 4096},
	}
	fetched := []ModelInfo{
		{ID: "a", Label: "Fetched A", SupportsTools: true, MaxOutputTokens: 99999},
		{ID: "c", Label: "Model C", SupportsTools: true, MaxOutputTokens: 4096},
	}
	got := Merge(existing, fetched)
	want := []config.ModelConfig{
		{ID: "a", Label: "Custom A", SupportsTools: false, Default: true, MaxOutputTokens: 2048},
		{ID: "b", Label: "Custom B", SupportsTools: true, Default: false, MaxOutputTokens: 4096},
		{ID: "c", Label: "Model C", SupportsTools: true, Default: false, MaxOutputTokens: 4096},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Merge = %+v, want %+v", got, want)
	}
}

func TestMergeSkipsDuplicateFetched(t *testing.T) {
	existing := []config.ModelConfig{
		{ID: "a", Label: "A", SupportsTools: true, Default: true, MaxOutputTokens: 4096},
	}
	fetched := []ModelInfo{
		{ID: "a", Label: "A again", SupportsTools: true, MaxOutputTokens: 4096}, // already exists
		{ID: "b", Label: "B", SupportsTools: true, MaxOutputTokens: 4096},
		{ID: "b", Label: "B again", SupportsTools: true, MaxOutputTokens: 4096}, // duplicate within fetch
		{ID: "", Label: "blank id", SupportsTools: true, MaxOutputTokens: 4096}, // blank id skipped
	}
	got := Merge(existing, fetched)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("unexpected entries: %+v", got)
	}
	if !got[0].Default {
		t.Fatalf("existing default must be preserved")
	}
	if got[1].Default {
		t.Fatalf("fetched entry must not be default")
	}
}

func TestMergeEmptyFetchedLeavesExistingUnchanged(t *testing.T) {
	existing := []config.ModelConfig{
		{ID: "a", Label: "A", SupportsTools: true, Default: true, MaxOutputTokens: 4096},
		{ID: "b", Label: "B", SupportsTools: true, Default: false, MaxOutputTokens: 8192},
	}
	got := Merge(existing, nil)
	if !reflect.DeepEqual(got, existing) {
		t.Fatalf("Merge = %+v, want unchanged %+v", got, existing)
	}
}

func TestMergeAddsDefaultOnlyWhenNoneExists(t *testing.T) {
	existing := []config.ModelConfig{
		{ID: "a", Label: "A", SupportsTools: true, Default: false, MaxOutputTokens: 4096},
	}
	fetched := []ModelInfo{
		{ID: "b", Label: "B", SupportsTools: true, MaxOutputTokens: 4096},
	}
	got := Merge(existing, fetched)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !got[0].Default {
		t.Fatalf("expected first entry to become default when none existed, got %+v", got[0])
	}
	if got[1].Default {
		t.Fatalf("expected only one default, got %+v", got[1])
	}

	// With an existing default, nothing is flipped.
	existing[0].Default = true
	got = Merge(existing, fetched)
	if !got[0].Default || got[1].Default {
		t.Fatalf("existing default must be preserved, got %+v", got)
	}
}

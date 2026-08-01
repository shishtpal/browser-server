package tools

import (
	"context"
	"testing"
	"time"
)

func repeat(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[0]
	}
	return string(out)
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestOutputBudget(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"default limit", 0, defaultMaxOutput - resultHeadroom},
		{"full headroom", 2*resultHeadroom + 1, resultHeadroom + 1},
		{"headroom boundary clamps to half", 2 * resultHeadroom, resultHeadroom},
		{"below headroom clamps to half", 1024, 512},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.limit == 0 {
				ctx = context.Background()
			} else {
				ctx = withToolLimits(context.Background(), toolLimits{maxOutput: tt.limit})
			}
			if got := outputBudget(ctx); got != tt.want {
				t.Errorf("outputBudget(limit=%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

func TestLimitsFromDefaults(t *testing.T) {
	l := limitsFrom(context.Background())
	if l.maxOutput != defaultMaxOutput {
		t.Errorf("maxOutput = %d, want %d", l.maxOutput, defaultMaxOutput)
	}
	if l.gitTimeout != defaultGitTimeout {
		t.Errorf("gitTimeout = %v, want %v", l.gitTimeout, defaultGitTimeout)
	}
	if l.gitMaxOutput != defaultMaxOutput || l.gitMaxDiffOutput != defaultMaxOutput {
		t.Errorf("git limits = %d/%d, want %d/%d", l.gitMaxOutput, l.gitMaxDiffOutput, defaultMaxOutput, defaultMaxOutput)
	}
	if l.rawOutput != nil {
		t.Errorf("rawOutput = %v, want nil default", l.rawOutput)
	}
}

func TestWithToolLimitsRoundtrip(t *testing.T) {
	// Compare field-by-field: toolLimits contains a map (rawOutput), so the
	// struct is not directly comparable with !=.
	want := toolLimits{
		maxOutput:        12345,
		gitTimeout:       7 * time.Second,
		gitMaxOutput:     23456,
		gitMaxDiffOutput: 34567,
		rawOutput:        map[string]bool{"read_file": true},
	}
	ctx := withToolLimits(context.Background(), want)
	got := limitsFrom(ctx)
	if got.maxOutput != want.maxOutput ||
		got.gitTimeout != want.gitTimeout ||
		got.gitMaxOutput != want.gitMaxOutput ||
		got.gitMaxDiffOutput != want.gitMaxDiffOutput {
		t.Fatalf("limitsFrom = %+v, want %+v", got, want)
	}
	if len(got.rawOutput) != len(want.rawOutput) || !got.rawOutput["read_file"] {
		t.Fatalf("rawOutput = %v, want %v", got.rawOutput, want.rawOutput)
	}
}

func TestRawOutputOverride(t *testing.T) {
	if got := rawOverrideFrom(context.Background()); got != nil {
		t.Fatalf("rawOverrideFrom(background) = %v, want nil", got)
	}
	raw := true
	ctx := WithRawOutputOverride(context.Background(), &raw)
	if got := rawOverrideFrom(ctx); got == nil || *got != true {
		t.Fatalf("rawOverrideFrom(override true) = %v, want true", got)
	}
	jsonMode := false
	ctx = WithRawOutputOverride(context.Background(), &jsonMode)
	if got := rawOverrideFrom(ctx); got == nil || *got != false {
		t.Fatalf("rawOverrideFrom(override false) = %v, want false", got)
	}
}

func TestTruncateUTF8(t *testing.T) {
	// The emoji is a 4-byte rune; cutting inside it must drop the partial
	// trailing bytes instead of emitting invalid UTF-8.
	s := "a\U0001F600b"
	if got := truncateUTF8(s, 3); got != "a" {
		t.Errorf("truncateUTF8(%q, 3) = %q, want %q", s, got, "a")
	}
	if got := truncateUTF8(s, 5); got != "a\U0001F600" {
		t.Errorf("truncateUTF8(%q, 5) = %q, want %q", s, got, "a\U0001F600")
	}
	if got := truncateUTF8(s, 100); got != s {
		t.Errorf("truncateUTF8(%q, 100) = %q, want unchanged", s, got)
	}
	if got := truncateUTF8("hello", 0); got != "" {
		t.Errorf("truncateUTF8 with limit 0 = %q, want empty", got)
	}
}

func TestTruncateBytesUTF8(t *testing.T) {
	b := []byte("a\U0001F600b")
	if got := truncateBytesUTF8(b, 3); string(got) != "a" {
		t.Errorf("truncateBytesUTF8(%q, 3) = %q, want %q", b, got, "a")
	}
	if got := truncateBytesUTF8(b, 100); string(got) != string(b) {
		t.Errorf("truncateBytesUTF8(%q, 100) = %q, want unchanged", b, got)
	}
}

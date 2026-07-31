package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// Output and size limits used across tools. These are package-level defaults;
// individual tools are wired with configurable values from bs-ai-config.json.
const (
	defaultMaxOutput      = 32 * 1024
	resultHeadroom        = 2048
	maxSourceSize         = 8 << 20
	codeToolTimeout       = 30 * time.Second
	maxCommandOutputBytes = 64 * 1024
	defaultGitTimeout     = 30 * time.Second
)

// toolLimits carries the per-invocation output limits and git settings. It is
// attached to the execution context by Registry.Execute so tool functions
// never read shared mutable package state, which would race (and clobber each
// other) when more than one Registry is constructed, e.g. under hot reload or
// in parallel tests. Calls made without a registry (direct test calls) fall
// back to defaultToolLimits.
type toolLimits struct {
	maxOutput        int
	gitTimeout       time.Duration
	gitMaxOutput     int
	gitMaxDiffOutput int
}

func defaultToolLimits() toolLimits {
	return toolLimits{
		maxOutput:        defaultMaxOutput,
		gitTimeout:       defaultGitTimeout,
		gitMaxOutput:     defaultMaxOutput,
		gitMaxDiffOutput: defaultMaxOutput,
	}
}

type toolLimitsKey struct{}

// withToolLimits returns a context carrying the limits for one invocation.
func withToolLimits(ctx context.Context, l toolLimits) context.Context {
	return context.WithValue(ctx, toolLimitsKey{}, l)
}

// limitsFrom returns the limits attached to ctx, or the package defaults when
// none were attached.
func limitsFrom(ctx context.Context) toolLimits {
	if l, ok := ctx.Value(toolLimitsKey{}).(toolLimits); ok {
		return l
	}
	return defaultToolLimits()
}

// maxOutputFrom returns the configured output limit for this invocation.
func maxOutputFrom(ctx context.Context) int {
	return limitsFrom(ctx).maxOutput
}

// outputBudget returns the payload budget a tool may fill before the JSON
// response envelope (keys, braces, summary fields) pushes the marshaled result
// over the configured limit. resultHeadroom bytes are reserved for that
// envelope; when the configured limit cannot fit the full headroom, the budget
// is clamped to half the limit so the threshold always stays positive. A zero
// or negative budget would silently truncate every result.
func outputBudget(ctx context.Context) int {
	limit := maxOutputFrom(ctx)
	if limit > 2*resultHeadroom {
		return limit - resultHeadroom
	}
	return limit / 2
}

// truncateUTF8 cuts s to at most limit bytes without splitting a UTF-8 rune.
func truncateUTF8(s string, limit int) string {
	return string(truncateBytesUTF8([]byte(s), limit))
}

// truncateBytesUTF8 cuts b to at most limit bytes without splitting a UTF-8
// rune. If the limit lands in the middle of a multi-byte rune, the partial
// trailing bytes are dropped.
func truncateBytesUTF8(b []byte, limit int) []byte {
	if len(b) <= limit {
		return b
	}
	b = b[:limit]
	for len(b) > 0 {
		r, size := utf8.DecodeLastRune(b)
		if r != utf8.RuneError || size > 1 {
			return b
		}
		b = b[:len(b)-size]
	}
	return b
}

// strict validates that a JSON raw message only contains allowed keys, then
// unmarshals it into dst.
func strict(raw json.RawMessage, dst any, allowed map[string]bool) error {
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	if fields == nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("unknown argument %q", k)
		}
	}
	return json.Unmarshal(raw, dst)
}

// validateGlobs validates that all provided glob patterns are syntactically correct.
func validateGlobs(patterns ...[]string) error {
	for _, set := range patterns {
		for _, pattern := range set {
			if _, err := path.Match(filepath.ToSlash(pattern), ""); err != nil {
				return fmt.Errorf("invalid glob %q: %w", pattern, err)
			}
		}
	}
	return nil
}

// globMatch checks if a relative path matches any of the given glob patterns.
func globMatch(patterns []string, rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		p = filepath.ToSlash(p)
		target := path.Base(rel)
		if strings.Contains(p, "/") {
			target = rel
		}
		if ok, _ := path.Match(p, target); ok {
			return true
		}
	}
	return false
}

// contextReader wraps an io.Reader with context cancellation support.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}

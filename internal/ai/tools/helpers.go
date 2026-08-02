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
	rawOutput        map[string]bool
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

type rawOverrideKey struct{}

// WithRawOutputOverride attaches a per-request raw-output override to ctx.
// nil means "use the config tools.raw_output allowlist"; true forces raw
// output for every tool that has a RawContentFunc; false forces JSON for every
// tool. This is how the chat UI toggles raw vs JSON per message without
// editing bs-ai-config.json.
func WithRawOutputOverride(ctx context.Context, override *bool) context.Context {
	return context.WithValue(ctx, rawOverrideKey{}, override)
}

// rawOverrideFrom returns the per-request raw-output override attached to ctx,
// or nil when the config allowlist should be used.
func rawOverrideFrom(ctx context.Context) *bool {
	if v, ok := ctx.Value(rawOverrideKey{}).(*bool); ok {
		return v
	}
	return nil
}

// maxOutputFrom returns the configured output limit for this invocation.
func maxOutputFrom(ctx context.Context) int {
	return limitsFrom(ctx).maxOutput
}

// outputLimitFor returns the maximum result size allowed for a tool's final
// output. Tools with a dedicated output cap (git_diff uses max_diff_output)
// are honored; every other tool is bounded by the general max_output. The
// same cap applies to raw and JSON-encoded output so the diff-specific limit
// is not silently overridden by Registry.Execute's generic size check.
func outputLimitFor(name string, lim toolLimits) int {
	if name == "git_diff" {
		if lim.gitMaxDiffOutput > 0 {
			return lim.gitMaxDiffOutput
		}
		return lim.maxOutput
	}
	return lim.maxOutput
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

func validateSearchPagination(raw json.RawMessage, legacy string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, name := range []string{"page", "page_size", legacy} {
		value, ok := fields[name]
		if !ok {
			continue
		}
		var n int
		if err := json.Unmarshal(value, &n); err != nil || n < 1 {
			return fmt.Errorf("%s must be at least 1", name)
		}
	}
	return nil
}

// resolvePageSize resolves the page size for a search request, supporting the
// legacy limit alias. It rejects requests that set both page_size and limit,
// clamps page_size to the 1–100 range, validates limit against oldMax (the
// tool-specific legacy maximum), and falls back to defaultSize when neither is
// provided.
func resolvePageSize(pageSize, limit, oldMax, defaultSize int) (int, error) {
	if pageSize != 0 && limit != 0 {
		return 0, fmt.Errorf("cannot specify both page_size and limit")
	}
	if pageSize != 0 {
		if pageSize < 1 || pageSize > 100 {
			return 0, fmt.Errorf("page_size must be between 1 and 100")
		}
		return pageSize, nil
	}
	if limit != 0 {
		if limit < 1 || limit > oldMax {
			return 0, fmt.Errorf("limit must be 1 to %d", oldMax)
		}
		return limit, nil
	}
	return defaultSize, nil
}

// fitSearchEnvelope keeps complete result objects within the configured tool
// output budget. Search totals continue to describe the evaluated candidate
// set, while truncated makes the partial response explicit.
//
// Results are ordered by relevance, so when the budget is exceeded the least
// relevant (trailing) results are dropped first. Instead of re-marshaling the
// whole envelope once per dropped result (O(n^2) when the budget is small), the
// largest fitting prefix is found by binary search: the marshaled size grows
// monotonically with the number of results, so only O(log n) marshals are
// needed.
func fitSearchEnvelope(ctx context.Context, envelope map[string]any) map[string]any {
	results, ok := envelope["results"].([]map[string]any)
	if !ok {
		return envelope
	}
	budget := outputBudget(ctx)

	// fits reports whether the envelope marshals within budget with the first
	// k results. Marshaled size is monotonic in k, which is what makes the
	// binary search below sound.
	fits := func(k int) bool {
		orig := envelope["results"]
		envelope["results"] = results[:k]
		encoded, err := json.Marshal(envelope)
		envelope["results"] = orig
		return err == nil && len(encoded) <= budget
	}

	// Fast path: the full result set already fits.
	if fits(len(results)) {
		return envelope
	}

	// The largest fitting prefix is in [0, len(results)). The fast path above
	// guarantees hi = len(results) does not fit; lo tracks the best prefix found
	// so far and ends as the largest k that fits (0 when even an empty result set
	// overflows the budget). Narrow the range until lo and hi are adjacent.
	lo, hi := 0, len(results)
	for lo+1 < hi {
		mid := lo + (hi-lo)/2
		if fits(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}

	envelope["results"] = results[:lo]
	envelope["truncated"] = true
	envelope["has_more"] = true
	return envelope
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

// rawMapField returns a RawContentFunc that extracts a named string field from
// a map result.
func rawMapField(field string) func(any) ([]byte, bool) {
	return func(v any) ([]byte, bool) {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		s, ok := m[field].(string)
		if !ok {
			return nil, false
		}
		return []byte(s), true
	}
}

// rawTrue returns "true" as raw output.
func rawTrue(v any) ([]byte, bool) {
	return []byte("true"), true
}

// rawString returns the string value of v as raw output, falling back to false
// if v is not a string.
func rawString(v any) ([]byte, bool) {
	s, ok := v.(string)
	if !ok {
		return nil, false
	}
	return []byte(s), true
}

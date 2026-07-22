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
)

// Output and size limits used across tools.
const (
	maxOutput       = 32 * 1024
	maxOutputBytes  = 64 * 1024
	resultHeadroom  = 2048
	maxSourceSize   = 8 << 20
	codeToolTimeout = 30 * time.Second
)

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

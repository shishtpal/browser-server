package modelrefresh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultFetchTimeout bounds each catalog request. The plan's convention is a
// 30-second default; Refresh callers may set their own deadline on ctx.
const defaultFetchTimeout = 30 * time.Second

// maxResponseBytes caps how much of a catalog response we will read.
const maxResponseBytes = 8 * 1024 * 1024

// fetchJSON GETs url with the provider API key and decodes the JSON response
// into dest. It matches the header conventions used by internal/ai/provider.
func fetchJSON(ctx context.Context, url, apiKey string, dest any) error {
	ctx, cancel := context.WithTimeout(ctx, defaultFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "browser-server")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("read response from %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d: %s", url, resp.StatusCode, truncate(string(body), 256))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse JSON from %s: %w", url, err)
	}
	return nil
}

// truncate returns s trimmed and capped at max runes for diagnostics.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

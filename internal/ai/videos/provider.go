package videos

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// sharedTransport lets every provider HTTP client reuse keep-alive connections
// instead of creating a fresh transport per request/poll.
var sharedTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          32,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// pollResult is the normalized outcome of a single provider poll.
type pollResult struct {
	Status   VideoStatus
	Progress int
	VideoURL string
	Size     string
	Seconds  float64
}

// providerImpl abstracts a video-generation vendor's async workflow: submit a
// task and later poll its result. Adding a new vendor means implementing this
// interface and registering its type in newProviderImpl. Clamp each
// implementation's numeric bounds against the model spec where possible (see
// validateAgnesConstraintsWithSpecs) instead of duplicating them in config.
type providerImpl interface {
	// Create submits a generation task and returns the provider's video ID used
	// to retrieve the result.
	Create(ctx context.Context, p Provider, m Model, r GenerateRequest) (videoID string, err error)
	// Poll retrieves the current task status, progress, and (when completed) the
	// result URL plus output metadata. model is the provider model ID that
	// created the task; some vendors require it to resolve the result.
	Poll(ctx context.Context, p Provider, videoID, model string) (pollResult, error)
}

func newProviderImpl(typ string) (providerImpl, error) {
	switch typ {
	case "agnes_video":
		return agnesProvider{}, nil
	case "openrouter_video":
		return openrouterVideoProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported video provider type %q", typ)
	}
}

// contentFetcher is implemented by providers whose finished video must be
// fetched through the provider API (e.g. with an Authorization header) rather
// than a plain GET of pollResult.VideoURL. When a provider does not implement
// it, the service downloads res.VideoURL directly.
type contentFetcher interface {
	// Fetch retrieves the completed video bytes and their content type.
	Fetch(ctx context.Context, p Provider, res pollResult) (data []byte, contentType string, err error)
}

func stringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"browser-server/internal/auth"
)

// Client is the interface the AI tool layer uses to drive the bus. Two
// implementations exist:
//
//   - LocalClient: in-process (server); commands flow through the bus directly.
//   - HTTPClient: REST (bs-ai-chat); commands round-trip through the running
//     server on localhost.
type Client interface {
	ListInstances(ctx context.Context) ([]Instance, error)
	ListTabs(ctx context.Context, instanceID string) ([]Tab, error)
	CreateCommand(ctx context.Context, req CreateCommandRequest) (Command, error)
	WaitCommand(ctx context.Context, commandID string, timeout time.Duration) (Command, error)
}

// LocalClient adapts a Bus for in-process tool calls.
type LocalClient struct {
	Bus *Bus
}

func (c *LocalClient) ListInstances(ctx context.Context) ([]Instance, error) {
	if c == nil || c.Bus == nil {
		return nil, errors.New("browser automation is not enabled")
	}
	return c.Bus.ListInstances(), nil
}

func (c *LocalClient) ListTabs(ctx context.Context, instanceID string) ([]Tab, error) {
	if c == nil || c.Bus == nil {
		return nil, errors.New("browser automation is not enabled")
	}
	return c.Bus.ListTabs(instanceID)
}

func (c *LocalClient) CreateCommand(ctx context.Context, req CreateCommandRequest) (Command, error) {
	if c == nil || c.Bus == nil {
		return Command{}, errors.New("browser automation is not enabled")
	}
	return c.Bus.CreateCommand(ctx, req)
}

func (c *LocalClient) WaitCommand(ctx context.Context, commandID string, timeout time.Duration) (Command, error) {
	if c == nil || c.Bus == nil {
		return Command{}, errors.New("browser automation is not enabled")
	}
	return c.Bus.WaitForResult(ctx, commandID, timeout)
}

// TokenProvider supplies the operator API token for HTTP calls.
type TokenProvider func() string

// HTTPClient drives a remote Bus over the server's REST API. It is used by
// the bs-ai-chat CLI: the CLI registers the same tools, but browser state and
// the extension connection live in the long-running server.
type HTTPClient struct {
	BaseURL string
	Token   TokenProvider
	HTTP    *http.Client
}

// ServerURL returns the browser-server base URL used by out-of-process
// clients (bs-ai-chat). It honors BS_BROWSER_SERVER_URL, falling back to the
// default localhost:9191.
func ServerURL() string {
	if v := os.Getenv("BS_BROWSER_SERVER_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:9191"
}

// OperatorTokenProvider returns a TokenProvider that lazily loads the server's
// operator token from disk (auth.Load) and returns it. Used by the CLI so the
// browser tools can authenticate to the running server.
func OperatorTokenProvider() TokenProvider {
	return func() string {
		if !auth.Configured() {
			if err := auth.Load(); err != nil {
				return ""
			}
		}
		return auth.Token()
	}
}

// NewHTTPClient creates an HTTP client with a sane default timeout.
func NewHTTPClient(baseURL string, token TokenProvider) *HTTPClient {
	if baseURL == "" {
		baseURL = "http://localhost:9191"
	}
	return &HTTPClient{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 0}, // per-request deadlines via ctx
	}
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != nil {
		if tok := c.Token(); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("browser automation requires the browser-server to be running at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &envelope)
		msg := envelope.Error
		if msg == "" {
			msg = strings.TrimSpace(string(data))
		}
		return fmt.Errorf("browser server: %s", msg)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *HTTPClient) ListInstances(ctx context.Context) ([]Instance, error) {
	var out []Instance
	err := c.do(ctx, http.MethodGet, "/api/browser/instances", nil, &out)
	return out, err
}

func (c *HTTPClient) ListTabs(ctx context.Context, instanceID string) ([]Tab, error) {
	var out []Tab
	err := c.do(ctx, http.MethodGet, "/api/browser/instances/"+instanceID+"/tabs", nil, &out)
	return out, err
}

func (c *HTTPClient) CreateCommand(ctx context.Context, req CreateCommandRequest) (Command, error) {
	var out Command
	err := c.do(ctx, http.MethodPost, "/api/browser/cmd", req, &out)
	return out, err
}

// WaitCommand polls GET /api/browser/commands/{id} until the command reaches a
// terminal state or the deadline passes.
func (c *HTTPClient) WaitCommand(ctx context.Context, commandID string, timeout time.Duration) (Command, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		var out Command
		err := c.do(ctx, http.MethodGet, "/api/browser/commands/"+commandID, nil, &out)
		if err != nil {
			return Command{}, err
		}
		switch out.Status {
		case StatusSucceeded, StatusFailed, StatusTimedOut:
			return out, nil
		}
		if time.Now().After(deadline) {
			return out, fmt.Errorf("command timed out after %s (status %s)", timeout, out.Status)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return out, ctx.Err()
		}
	}
}

package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxTools = 200

// DiscoveredTool is the transport-neutral tool definition consumed by the AI layer.
type DiscoveredTool struct {
	Name         string          `json:"name"`
	OriginalName string          `json:"original_name"`
	Description  string          `json:"description,omitempty"`
	Category     string          `json:"category"`
	Schema       json.RawMessage `json:"schema"`
}

type route struct {
	server, original string
	session          *sdk.ClientSession
	timeout          time.Duration
}
type managedServer struct {
	name    string
	session *sdk.ClientSession
	status  ServerStatus
}

// Manager owns MCP sessions, discovered tool routes, and sanitized status.
type Manager struct {
	configured bool
	mu         sync.RWMutex
	servers    []*managedServer
	tools      []DiscoveredTool
	routes     map[string]route
	closeOnce  sync.Once
	closeErr   error
}

// NewManager connects enabled servers concurrently. A server connection failure
// is recorded as unavailable and does not prevent other servers from working.
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("nil MCP configuration")
	}
	m := &Manager{configured: cfg.Configured, routes: make(map[string]route)}
	names := make([]string, 0, len(cfg.MCPServers))
	for name := range cfg.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)
	results := make(chan *managedServer, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		sc := cfg.MCPServers[name]
		if !sc.enabled() {
			results <- &managedServer{name: name, status: ServerStatus{Name: name, Status: "disabled"}}
			continue
		}
		wg.Add(1)
		go func() { defer wg.Done(); results <- connectServer(ctx, name, sc) }()
	}
	wg.Wait()
	close(results)
	for s := range results {
		m.servers = append(m.servers, s)
	}
	sort.Slice(m.servers, func(i, j int) bool { return m.servers[i].name < m.servers[j].name })
	for _, s := range m.servers {
		if s.session == nil {
			continue
		}
		for _, t := range s.status.discovered {
			if len(m.tools) >= maxTools {
				s.status.Warnings = append(s.status.Warnings, "tool limit reached")
				break
			}
			public := publicName(s.name, t.original)
			if _, exists := m.routes[public]; exists {
				public = publicNameHashed(s.name, t.original)
				if _, hashedExists := m.routes[public]; hashedExists {
					s.status.Warnings = append(s.status.Warnings, fmt.Sprintf("tool %q has a duplicate public name", safeLabel(t.original)))
					continue
				}
			}
			m.routes[public] = route{server: s.name, original: t.original, session: s.session, timeout: t.timeout}
			m.tools = append(m.tools, DiscoveredTool{Name: public, OriginalName: t.original, Description: t.description, Category: "MCP: " + s.name, Schema: t.schema})
			s.status.Tools = append(s.status.Tools, public)
		}
		s.status.discovered = nil
	}
	sort.Slice(m.tools, func(i, j int) bool { return m.tools[i].Name < m.tools[j].Name })
	return m, nil
}

type pendingTool struct {
	original, description string
	schema                json.RawMessage
	timeout               time.Duration
}

func connectServer(parent context.Context, name string, sc ServerConfig) *managedServer {
	ms := &managedServer{name: name, status: ServerStatus{Name: name, Status: "unavailable"}}
	transport, err := makeTransport(name, sc)
	if err != nil {
		ms.status.Error = "transport setup failed"
		return ms
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(sc.ConnectTimeoutSeconds)*time.Second)
	defer cancel()
	client := sdk.NewClient(
		&sdk.Implementation{Name: "browser-server", Version: "1"},
		&sdk.ClientOptions{Capabilities: &sdk.ClientCapabilities{}},
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		ms.status.Error = sanitizedConnectionError(err)
		return ms
	}
	ms.session = session
	allowed := make(map[string]bool, len(sc.AllowedTools))
	for _, n := range sc.AllowedTools {
		allowed[n] = true
	}
	seen := make(map[string]bool)
	for tool, listErr := range session.Tools(ctx, nil) {
		if listErr != nil {
			ms.status.Status = "unavailable"
			ms.status.Error = "tool discovery failed"
			_ = session.Close()
			ms.session = nil
			return ms
		}
		if tool == nil || tool.Name == "" || len(allowed) != 0 && !allowed[tool.Name] {
			continue
		}
		if seen[tool.Name] {
			ms.status.Warnings = append(ms.status.Warnings, fmt.Sprintf("tool %q was discovered more than once", safeLabel(tool.Name)))
			continue
		}
		seen[tool.Name] = true
		schema, schemaErr := validSchema(tool.InputSchema)
		if schemaErr != nil {
			ms.status.Warnings = append(ms.status.Warnings, fmt.Sprintf("tool %q has an invalid schema", safeLabel(tool.Name)))
			continue
		}
		desc := tool.Description
		if len(desc) > 4096 {
			desc = desc[:4096]
		}
		ms.status.discovered = append(ms.status.discovered, pendingTool{original: tool.Name, description: desc, schema: schema, timeout: time.Duration(sc.CallTimeoutSeconds) * time.Second})
	}
	for n := range allowed {
		if !seen[n] {
			ms.status.Warnings = append(ms.status.Warnings, fmt.Sprintf("allowed tool %q was not discovered", safeLabel(n)))
		}
	}
	sort.Slice(ms.status.discovered, func(i, j int) bool { return ms.status.discovered[i].original < ms.status.discovered[j].original })
	sort.Strings(ms.status.Warnings)
	ms.status.Status = "connected"
	return ms
}

func makeTransport(serverName string, sc ServerConfig) (sdk.Transport, error) {
	if sc.Command != "" {
		cmd := exec.Command(sc.Command, sc.Args...)
		cmd.Dir = sc.Cwd
		cmd.Env = mergedEnvironment(sc.Env)
		cmd.Stderr = &boundedStderrWriter{server: serverName, secrets: mapValues(sc.Env), remaining: 16 * 1024}
		return &sdk.CommandTransport{Command: cmd}, nil
	}
	base := http.DefaultTransport.(*http.Transport).Clone()
	rt := http.RoundTripper(base)
	if len(sc.Headers) != 0 {
		rt = headerTransport{base: rt, headers: sc.Headers}
	}
	origin, _ := url.Parse(sc.URL)
	client := &http.Client{
		Transport: rt,
		Timeout:   time.Duration(sc.CallTimeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != origin.Scheme || !strings.EqualFold(req.URL.Host, origin.Host) {
				return errors.New("cross-origin redirect refused")
			}
			return nil
		},
	}
	return &sdk.StreamableClientTransport{Endpoint: sc.URL, HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true}, nil
}

type boundedStderrWriter struct {
	server    string
	secrets   []string
	remaining int
	mu        sync.Mutex
}

func (w *boundedStderrWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.remaining <= 0 {
		return len(p), nil
	}
	message := string(p)
	for _, secret := range w.secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[REDACTED]")
		}
	}
	message = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r >= 32 {
			return r
		}
		return -1
	}, message)
	if len(message) > w.remaining {
		message = message[:w.remaining]
	}
	w.remaining -= len(message)
	message = strings.TrimSpace(message)
	if message != "" {
		log.Printf("MCP stdio (%s): %s", safeLabel(w.server), message)
	}
	return len(p), nil
}

func mapValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func mergedEnvironment(overrides map[string]string) []string {
	values := make(map[string]string)
	keys := make(map[string]string)
	for _, entry := range os.Environ() {
		separator := strings.IndexByte(entry, '=')
		// Windows drive-current-directory variables have names such as =C:.
		if separator == 0 {
			if next := strings.IndexByte(entry[1:], '='); next >= 0 {
				separator = next + 1
			}
		}
		if separator < 1 {
			continue
		}
		key, value := entry[:separator], entry[separator+1:]
		canonical := environmentKey(key)
		keys[canonical], values[canonical] = key, value
	}
	for key, value := range overrides {
		canonical := environmentKey(key)
		keys[canonical], values[canonical] = key, value
	}
	ordered := make([]string, 0, len(keys))
	for canonical := range keys {
		ordered = append(ordered, canonical)
	}
	sort.Strings(ordered)
	out := make([]string, 0, len(ordered))
	for _, canonical := range ordered {
		out = append(out, keys[canonical]+"="+values[canonical])
	}
	return out
}

func environmentKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header = r.Header.Clone()
	for k, v := range h.headers {
		clone.Header.Set(k, v)
	}
	return h.base.RoundTrip(clone)
}

func validSchema(value any) (json.RawMessage, error) {
	b, err := json.Marshal(value)
	if err != nil || len(b) > 64*1024 {
		return nil, errors.New("invalid schema")
	}
	var object map[string]any
	if json.Unmarshal(b, &object) != nil || object == nil {
		return nil, errors.New("schema must be an object")
	}
	if typ, ok := object["type"]; ok && typ != "object" {
		return nil, errors.New("schema type must be object")
	}
	return b, nil
}

func safeLabel(s string) string {
	if len(s) > 100 {
		return s[:100]
	}
	return s
}
func sanitizedConnectionError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "connection timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "connection canceled"
	}
	return "connection failed"
}

func publicName(server, tool string) string {
	raw := "mcp_" + server + "_" + tool
	n := normalizeName(raw)
	if n != raw || len(n) > 64 {
		return hashedName(n, raw)
	}
	return n
}
func publicNameHashed(server, tool string) string {
	raw := "mcp_" + server + "_" + tool
	return hashedName(normalizeName(raw), raw)
}
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	n := strings.Trim(b.String(), "_-")
	if n == "" {
		return "mcp_tool"
	}
	return n
}
func hashedName(base, raw string) string {
	sum := sha256.Sum256([]byte(raw))
	suffix := "_" + hex.EncodeToString(sum[:5])
	if len(base) > 64-len(suffix) {
		base = base[:64-len(suffix)]
	}
	return strings.TrimRight(base, "_-") + suffix
}

// Configured reports whether an MCP configuration file was loaded.
func (m *Manager) Configured() bool { return m != nil && m.configured }

// Tools returns a deterministic defensive copy of usable discovered tools.
func (m *Manager) Tools() []DiscoveredTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]DiscoveredTool(nil), m.tools...)
}

// Execute invokes a discovered tool by public name.
func (m *Manager) Execute(ctx context.Context, publicName string, raw json.RawMessage) (any, error) {
	m.mu.RLock()
	r, ok := m.routes[publicName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown MCP tool %q", publicName)
	}
	var args map[string]any
	if len(raw) == 0 {
		args = map[string]any{}
	} else if err := json.Unmarshal(raw, &args); err != nil || args == nil {
		return nil, errors.New("MCP tool arguments must be a JSON object")
	}
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	result, err := r.session.CallTool(callCtx, &sdk.CallToolParams{Name: r.original, Arguments: args})
	if err != nil {
		return nil, fmt.Errorf("MCP server %q tool %q call failed: %w", r.server, r.original, sanitizeCallError(err))
	}
	return normalizeResult(result)
}

func sanitizeCallError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("timed out")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("canceled")
	}
	return errors.New("protocol or transport error")
}

// Close closes every session at most once. It is safe to call repeatedly.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.closeOnce.Do(func() {
		for _, s := range m.servers {
			if s.session != nil && s.session.Close() != nil {
				m.closeErr = errors.New("one or more MCP sessions failed to close")
			}
		}
	})
	return m.closeErr
}

package search

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCacheControlsAndCopies(t *testing.T) {
	c := NewSearchCache(time.Minute, 2)
	q := SearchQuery{Query: "q", IncludeDomains: []string{"a"}, ExcludeDomains: []string{"b"}}
	original := &SearchResponse{Results: []SearchResult{{Title: "safe"}}}
	c.Set(q, original)
	original.Results[0].Title = "mutated"
	got, ok := c.Get(q)
	if !ok || got.Results[0].Title != "safe" {
		t.Fatal("cache did not copy stored response")
	}
	got.Results[0].Title = "caller"
	again, _ := c.Get(q)
	if again.Results[0].Title != "safe" {
		t.Fatal("cache result is mutable")
	}
	q.IncludeDomains = []string{"other"}
	if _, ok = c.Get(q); ok {
		t.Fatal("domain controls must be part of key")
	}
}

type mockProvider struct {
	name  string
	mu    sync.Mutex
	calls int
	errs  []error
}

func (m *mockProvider) Name() string                     { return m.name }
func (m *mockProvider) RequiresAPIKey() bool             { return false }
func (m *mockProvider) IsAvailable(context.Context) bool { return true }
func (m *mockProvider) Search(_ context.Context, q SearchQuery) (*SearchResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if len(m.errs) > 0 {
		e := m.errs[0]
		m.errs = m.errs[1:]
		return nil, e
	}
	return &SearchResponse{Results: []SearchResult{{Title: m.name}}, Provider: m.name, Query: q.Query}, nil
}
func TestRouterExplicitFallbackAndCancelledRetry(t *testing.T) {
	first := &mockProvider{name: "chosen", errs: []error{&SearchError{Code: ErrProviderDown, Message: "down"}}}
	second := &mockProvider{name: "fallback"}
	r := NewRouter([]Provider{second, first}, WithMaxRetries(0), WithFallback(true), WithRateLimiter(nil))
	got, err := r.SearchWithProvider(context.Background(), "chosen", SearchQuery{Query: "x"})
	if err != nil || got.Provider != "fallback" {
		t.Fatalf("fallback failed: %#v %v", got, err)
	}
	retry := &mockProvider{name: "retry", errs: []error{&SearchError{Code: ErrProviderDown, Message: "down", RetryAfter: time.Second}}}
	r = NewRouter([]Provider{retry}, WithMaxRetries(2), WithRateLimiter(nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	_, err = r.Search(ctx, SearchQuery{Query: "x"})
	if err == nil || time.Since(start) > 200*time.Millisecond {
		t.Fatalf("retry did not honor context: %v", err)
	}
}

func TestDuckDuckGoHTMLAndLiteParsing(t *testing.T) {
	d := NewDuckDuckGoProvider()
	htmlFixture := `<div class="result"><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fa">A title</a><div class="result__snippet">A snippet</div></div>`
	got, err := d.parseHTMLResults(strings.NewReader(htmlFixture), 10)
	if err != nil || len(got) != 1 || got[0].URL != "https://example.com/a" || got[0].Snippet != "A snippet" {
		t.Fatalf("HTML parse: %#v %v", got, err)
	}
	lite := `<table><tr><td><a class="result-link" href="https://example.org/b">Lite title</a></td></tr><tr><td class="result-snippet">Lite snippet</td></tr></table>`
	got, err = d.parseHTMLResults(strings.NewReader(lite), 10)
	if err != nil || len(got) != 1 || got[0].Title != "Lite title" {
		t.Fatalf("lite parse: %#v %v", got, err)
	}
}

func TestSearxNGParsingAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/missing") {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Query().Get("format") != "json" {
			t.Error("missing format")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"T","url":"https://example.com","content":"S"}]}`))
	}))
	defer srv.Close()
	p := NewSearxNGProvider(srv.URL)
	got, err := p.Search(context.Background(), SearchQuery{Query: "q", Count: 1})
	if err != nil || len(got.Results) != 1 {
		t.Fatalf("parse: %#v %v", got, err)
	}
	p.baseURL = srv.URL + "/missing"
	_, err = p.Search(context.Background(), SearchQuery{Query: "q"})
	var se *SearchError
	if err == nil || !errors.As(err, &se) {
		t.Fatalf("expected structured status error: %v", err)
	}
}

func testFetcher(srv *httptest.Server) *Fetcher {
	f := NewFetcher()
	f.allowPrivate = true
	f.httpClient = srv.Client()
	return f
}
func TestFetcherExtractionRedirectLimitAndSSRF(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/page", http.StatusFound)
		case "/large":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(strings.Repeat("x", 20)))
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><head><title>Title</title></head><body><article><p>This is substantial readable fixture content for deterministic extraction.</p></article></body></html>`))
		}
	}))
	defer srv.Close()
	f := testFetcher(srv)
	got, err := f.Fetch(context.Background(), srv.URL+"/redirect", nil)
	if err != nil || !strings.HasSuffix(got.URL, "/page") || got.RawHTML != "" || got.Content == "" {
		t.Fatalf("fetch: %#v %v", got, err)
	}
	_, err = f.Fetch(context.Background(), srv.URL+"/large", &FetchOptions{MaxBytes: 10, Timeout: 2})
	var se *SearchError
	if !errors.As(err, &se) || se.Code != ErrResponseTooLarge {
		t.Fatalf("expected over-limit error: %v", err)
	}
	safe := NewFetcher()
	_, err = safe.Fetch(context.Background(), "http://127.0.0.1/x", nil)
	if !errors.As(err, &se) || se.Code != ErrInvalidQuery {
		t.Fatalf("expected SSRF rejection: %v", err)
	}
}

package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultUA = "Mozilla/5.0 (compatible; browser-server-ai/1.0)"

func readErrorBody(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 4096))
	return strings.TrimSpace(string(b))
}
func providerDecode(name string, r io.Reader, dst any) error {
	if err := json.NewDecoder(io.LimitReader(r, 4<<20)).Decode(dst); err != nil {
		return &SearchError{Provider: name, Code: ErrUnknown, Message: "invalid provider response: " + err.Error()}
	}
	return nil
}
func addDomainTerms(query string, include, exclude []string) string {
	for _, d := range include {
		if d = strings.TrimSpace(d); d != "" {
			query += " site:" + d
		}
	}
	for _, d := range exclude {
		if d = strings.TrimSpace(d); d != "" {
			query += " -site:" + d
		}
	}
	return query
}

type BraveProvider struct {
	apiKey, baseURL string
	httpClient      *http.Client
}

func NewBraveProvider(key string) *BraveProvider {
	return &BraveProvider{key, "https://api.search.brave.com/res/v1/web/search", &http.Client{Timeout: 20 * time.Second}}
}
func (b *BraveProvider) Name() string                     { return "brave" }
func (b *BraveProvider) RequiresAPIKey() bool             { return true }
func (b *BraveProvider) IsAvailable(context.Context) bool { return b.apiKey != "" }
func (b *BraveProvider) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	q.GetDefaults()
	u, err := url.Parse(b.baseURL)
	if err != nil {
		return nil, err
	}
	v := u.Query()
	v.Set("q", addDomainTerms(q.Query, q.IncludeDomains, q.ExcludeDomains))
	v.Set("count", strconv.Itoa(q.Count))
	v.Set("country", q.Country)
	v.Set("search_lang", q.Language)
	if q.SafeSearch {
		v.Set("safesearch", "strict")
	} else {
		v.Set("safesearch", "off")
	}
	fresh := map[TimeRange]string{TimeDay: "pd", TimeWeek: "pw", TimeMonth: "pm", TimeYear: "py"}[q.TimeRange]
	if fresh != "" {
		v.Set("freshness", fresh)
	}
	u.RawQuery = v.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	req.Header.Set("X-Subscription-Token", b.apiKey)
	req.Header.Set("Accept", "application/json")
	start := time.Now()
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, &SearchError{b.Name(), ErrNetworkError, err.Error(), 0}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(b.Name(), resp.StatusCode, readErrorBody(resp.Body))
	}
	var x struct {
		Web struct {
			Results      []struct{ Title, URL, Description string }
			TotalResults int `json:"total_results"`
		} `json:"web"`
	}
	if err = providerDecode(b.Name(), resp.Body, &x); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(x.Web.Results))
	for _, r := range x.Web.Results {
		out = append(out, SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Description, Source: b.Name()})
	}
	return &SearchResponse{out, q.Query, b.Name(), time.Since(start), x.Web.TotalResults}, nil
}

type TavilyProvider struct {
	apiKey, baseURL string
	httpClient      *http.Client
}

func NewTavilyProvider(key string) *TavilyProvider {
	return &TavilyProvider{key, "https://api.tavily.com/search", &http.Client{Timeout: 30 * time.Second}}
}
func (t *TavilyProvider) Name() string                     { return "tavily" }
func (t *TavilyProvider) RequiresAPIKey() bool             { return true }
func (t *TavilyProvider) IsAvailable(context.Context) bool { return t.apiKey != "" }
func (t *TavilyProvider) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	q.GetDefaults()
	payload := struct {
		Query          string   `json:"query"`
		SearchDepth    string   `json:"search_depth"`
		MaxResults     int      `json:"max_results"`
		IncludeDomains []string `json:"include_domains,omitempty"`
		ExcludeDomains []string `json:"exclude_domains,omitempty"`
	}{q.Query, "basic", q.Count, q.IncludeDomains, q.ExcludeDomains}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", t.baseURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, &SearchError{t.Name(), ErrNetworkError, err.Error(), 0}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(t.Name(), resp.StatusCode, readErrorBody(resp.Body))
	}
	var x struct {
		Results []struct {
			Title, URL, Content string
			Score               float64
		}
	}
	if err = providerDecode(t.Name(), resp.Body, &x); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(x.Results))
	for _, r := range x.Results {
		out = append(out, SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Content, Source: t.Name(), Score: r.Score})
	}
	return &SearchResponse{out, q.Query, t.Name(), time.Since(start), len(out)}, nil
}

type GoogleProvider struct {
	apiKey, searchID, baseURL string
	client                    *http.Client
}

func NewGoogleProvider(key, id string) *GoogleProvider {
	return &GoogleProvider{key, id, "https://www.googleapis.com/customsearch/v1", &http.Client{Timeout: 20 * time.Second}}
}
func (g *GoogleProvider) Name() string                     { return "google" }
func (g *GoogleProvider) RequiresAPIKey() bool             { return true }
func (g *GoogleProvider) IsAvailable(context.Context) bool { return g.apiKey != "" && g.searchID != "" }
func (g *GoogleProvider) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	q.GetDefaults()
	u, _ := url.Parse(g.baseURL)
	v := u.Query()
	v.Set("key", g.apiKey)
	v.Set("cx", g.searchID)
	v.Set("q", addDomainTerms(q.Query, q.IncludeDomains, q.ExcludeDomains))
	v.Set("num", strconv.Itoa(min(q.Count, 10)))
	v.Set("lr", "lang_"+q.Language)
	v.Set("gl", q.Country)
	if q.SafeSearch {
		v.Set("safe", "active")
	}
	dates := map[TimeRange]string{TimeDay: "d1", TimeWeek: "w1", TimeMonth: "m1", TimeYear: "y1"}
	if dates[q.TimeRange] != "" {
		v.Set("dateRestrict", dates[q.TimeRange])
	}
	u.RawQuery = v.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	start := time.Now()
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, &SearchError{g.Name(), ErrNetworkError, err.Error(), 0}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(g.Name(), resp.StatusCode, readErrorBody(resp.Body))
	}
	var x struct {
		SearchInformation struct {
			TotalResults string `json:"totalResults"`
		} `json:"searchInformation"`
		Items []struct{ Title, Link, Snippet string }
	}
	if err = providerDecode(g.Name(), resp.Body, &x); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(x.Items))
	for _, r := range x.Items {
		out = append(out, SearchResult{Title: r.Title, URL: r.Link, Snippet: r.Snippet, Source: g.Name()})
	}
	total, _ := strconv.Atoi(x.SearchInformation.TotalResults)
	return &SearchResponse{out, q.Query, g.Name(), time.Since(start), total}, nil
}

type SearxNGProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewSearxNGProvider(base string) *SearxNGProvider {
	if base == "" {
		base = "http://localhost:8888"
	}
	return &SearxNGProvider{strings.TrimRight(base, "/"), &http.Client{Timeout: 20 * time.Second}}
}
func (s *SearxNGProvider) Name() string                     { return "searxng" }
func (s *SearxNGProvider) RequiresAPIKey() bool             { return false }
func (s *SearxNGProvider) IsAvailable(context.Context) bool { return s.baseURL != "" }
func (s *SearxNGProvider) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	q.GetDefaults()
	v := url.Values{"q": {addDomainTerms(q.Query, q.IncludeDomains, q.ExcludeDomains)}, "format": {"json"}, "language": {q.Language}}
	if q.SafeSearch {
		v.Set("safesearch", "2")
	} else {
		v.Set("safesearch", "0")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/search?"+v.Encode(), nil)
	if err != nil {
		return nil, &SearchError{s.Name(), ErrInvalidQuery, err.Error(), 0}
	}
	req.Header.Set("User-Agent", defaultUA)
	start := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &SearchError{s.Name(), ErrNetworkError, err.Error(), 0}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(s.Name(), resp.StatusCode, readErrorBody(resp.Body))
	}
	var x struct {
		Results []struct{ Title, URL, Content string }
	}
	if err = providerDecode(s.Name(), resp.Body, &x); err != nil {
		return nil, err
	}
	n := min(len(x.Results), q.Count)
	out := make([]SearchResult, 0, n)
	for _, r := range x.Results[:n] {
		out = append(out, SearchResult{Title: r.Title, URL: r.URL, Snippet: r.Content, Source: s.Name()})
	}
	return &SearchResponse{out, q.Query, s.Name(), time.Since(start), len(x.Results)}, nil
}

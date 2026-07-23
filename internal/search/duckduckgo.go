package search

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type DuckDuckGoProvider struct {
	httpClient                  *http.Client
	userAgent, htmlURL, liteURL string
}

func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{&http.Client{Timeout: 15 * time.Second}, defaultUA, "https://html.duckduckgo.com/html/", "https://lite.duckduckgo.com/lite/"}
}
func (d *DuckDuckGoProvider) Name() string                     { return "duckduckgo" }
func (d *DuckDuckGoProvider) RequiresAPIKey() bool             { return false }
func (d *DuckDuckGoProvider) IsAvailable(context.Context) bool { return true }
func (d *DuckDuckGoProvider) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	q.GetDefaults()
	start := time.Now()
	results, err := d.searchEndpoint(ctx, d.htmlURL, q)
	if err != nil {
		results, err = d.searchEndpoint(ctx, d.liteURL, q)
	}
	if err != nil {
		return nil, err
	}
	return &SearchResponse{results, q.Query, d.Name(), time.Since(start), len(results)}, nil
}
func (d *DuckDuckGoProvider) searchEndpoint(ctx context.Context, endpoint string, q SearchQuery) ([]SearchResult, error) {
	form := url.Values{"q": {addDomainTerms(q.Query, q.IncludeDomains, q.ExcludeDomains)}}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", d.userAgent)
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, &SearchError{d.Name(), ErrNetworkError, err.Error(), 0}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(d.Name(), resp.StatusCode, readErrorBody(resp.Body))
	}
	return d.parseHTMLResults(resp.Body, q.Count)
}
func ddgURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		u.Scheme = "https"
	}
	if strings.Contains(strings.ToLower(u.Host), "duckduckgo.com") {
		if target := u.Query().Get("uddg"); target != "" {
			if decoded, e := url.QueryUnescape(target); e == nil {
				return decoded
			}
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}
func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
			b.WriteByte(' ')
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func hasClass(n *html.Node, want ...string) bool {
	classes := strings.Fields(attr(n, "class"))
	for _, c := range classes {
		for _, w := range want {
			if c == w {
				return true
			}
		}
	}
	return false
}
func (d *DuckDuckGoProvider) parseHTMLResults(r io.Reader, max int) ([]SearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, &SearchError{d.Name(), ErrUnknown, "invalid HTML: " + err.Error(), 0}
	}
	var out []SearchResult
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(out) >= max {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && (hasClass(n, "result__a", "result-link") || strings.Contains(attr(n, "class"), "result-link")) {
			u := ddgURL(attr(n, "href"))
			title := nodeText(n)
			if u != "" && title != "" && !seen[u] {
				snippet := ""
				container := n.Parent
				for container != nil && container.Data != "body" {
					var find func(*html.Node)
					find = func(x *html.Node) {
						if snippet == "" && x.Type == html.ElementNode && (hasClass(x, "result__snippet", "result-snippet") || strings.Contains(attr(x, "class"), "snippet")) {
							snippet = nodeText(x)
						}
						for c := x.FirstChild; c != nil; c = c.NextSibling {
							find(c)
						}
					}
					find(container)
					if snippet != "" {
						break
					}
					container = container.Parent
				}
				seen[u] = true
				out = append(out, SearchResult{Title: title, URL: u, Snippet: snippet, Source: d.Name()})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(out) == 0 {
		return nil, &SearchError{d.Name(), ErrNoResults, "no results parsed", 0}
	}
	return out, nil
}

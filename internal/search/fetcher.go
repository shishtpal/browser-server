package search

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// Content and TextContent are both part of FetchResult, so cap each such that
// the complete JSON result remains below the tool registry's 32 KiB ceiling.
const maxFetchOutput = 13 * 1024

type Fetcher struct {
	httpClient   *http.Client
	defaults     FetchOptions
	resolver     *net.Resolver
	allowPrivate bool
}

func NewFetcher() *Fetcher {
	f := &Fetcher{defaults: FetchOptions{MaxBytes: 5 << 20, Timeout: 30, FollowRedirects: true, ExtractContent: true, UserAgent: defaultUA}, resolver: net.DefaultResolver}
	f.httpClient = &http.Client{Transport: f.safeTransport(), Timeout: 30 * time.Second}
	return f
}
func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
func (f *Fetcher) resolvePublic(ctx context.Context, host string) ([]net.IP, error) {
	ips, err := f.resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host has no addresses")
	}
	for _, ip := range ips {
		if blockedIP(ip) && !f.allowPrivate {
			return nil, fmt.Errorf("target resolves to a non-public address")
		}
	}
	return ips, nil
}
func (f *Fetcher) safeTransport() *http.Transport {
	return &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := f.resolvePublic(ctx, host)
		if err != nil {
			return nil, err
		}
		var d net.Dialer
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}, TLSHandshakeTimeout: 10 * time.Second}
}
func (f *Fetcher) validate(ctx context.Context, u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return &SearchError{Code: ErrInvalidQuery, Message: "URL must be an http(s) URL without credentials"}
	}
	if strings.EqualFold(u.Hostname(), "localhost") {
		return &SearchError{Code: ErrInvalidQuery, Message: "localhost is not allowed"}
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		if blockedIP(ip) && !f.allowPrivate {
			return &SearchError{Code: ErrInvalidQuery, Message: "non-public target is not allowed"}
		}
		return nil
	}
	if _, err := f.resolvePublic(ctx, u.Hostname()); err != nil {
		return &SearchError{Code: ErrInvalidQuery, Message: "unsafe or unresolved target: " + err.Error()}
	}
	return nil
}
func fetchDefaults(base FetchOptions, in *FetchOptions) FetchOptions {
	if in == nil {
		return base
	}
	o := *in
	if o.MaxBytes <= 0 {
		o.MaxBytes = base.MaxBytes
	}
	if o.Timeout <= 0 {
		o.Timeout = base.Timeout
	}
	if o.UserAgent == "" {
		o.UserAgent = base.UserAgent
	}
	return o
}
func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "\n\n... [truncated]"
}
func (f *Fetcher) Fetch(ctx context.Context, target string, in *FetchOptions) (*FetchResult, error) {
	start := time.Now()
	if len(target) > 4096 {
		return nil, &SearchError{Code: ErrInvalidQuery, Message: "URL is too long"}
	}
	u, err := url.Parse(target)
	if err != nil {
		return nil, &SearchError{Code: ErrInvalidQuery, Message: err.Error()}
	}
	if err = f.validate(ctx, u); err != nil {
		return nil, err
	}
	o := fetchDefaults(f.defaults, in)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(o.Timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, &SearchError{Code: ErrInvalidQuery, Message: err.Error()}
	}
	req.Header.Set("User-Agent", o.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.1")
	client := *f.httpClient
	client.Timeout = time.Duration(o.Timeout) * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !o.FollowRedirects {
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return f.validate(req.Context(), req.URL)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &SearchError{Code: ErrNetworkError, Message: err.Error()}
	}
	defer resp.Body.Close()
	result := &FetchResult{URL: resp.Request.URL.String(), StatusCode: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), FetchTime: time.Since(start)}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result, nil
	}
	limited := io.LimitReader(resp.Body, o.MaxBytes+1)
	reader, encErr := charset.NewReader(limited, result.ContentType)
	if encErr != nil {
		reader = limited
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, &SearchError{Code: ErrNetworkError, Message: err.Error()}
	}
	if int64(len(body)) > o.MaxBytes {
		return nil, &SearchError{Code: ErrResponseTooLarge, Message: fmt.Sprintf("response exceeds %d bytes", o.MaxBytes)}
	}
	result.Size = int64(len(body))
	text := string(body)
	if o.ExtractContent && strings.Contains(strings.ToLower(result.ContentType), "html") {
		article, e := readability.FromReader(strings.NewReader(text), resp.Request.URL)
		if e == nil {
			result.Title = truncateUTF8(article.Title, 512)
			result.TextContent = strings.TrimSpace(article.TextContent)
			result.Content = result.TextContent
			if result.Title != "" {
				result.Content = "# " + result.Title + "\n\n" + result.Content
			}
		} else {
			result.TextContent = strings.TrimSpace(nodeTextFromHTML(text))
			result.Content = result.TextContent
		}
	} else {
		result.TextContent = text
		result.Content = text
	}
	result.Content = truncateUTF8(result.Content, maxFetchOutput)
	result.TextContent = truncateUTF8(result.TextContent, maxFetchOutput)
	result.FetchTime = time.Since(start)
	return result, nil
}
func nodeTextFromHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return nodeText(doc)
}

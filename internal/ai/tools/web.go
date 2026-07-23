package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"browser-server/internal/ai/config"
	"browser-server/internal/search"
)

const maxWebToolText = 24 * 1024

type webTools struct {
	router          *search.Router
	fetcher         *search.Fetcher
	defaultProvider string
	maxResults      int
}

func registerWebTools(r *Registry, cfg config.WebSearchConfig) {
	providers := make([]search.Provider, 0, 5)
	if cfg.Providers.Tavily.Enabled {
		providers = append(providers, search.NewTavilyProvider(cfg.Providers.Tavily.APIKey))
	}
	if cfg.Providers.Brave.Enabled {
		providers = append(providers, search.NewBraveProvider(cfg.Providers.Brave.APIKey))
	}
	if cfg.Providers.Google.Enabled {
		providers = append(providers, search.NewGoogleProvider(cfg.Providers.Google.APIKey, cfg.Providers.Google.SearchEngineID))
	}
	if cfg.Providers.SearxNG.Enabled {
		providers = append(providers, search.NewSearxNGProvider(cfg.Providers.SearxNG.BaseURL))
	}
	if cfg.Providers.DuckDuckGo.Enabled {
		providers = append(providers, search.NewDuckDuckGoProvider())
	}

	w := &webTools{
		router: search.NewRouter(providers,
			search.WithCache(time.Duration(cfg.CacheTTLMinutes)*time.Minute, cfg.CacheMaxEntries),
			search.WithTimeout(time.Duration(cfg.TimeoutSeconds)*time.Second),
			search.WithFallback(cfg.Fallback),
		),
		fetcher:         search.NewFetcher(),
		defaultProvider: cfg.DefaultProvider,
		maxResults:      cfg.MaxResults,
	}
	r.add(Tool{
		Name:        "web_search",
		Category:    "Web",
		Description: "Search the web for current information. Returns titles, URLs, and snippets. Use for up-to-date documentation, news, releases, or other time-sensitive facts.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Precise search query","minLength":1,"maxLength":1000},"max_results":{"type":"integer","description":"Number of results","minimum":1,"maximum":20},"provider":{"type":"string","enum":["auto","brave","tavily","google","searxng","duckduckgo"]},"time_range":{"type":"string","enum":["day","week","month","year"]},"country":{"type":"string","description":"Two-letter country code"},"language":{"type":"string","description":"Two-letter language code"},"safe_search":{"type":"boolean","default":true},"include_domains":{"type":"array","items":{"type":"string"},"maxItems":20},"exclude_domains":{"type":"array","items":{"type":"string"},"maxItems":20}},"required":["query"],"additionalProperties":false}`),
		Execute:     w.search,
	})
	r.add(Tool{
		Name:        "web_fetch",
		Category:    "Web",
		Description: "Fetch and extract readable content from a public URL. Use after web_search to read a specific article or documentation page.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"Public HTTP or HTTPS URL","maxLength":4096},"extract_content":{"type":"boolean","default":true},"max_chars":{"type":"integer","description":"Maximum characters to return","default":10000,"minimum":1000,"maximum":24000}},"required":["url"],"additionalProperties":false}`),
		Execute:     w.fetch,
	})
}

func (w *webTools) search(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Query          string   `json:"query"`
		MaxResults     int      `json:"max_results"`
		Provider       string   `json:"provider"`
		TimeRange      string   `json:"time_range"`
		Country        string   `json:"country"`
		Language       string   `json:"language"`
		SafeSearch     *bool    `json:"safe_search"`
		IncludeDomains []string `json:"include_domains"`
		ExcludeDomains []string `json:"exclude_domains"`
	}
	if err := strict(raw, &a, map[string]bool{
		"query": true, "max_results": true, "provider": true, "time_range": true,
		"country": true, "language": true, "safe_search": true,
		"include_domains": true, "exclude_domains": true,
	}); err != nil {
		return nil, err
	}
	a.Query = strings.TrimSpace(a.Query)
	if a.Query == "" || len(a.Query) > 1000 {
		return nil, fmt.Errorf("query must contain 1 to 1000 characters")
	}
	if a.MaxResults == 0 {
		a.MaxResults = w.maxResults
	}
	if a.MaxResults < 1 || a.MaxResults > 20 {
		return nil, fmt.Errorf("max_results must be between 1 and 20")
	}
	if len(a.IncludeDomains) > 20 || len(a.ExcludeDomains) > 20 {
		return nil, fmt.Errorf("include_domains and exclude_domains accept at most 20 entries")
	}
	if !validTimeRange(a.TimeRange) {
		return nil, fmt.Errorf("time_range must be day, week, month, or year")
	}
	if a.Provider == "" {
		a.Provider = w.defaultProvider
	}
	if !validSearchProvider(a.Provider) {
		return nil, fmt.Errorf("unsupported search provider %q", a.Provider)
	}
	safe := true
	if a.SafeSearch != nil {
		safe = *a.SafeSearch
	}
	resp, err := w.router.SearchWithProvider(ctx, a.Provider, search.SearchQuery{
		Query: a.Query, Count: a.MaxResults, Country: a.Country, Language: a.Language,
		TimeRange: search.TimeRange(a.TimeRange), SafeSearch: safe,
		IncludeDomains: a.IncludeDomains, ExcludeDomains: a.ExcludeDomains,
	})
	if err != nil {
		return nil, fmt.Errorf("web search failed: %w", err)
	}
	return formatSearchResults(resp), nil
}

func (w *webTools) fetch(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		URL            string `json:"url"`
		ExtractContent *bool  `json:"extract_content"`
		MaxChars       int    `json:"max_chars"`
	}
	if err := strict(raw, &a, map[string]bool{"url": true, "extract_content": true, "max_chars": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.URL) == "" {
		return nil, fmt.Errorf("url is required")
	}
	if a.MaxChars == 0 {
		a.MaxChars = 10000
	}
	if a.MaxChars < 1000 || a.MaxChars > maxWebToolText {
		return nil, fmt.Errorf("max_chars must be between 1000 and %d", maxWebToolText)
	}
	extract := true
	if a.ExtractContent != nil {
		extract = *a.ExtractContent
	}
	result, err := w.fetcher.Fetch(ctx, a.URL, &search.FetchOptions{
		MaxBytes: 5 << 20, Timeout: 30, FollowRedirects: true, ExtractContent: extract,
	})
	if err != nil {
		return nil, fmt.Errorf("web fetch failed: %w", err)
	}
	if result.Error != "" {
		return map[string]any{"url": result.URL, "status": result.StatusCode, "error": result.Error}, nil
	}
	content := truncateWebText(result.Content, a.MaxChars)
	var b strings.Builder
	if result.Title != "" {
		fmt.Fprintf(&b, "# %s\n\n", result.Title)
	}
	fmt.Fprintf(&b, "URL: %s\nSize: %s | Type: %s | Fetch: %dms\n\n", result.URL, formatBytes(result.Size), result.ContentType, result.FetchTime.Milliseconds())
	b.WriteString(content)
	return truncateWebText(b.String(), maxWebToolText), nil
}

func validTimeRange(value string) bool {
	return value == "" || value == "day" || value == "week" || value == "month" || value == "year"
}

func validSearchProvider(value string) bool {
	return value == "auto" || value == "brave" || value == "tavily" || value == "google" || value == "searxng" || value == "duckduckgo"
}

func formatSearchResults(resp *search.SearchResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Search results for %q (via %s, %dms)\n\n", resp.Query, resp.Provider, resp.Duration.Milliseconds())
	for i, result := range resp.Results {
		fmt.Fprintf(&b, "%d. %s\n   URL: %s\n", i+1, strings.TrimSpace(result.Title), result.URL)
		if result.Snippet != "" {
			fmt.Fprintf(&b, "   %s\n", strings.TrimSpace(result.Snippet))
		}
		if result.PublishedAt != nil {
			fmt.Fprintf(&b, "   Published: %s\n", result.PublishedAt.Format("2006-01-02"))
		}
		b.WriteByte('\n')
		if b.Len() >= maxWebToolText-128 {
			break
		}
	}
	b.WriteString("---\nUse web_fetch to read the full content of a result URL.")
	return truncateWebText(b.String(), maxWebToolText)
}

func truncateWebText(value string, max int) string {
	if len(value) <= max {
		return value
	}
	value = value[:max]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + "\n\n... [truncated]"
}

func formatBytes(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

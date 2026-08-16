// Package openrouter centralizes how requests to OpenRouter are identified and
// tagged with attribution headers, so the chat, image, and video code paths all
// make the same request-shape decisions from one implementation.
package openrouter

import (
	"net/http"
	"net/url"
	"strings"
)

// IsOpenRouterBaseURL reports whether a base URL targets OpenRouter, in which
// case the attribution headers should be attached. Matching is host-based and
// exact: the bare host "openrouter.ai" or any subdomain ending in
// ".openrouter.ai" (e.g. api.openrouter.ai) qualifies. Lookalike hosts such as
// "myopenrouter.ai" or suffix tricks like "openrouter.ai.evil.example.com"
// contain the substring but are NOT OpenRouter, so they never match and never
// receive attribution headers.
func IsOpenRouterBaseURL(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "openrouter.ai" || strings.HasSuffix(host, ".openrouter.ai")
}

// SetAttributionHeaders attaches the OpenRouter attribution headers
// (HTTP-Referer, Referer, X-Title) when the request targets OpenRouter. Other
// providers get none, and an empty site URL (no attribution configured) skips
// them entirely. An empty app name omits X-Title rather than sending an empty
// value, since OpenRouter expects a real title.
func SetAttributionHeaders(h http.Header, baseURL, siteURL, appName string) {
	if siteURL == "" || !IsOpenRouterBaseURL(baseURL) {
		return
	}
	h.Set("HTTP-Referer", siteURL)
	h.Set("Referer", siteURL)
	if appName != "" {
		h.Set("X-Title", appName)
	}
}

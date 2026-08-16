package openrouter

import (
	"net/http"
	"testing"
)

func TestIsOpenRouterBaseURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://openrouter.ai/api/v1", true},
		{"https://openrouter.ai", true},
		{"https://openrouter.ai/", true},
		{"https://api.openrouter.ai/api/v1", true},
		{"https://my.openrouter.ai", true},
		{"http://openrouter.ai:8080/api/v1", true},
		{"https://myopenrouter.ai", false},
		{"https://openrouter.ai.evil.example.com", false},
		{"https://generativelanguage.googleapis.com/v1beta", false},
		{"https://apihub.agnes-ai.com/v1", false},
		{"not-a-url", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsOpenRouterBaseURL(c.url); got != c.want {
			t.Errorf("IsOpenRouterBaseURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestSetAttributionHeaders(t *testing.T) {
	const (
		siteURL = "https://example.com/app"
		appName = "My App"
	)
	t.Run("openrouter target attaches attribution", func(t *testing.T) {
		h := http.Header{}
		SetAttributionHeaders(h, "https://openrouter.ai/api/v1", siteURL, appName)
		if h.Get("HTTP-Referer") != siteURL {
			t.Errorf("HTTP-Referer = %q, want %q", h.Get("HTTP-Referer"), siteURL)
		}
		if h.Get("Referer") != siteURL {
			t.Errorf("Referer = %q, want %q", h.Get("Referer"), siteURL)
		}
		if h.Get("X-Title") != appName {
			t.Errorf("X-Title = %q, want %q", h.Get("X-Title"), appName)
		}
	})
	t.Run("non-openrouter target gets nothing", func(t *testing.T) {
		h := http.Header{}
		SetAttributionHeaders(h, "https://apihub.agnes-ai.com/v1", siteURL, appName)
		if h.Get("HTTP-Referer") != "" || h.Get("Referer") != "" || h.Get("X-Title") != "" {
			t.Errorf("non-openrouter request got attribution headers: %v", h)
		}
	})
	t.Run("empty app name omits X-Title", func(t *testing.T) {
		h := http.Header{}
		SetAttributionHeaders(h, "https://openrouter.ai/api/v1", siteURL, "")
		if h.Get("HTTP-Referer") != siteURL {
			t.Errorf("HTTP-Referer = %q, want %q", h.Get("HTTP-Referer"), siteURL)
		}
		if h.Get("X-Title") != "" {
			t.Errorf("X-Title = %q, want empty", h.Get("X-Title"))
		}
	})
	t.Run("empty site url skips everything", func(t *testing.T) {
		h := http.Header{}
		SetAttributionHeaders(h, "https://openrouter.ai/api/v1", "", appName)
		if h.Get("HTTP-Referer") != "" || h.Get("Referer") != "" || h.Get("X-Title") != "" {
			t.Errorf("empty site_url still set headers: %v", h)
		}
	})
}

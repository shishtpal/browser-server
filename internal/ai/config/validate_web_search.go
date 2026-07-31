package config

import (
	"fmt"
	"net/url"
	"strings"
)

func validateWebSearch(cfg WebSearchConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.TimeoutSeconds < 1 || cfg.TimeoutSeconds > 120 {
		return fmt.Errorf("web_search.timeout_seconds must be between 1 and 120")
	}
	if cfg.MaxResults < 1 || cfg.MaxResults > 20 {
		return fmt.Errorf("web_search.max_results must be between 1 and 20")
	}
	if cfg.CacheTTLMinutes < 1 || cfg.CacheTTLMinutes > 1440 {
		return fmt.Errorf("web_search.cache_ttl_minutes must be between 1 and 1440")
	}
	if cfg.CacheMaxEntries < 1 || cfg.CacheMaxEntries > 10000 {
		return fmt.Errorf("web_search.cache_max_entries must be between 1 and 10000")
	}
	available := map[string]bool{
		"brave":      cfg.Providers.Brave.Enabled,
		"tavily":     cfg.Providers.Tavily.Enabled,
		"google":     cfg.Providers.Google.Enabled,
		"searxng":    cfg.Providers.SearxNG.Enabled,
		"duckduckgo": cfg.Providers.DuckDuckGo.Enabled,
	}
	if cfg.DefaultProvider != "auto" && !available[cfg.DefaultProvider] {
		return fmt.Errorf("web_search.default_provider %q is not enabled", cfg.DefaultProvider)
	}
	if cfg.Providers.Brave.Enabled && strings.TrimSpace(cfg.Providers.Brave.APIKey) == "" {
		return fmt.Errorf("web_search provider \"brave\" api_key is required")
	}
	if cfg.Providers.Tavily.Enabled && strings.TrimSpace(cfg.Providers.Tavily.APIKey) == "" {
		return fmt.Errorf("web_search provider \"tavily\" api_key is required")
	}
	if cfg.Providers.Google.Enabled && (strings.TrimSpace(cfg.Providers.Google.APIKey) == "" || strings.TrimSpace(cfg.Providers.Google.SearchEngineID) == "") {
		return fmt.Errorf("web_search provider \"google\" api_key and search_engine_id are required")
	}
	if cfg.Providers.SearxNG.Enabled {
		u, err := url.Parse(cfg.Providers.SearxNG.BaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "https" && !isLocalHost(u.Hostname())) {
			return fmt.Errorf("web_search provider \"searxng\" base_url must be a valid HTTPS or local URL")
		}
	}
	for _, enabled := range available {
		if enabled {
			return nil
		}
	}
	return fmt.Errorf("web_search must enable at least one provider")
}

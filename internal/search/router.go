package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Router struct {
	providers   []Provider
	cache       *SearchCache
	rateLimiter *RateLimiter
	timeout     time.Duration
	maxRetries  int
	fallback    bool
	retryBase   time.Duration
}
type RouterOption func(*Router)

func WithCache(ttl time.Duration, n int) RouterOption {
	return func(r *Router) { r.cache = NewSearchCache(ttl, n) }
}
func WithTimeout(d time.Duration) RouterOption {
	return func(r *Router) {
		if d > 0 {
			r.timeout = d
		}
	}
}
func WithFallback(v bool) RouterOption { return func(r *Router) { r.fallback = v } }
func WithMaxRetries(n int) RouterOption {
	return func(r *Router) {
		if n >= 0 {
			r.maxRetries = n
		}
	}
}
func WithRateLimiter(l *RateLimiter) RouterOption { return func(r *Router) { r.rateLimiter = l } }
func NewRouter(p []Provider, opts ...RouterOption) *Router {
	r := &Router{providers: append([]Provider(nil), p...), rateLimiter: NewRateLimiter(), timeout: 30 * time.Second, maxRetries: 2, fallback: true, retryBase: time.Second}
	for _, o := range opts {
		o(r)
	}
	return r
}
func (r *Router) SelectProvider(name string) Provider {
	for _, p := range r.ordered(name) {
		if p.IsAvailable(context.Background()) {
			return p
		}
	}
	return nil
}
func (r *Router) ordered(selected string) []Provider {
	if selected != "" && selected != "auto" {
		var out []Provider
		for _, p := range r.providers {
			if strings.EqualFold(p.Name(), selected) {
				out = append(out, p)
			}
		}
		if r.fallback {
			for _, p := range r.providers {
				if !strings.EqualFold(p.Name(), selected) {
					out = append(out, p)
				}
			}
		}
		return out
	}
	priority := []string{"tavily", "brave", "google", "searxng", "duckduckgo"}
	out := make([]Provider, 0, len(r.providers))
	used := map[Provider]bool{}
	for _, n := range priority {
		for _, p := range r.providers {
			if p.Name() == n {
				out = append(out, p)
				used[p] = true
			}
		}
	}
	for _, p := range r.providers {
		if !used[p] {
			out = append(out, p)
		}
	}
	return out
}
func (r *Router) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	return r.search(ctx, q, "auto")
}
func (r *Router) SearchWithProvider(ctx context.Context, name string, q SearchQuery) (*SearchResponse, error) {
	if name == "" {
		name = "auto"
	}
	return r.search(ctx, q, name)
}
func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func (r *Router) search(ctx context.Context, q SearchQuery, selected string) (*SearchResponse, error) {
	q.GetDefaults()
	if strings.TrimSpace(q.Query) == "" {
		return nil, &SearchError{Code: ErrInvalidQuery, Message: "query is required"}
	}
	cacheQ := q
	cacheQ.Query = selected + "\x00" + q.Query
	if r.cache != nil {
		if v, ok := r.cache.Get(cacheQ); ok {
			return v, nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	providers := r.ordered(selected)
	if len(providers) == 0 {
		return nil, &SearchError{Code: ErrProviderDown, Message: fmt.Sprintf("provider %q is not configured", selected)}
	}
	var last error
	for _, p := range providers {
		if !p.IsAvailable(ctx) {
			continue
		}
		for attempt := 0; attempt <= r.maxRetries; attempt++ {
			if err := ctx.Err(); err != nil {
				return nil, &SearchError{Provider: p.Name(), Code: ErrTimeout, Message: err.Error()}
			}
			if r.rateLimiter != nil {
				if err := r.rateLimiter.Allow(ctx, p.Name()); err != nil {
					return nil, &SearchError{Provider: p.Name(), Code: ErrTimeout, Message: err.Error()}
				}
			}
			resp, err := p.Search(ctx, q)
			if err == nil && resp != nil && len(resp.Results) > 0 {
				if r.cache != nil {
					r.cache.Set(cacheQ, resp)
				}
				return resp, nil
			}
			if err == nil {
				err = &SearchError{Provider: p.Name(), Code: ErrNoResults, Message: "provider returned no results"}
			}
			last = err
			var se *SearchError
			if !errors.As(err, &se) || !se.Retryable() || attempt == r.maxRetries {
				break
			}
			delay := r.retryBase * time.Duration(attempt+1)
			if se.RetryAfter > 0 {
				delay = se.RetryAfter
			}
			if err = waitContext(ctx, delay); err != nil {
				return nil, &SearchError{Provider: p.Name(), Code: ErrTimeout, Message: err.Error()}
			}
		}
		if !r.fallback {
			break
		}
	}
	if last != nil {
		return nil, last
	}
	return nil, &SearchError{Code: ErrNoResults, Message: "all providers returned no results"}
}

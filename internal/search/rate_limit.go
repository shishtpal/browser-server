package search

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter { return &RateLimiter{limiters: map[string]*rate.Limiter{}} }
func (r *RateLimiter) GetLimiter(name string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l := r.limiters[name]; l != nil {
		return l
	}
	d := time.Second
	b := 1
	switch name {
	case "tavily":
		d = 12 * time.Second
	case "searxng":
		d = 500 * time.Millisecond
		b = 5
	case "duckduckgo":
		d = 2 * time.Second
	}
	l := rate.NewLimiter(rate.Every(d), b)
	r.limiters[name] = l
	return l
}
func (r *RateLimiter) Allow(ctx context.Context, name string) error {
	return r.GetLimiter(name).Wait(ctx)
}

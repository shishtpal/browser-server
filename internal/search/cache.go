package search

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

type cacheEntry struct {
	response             *SearchResponse
	expiresAt, createdAt time.Time
}
type SearchCache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int
}

func NewSearchCache(ttl time.Duration, maxSize int) *SearchCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if maxSize <= 0 {
		maxSize = 100
	}
	return &SearchCache{entries: map[string]*cacheEntry{}, ttl: ttl, maxSize: maxSize}
}
func (c *SearchCache) key(q SearchQuery) string {
	q.GetDefaults()
	b, _ := json.Marshal(q)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func cloneResponse(r *SearchResponse) *SearchResponse {
	if r == nil {
		return nil
	}
	out := *r
	out.Results = append([]SearchResult(nil), r.Results...)
	return &out
}
func (c *SearchCache) Get(q SearchQuery) (*SearchResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.key(q)
	e, ok := c.entries[k]
	if !ok {
		return nil, false
	}
	if !time.Now().Before(e.expiresAt) {
		delete(c.entries, k)
		return nil, false
	}
	return cloneResponse(e.response), true
}
func (c *SearchCache) Set(q SearchQuery, r *SearchResponse) {
	if r == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.entries {
		if !now.Before(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	k := c.key(q)
	if _, exists := c.entries[k]; !exists && len(c.entries) >= c.maxSize {
		var oldest string
		var at time.Time
		for key, e := range c.entries {
			if oldest == "" || e.createdAt.Before(at) {
				oldest, at = key, e.createdAt
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[k] = &cacheEntry{cloneResponse(r), now.Add(c.ttl), now}
}
func (c *SearchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]*cacheEntry{}
}

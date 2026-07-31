package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (s *memoryStore) manageCache(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Action  string `json:"action"`
		MaxAge  string `json:"max_age"`
		MinSize int    `json:"min_size"`
		MaxSize int    `json:"max_size"`
	}
	if e := strict(raw, &a, map[string]bool{"action": true, "max_age": true, "min_size": true, "max_size": true}); e != nil {
		return nil, e
	}
	if e := s.ensure(); e != nil {
		return nil, e
	}
	type entry struct {
		p string
		t time.Time
		n int64
	}
	es := []entry{}
	var total int64
	ds, e := os.ReadDir(s.cache)
	if e != nil {
		return nil, e
	}
	for _, x := range ds {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		i, e := x.Info()
		if e == nil && !x.IsDir() {
			es = append(es, entry{filepath.Join(s.cache, x.Name()), i.ModTime(), i.Size()})
			total += i.Size()
		}
	}
	removed := 0
	switch a.Action {
	case "stats":
	case "cleanup":
		d := 24 * time.Hour
		if a.MaxAge != "" {
			d, e = time.ParseDuration(a.MaxAge)
			if e != nil {
				return nil, fmt.Errorf("invalid max_age")
			}
		}
		for _, x := range es {
			if time.Since(x.t) > d {
				if os.Remove(x.p) == nil {
					total -= x.n
					removed++
				}
			}
		}
	case "optimize":
		limit := s.cacheLimit
		if a.MaxSize > 0 {
			limit = int64(a.MaxSize) * 1024
		}
		sort.Slice(es, func(i, j int) bool { return es[i].t.Before(es[j].t) })
		for _, x := range es {
			if total <= limit {
				break
			}
			if os.Remove(x.p) == nil {
				total -= x.n
				removed++
			}
		}
	default:
		return nil, fmt.Errorf("unknown action: %s (valid: cleanup, stats, optimize)", a.Action)
	}
	return map[string]any{"entries": len(es) - removed, "size_bytes": total, "removed": removed, "limit_bytes": s.cacheLimit}, nil
}

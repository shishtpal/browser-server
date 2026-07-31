package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *memoryStore) resolve(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		MemoryID string `json:"memory_id"`
		Depth    int    `json:"depth"`
		LoadAll  bool   `json:"load_all"`
	}
	if e := strict(raw, &a, map[string]bool{"memory_id": true, "depth": true, "load_all": true}); e != nil {
		return nil, e
	}
	ms, e := s.resolveFrom(ctx, a.MemoryID, a.Depth, a.LoadAll)
	if e != nil {
		return nil, e
	}
	total := 0
	for _, m := range ms {
		total += len(m.Content)
	}
	return map[string]any{"resolved_memories": ms, "chain_length": len(ms), "total_size": total}, nil
}

func (s *memoryStore) resolveFrom(ctx context.Context, id string, depth int, full bool) ([]memoryData, error) {
	if depth == 0 {
		depth = 3
	}
	if depth < 1 || depth > s.maxDepth {
		return nil, fmt.Errorf("depth must be between 1 and %d", s.maxDepth)
	}
	seen := map[string]bool{}
	out := []memoryData{}
	var visit func(string, int) error
	visit = func(x string, d int) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if seen[x] {
			return fmt.Errorf("circular memory reference detected at %s", x)
		}
		m, e := s.read(x, full)
		if e != nil {
			return nil
		}
		seen[x] = true
		out = append(out, m)
		if d > 0 {
			for _, r := range m.Metadata.References {
				if e = visit(r, d-1); e != nil {
					return e
				}
			}
		}
		delete(seen, x)
		return nil
	}
	return out, visit(id, depth)
}

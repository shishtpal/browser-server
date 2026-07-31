package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *memoryStore) lazy(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		MemoryID     string `json:"memory_id"`
		Trigger      string `json:"trigger"`
		ExpiresAfter string `json:"expires_after"`
	}
	if e := strict(raw, &a, map[string]bool{"memory_id": true, "trigger": true, "expires_after": true}); e != nil {
		return nil, e
	}
	if a.Trigger == "" {
		a.Trigger = "access"
	}
	if a.Trigger != "access" && a.Trigger != "search" && a.Trigger != "time" {
		return nil, fmt.Errorf("invalid trigger")
	}
	m, e := s.read(a.MemoryID, true)
	if e != nil {
		return nil, e
	}
	m.Metadata.Lazy = true
	m.Metadata.LazyTrigger = a.Trigger
	if a.ExpiresAfter != "" {
		d, e := time.ParseDuration(a.ExpiresAfter)
		if e != nil || d <= 0 {
			return nil, fmt.Errorf("invalid expires_after")
		}
		t := time.Now().UTC().Add(d)
		m.Metadata.LazyExpiresAt = &t
	}
	if e = s.write(m); e != nil {
		return nil, e
	}
	return map[string]any{"memory_id": a.MemoryID, "trigger": a.Trigger, "expires_at": m.Metadata.LazyExpiresAt, "status": "pending"}, nil
}

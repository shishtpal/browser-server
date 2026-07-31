package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *memoryStore) remember(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Content        string   `json:"content"`
		Title          string   `json:"title"`
		Type           string   `json:"type"`
		TargetID       string   `json:"target_id"`
		Relationship   string   `json:"relationship"`
		Category       string   `json:"category"`
		Importance     string   `json:"importance"`
		References     []string `json:"references"`
		Tags           []string `json:"tags"`
		AutoCreateRefs bool     `json:"auto_create_refs"`
	}
	if e := strict(raw, &a, map[string]bool{"content": true, "title": true, "type": true, "target_id": true, "relationship": true, "references": true, "tags": true, "category": true, "importance": true, "auto_create_refs": true}); e != nil {
		return nil, e
	}
	if strings.TrimSpace(a.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	if a.Type == "" {
		a.Type = "primary"
	}
	if a.Type != "primary" && a.Type != "reference" {
		return nil, fmt.Errorf("type must be primary or reference")
	}
	if a.Type == "reference" {
		if validID(a.TargetID) != nil {
			return nil, fmt.Errorf("valid target_id is required for reference memories")
		}
		if _, e := s.pathFor(a.TargetID); e != nil {
			return nil, fmt.Errorf("target memory does not exist")
		}
	}
	for _, id := range a.References {
		if e := validID(id); e != nil {
			return nil, e
		}
		if _, e := s.pathFor(id); e != nil {
			if !a.AutoCreateRefs {
				return nil, fmt.Errorf("referenced memory %s does not exist", id)
			}
			stub := memoryData{Metadata: memoryMeta{ID: id, Timestamp: time.Now().UTC(), Title: "Auto-created reference", Type: "reference", Source: "ai"}, Content: "Reference placeholder."}
			if e := s.write(stub); e != nil {
				return nil, e
			}
		}
	}
	id, e := newMemoryID()
	if e != nil {
		return nil, e
	}
	now := time.Now().UTC()
	m := memoryData{Metadata: memoryMeta{ID: id, Timestamp: now, Title: a.Title, Type: a.Type, TargetID: a.TargetID, Relationship: a.Relationship, References: a.References, Tags: a.Tags, Category: a.Category, Importance: a.Importance, Source: "ai"}, Content: a.Content}
	if e = s.write(m); e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "timestamp": now, "type": a.Type}, nil
}

func (s *memoryStore) update(_ context.Context, raw json.RawMessage) (any, error) {
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	var a struct {
		ID, Content, Title, Category, Importance string
		References, Tags                         []string
	}
	if e := strict(raw, &a, map[string]bool{"id": true, "content": true, "title": true, "references": true, "tags": true, "category": true, "importance": true}); e != nil {
		return nil, e
	}
	m, e := s.read(a.ID, true)
	if e != nil {
		return nil, e
	}
	if _, ok := fields["content"]; ok {
		m.Content = a.Content
	}
	if _, ok := fields["title"]; ok {
		m.Metadata.Title = a.Title
	}
	if _, ok := fields["references"]; ok {
		for _, id := range a.References {
			if validID(id) != nil {
				return nil, fmt.Errorf("invalid reference id")
			}
			if id == a.ID {
				return nil, fmt.Errorf("memory cannot reference itself")
			}
		}
		m.Metadata.References = a.References
	}
	if _, ok := fields["tags"]; ok {
		m.Metadata.Tags = a.Tags
	}
	if _, ok := fields["category"]; ok {
		m.Metadata.Category = a.Category
	}
	if _, ok := fields["importance"]; ok {
		m.Metadata.Importance = a.Importance
	}
	m.Metadata.UpdatedAt = time.Now().UTC()
	if e = s.write(m); e != nil {
		return nil, e
	}
	return map[string]any{"id": a.ID, "updated": true}, nil
}

func (s *memoryStore) forget(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		ID string `json:"id"`
	}
	if e := strict(raw, &a, map[string]bool{"id": true}); e != nil {
		return nil, e
	}
	p, e := s.pathFor(a.ID)
	if e != nil {
		return nil, e
	}
	if e = os.Remove(p); e != nil {
		return nil, e
	}
	_ = os.Remove(filepath.Join(s.cache, a.ID+".md"))
	_ = s.walk(ctx, func(_ string, m memoryData) error {
		n := m.Metadata.References[:0]
		for _, id := range m.Metadata.References {
			if id != a.ID {
				n = append(n, id)
			}
		}
		if len(n) != len(m.Metadata.References) {
			m.Metadata.References = n
			return s.write(m)
		}
		return nil
	})
	return map[string]any{"id": a.ID, "deleted": true}, nil
}

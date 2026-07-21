package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"browser-server/internal/ai/config"
)

var memoryIDPattern = regexp.MustCompile(`^memory_[a-z0-9_-]{1,80}$`)

type memoryMeta struct {
	ID            string     `json:"id"`
	Timestamp     time.Time  `json:"timestamp"`
	UpdatedAt     time.Time  `json:"updated_at,omitempty"`
	Title         string     `json:"title,omitempty"`
	Type          string     `json:"type"`
	TargetID      string     `json:"target_id,omitempty"`
	Relationship  string     `json:"relationship,omitempty"`
	References    []string   `json:"references,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Category      string     `json:"category,omitempty"`
	Importance    string     `json:"importance,omitempty"`
	Source        string     `json:"source"`
	Lazy          bool       `json:"lazy,omitempty"`
	LazyTrigger   string     `json:"lazy_trigger,omitempty"`
	LazyExpiresAt *time.Time `json:"lazy_expires_at,omitempty"`
}
type memoryData struct {
	Metadata memoryMeta `json:"metadata"`
	Content  string     `json:"content,omitempty"`
}
type memoryStore struct {
	root, primary, refs, cache string
	maxFile                    int64
	maxDepth                   int
	cacheLimit                 int64
}

func newMemoryStore(c config.MemoryConfig) *memoryStore {
	if c.Directory == "" {
		c.Directory = ".memory"
	}
	if c.PrimaryDir == "" {
		c.PrimaryDir = "memories"
	}
	if c.RefsDir == "" {
		c.RefsDir = "refs"
	}
	if c.CacheDir == "" {
		c.CacheDir = "cache"
	}
	if c.MaxFileSizeKB == 0 {
		c.MaxFileSizeKB = 1024
	}
	if c.MaxReferenceDepth == 0 {
		c.MaxReferenceDepth = 5
	}
	if c.CacheSizeLimitMB == 0 {
		c.CacheSizeLimitMB = 100
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	root := filepath.Join(filepath.Dir(exe), c.Directory)
	return &memoryStore{root: root, primary: filepath.Join(root, c.PrimaryDir), refs: filepath.Join(root, c.RefsDir), cache: filepath.Join(root, c.CacheDir), maxFile: int64(c.MaxFileSizeKB) * 1024, maxDepth: c.MaxReferenceDepth, cacheLimit: int64(c.CacheSizeLimitMB) * 1024 * 1024}
}

func registerMemoryTools(r *Registry, s *memoryStore) {
	add := func(name, desc, schema string, fn func(context.Context, json.RawMessage) (any, error)) {
		r.add(Tool{Name: name, Category: "Memory", Description: desc, Schema: json.RawMessage(schema), Execute: fn})
	}
	add("ai_remember", "Store persistent markdown memory with JSON-compatible YAML frontmatter", `{"type":"object","properties":{"content":{"type":"string"},"title":{"type":"string"},"type":{"type":"string","enum":["primary","reference"]},"target_id":{"type":"string"},"relationship":{"type":"string"},"references":{"type":"array","items":{"type":"string"}},"tags":{"type":"array","items":{"type":"string"}},"category":{"type":"string"},"importance":{"type":"string"},"auto_create_refs":{"type":"boolean"}},"required":["content"],"additionalProperties":false}`, s.remember)
	add("ai_recall", "Recall memory by ID or text search", `{"type":"object","properties":{"id":{"type":"string"},"search":{"type":"string"},"include_references":{"type":"boolean"},"max_depth":{"type":"integer","minimum":1,"maximum":20},"load_lazy":{"type":"boolean"},"limit":{"type":"integer","minimum":1,"maximum":100}},"additionalProperties":false}`, s.recall)
	add("ai_search_memory", "Search memory content and metadata", `{"type":"object","properties":{"query":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}},"category":{"type":"string"},"importance":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":100},"metadata_only":{"type":"boolean"}},"additionalProperties":false}`, s.search)
	add("ai_list_memories", "List memory metadata", `{"type":"object","properties":{"type":{"type":"string"},"tag":{"type":"string"},"category":{"type":"string"},"importance":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":100}},"additionalProperties":false}`, s.list)
	add("ai_forget", "Delete a memory and remove references to it", `{"type":"object","properties":{"id":{"type":"string"}},"required":["id"],"additionalProperties":false}`, s.forget)
	add("ai_update_memory", "Update memory content or metadata atomically", `{"type":"object","properties":{"id":{"type":"string"},"content":{"type":"string"},"title":{"type":"string"},"references":{"type":"array","items":{"type":"string"}},"tags":{"type":"array","items":{"type":"string"}},"category":{"type":"string"},"importance":{"type":"string"}},"required":["id"],"additionalProperties":false}`, s.update)
	add("ai_resolve_references", "Resolve a memory reference chain with cycle protection", `{"type":"object","properties":{"memory_id":{"type":"string"},"depth":{"type":"integer","minimum":1,"maximum":20},"load_all":{"type":"boolean"}},"required":["memory_id"],"additionalProperties":false}`, s.resolve)
	add("ai_lazy_memory", "Attach lazy-loading metadata to a memory", `{"type":"object","properties":{"memory_id":{"type":"string"},"trigger":{"type":"string","enum":["access","search","time"]},"expires_after":{"type":"string"}},"required":["memory_id"],"additionalProperties":false}`, s.lazy)
	add("ai_manage_cache", "Inspect, clean, or optimize the memory cache", `{"type":"object","properties":{"action":{"type":"string","enum":["cleanup","stats","optimize"]},"max_age":{"type":"string"},"min_size":{"type":"integer"},"max_size":{"type":"integer"}},"required":["action"],"additionalProperties":false}`, s.manageCache)
}

func validID(id string) error {
	if !memoryIDPattern.MatchString(id) {
		return fmt.Errorf("invalid memory id")
	}
	return nil
}
func newMemoryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "memory_" + hex.EncodeToString(b), nil
}
func (s *memoryStore) ensure() error {
	for _, d := range []string{s.primary, s.refs, s.cache} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
func encodeMemory(m memoryData) ([]byte, error) {
	h, e := json.MarshalIndent(m.Metadata, "", "  ")
	if e != nil {
		return nil, e
	}
	return []byte("---\n" + string(h) + "\n---\n" + m.Content), nil
}
func decodeMemory(b []byte) (memoryData, error) {
	var m memoryData
	p := strings.SplitN(string(b), "\n---\n", 2)
	if len(p) != 2 || !strings.HasPrefix(p[0], "---\n") {
		return m, fmt.Errorf("invalid memory frontmatter")
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(p[0], "---\n")), &m.Metadata); err != nil {
		return m, err
	}
	m.Content = p[1]
	return m, nil
}
func atomicWrite(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".memory-*")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(n, path)
}
func (s *memoryStore) pathFor(id string) (string, error) {
	if err := validID(id); err != nil {
		return "", err
	}
	for _, d := range []string{s.primary, s.refs} {
		p := filepath.Join(d, id+".md")
		if _, e := os.Stat(p); e == nil {
			return p, nil
		}
	}
	return "", fs.ErrNotExist
}
func (s *memoryStore) read(id string, content bool) (memoryData, error) {
	p, e := s.pathFor(id)
	if e != nil {
		return memoryData{}, e
	}
	st, e := os.Stat(p)
	if e != nil {
		return memoryData{}, e
	}
	if st.Size() > s.maxFile {
		return memoryData{}, fmt.Errorf("memory exceeds file size limit")
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return memoryData{}, e
	}
	m, e := decodeMemory(b)
	if e == nil && content {
		_ = atomicWrite(filepath.Join(s.cache, id+".md"), b)
	}
	if e == nil && !content {
		m.Content = ""
	}
	return m, e
}
func (s *memoryStore) write(m memoryData) error {
	b, e := encodeMemory(m)
	if e != nil {
		return e
	}
	if int64(len(b)) > s.maxFile {
		return fmt.Errorf("memory exceeds file size limit")
	}
	d := s.primary
	if m.Metadata.Type == "reference" {
		d = s.refs
	}
	return atomicWrite(filepath.Join(d, m.Metadata.ID+".md"), b)
}
func (s *memoryStore) walk(ctx context.Context, fn func(string, memoryData) error) error {
	if e := s.ensure(); e != nil {
		return e
	}
	for _, d := range []string{s.primary, s.refs} {
		es, e := os.ReadDir(d)
		if e != nil {
			return e
		}
		for _, x := range es {
			if e := ctx.Err(); e != nil {
				return e
			}
			if x.IsDir() || filepath.Ext(x.Name()) != ".md" {
				continue
			}
			b, e := os.ReadFile(filepath.Join(d, x.Name()))
			if e != nil {
				return e
			}
			if int64(len(b)) > s.maxFile {
				continue
			}
			m, e := decodeMemory(b)
			if e != nil {
				continue
			}
			if e = fn(filepath.Join(d, x.Name()), m); e != nil {
				return e
			}
		}
	}
	return nil
}

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
func (s *memoryStore) recall(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		ID                string `json:"id"`
		Search            string `json:"search"`
		IncludeReferences bool   `json:"include_references"`
		LoadLazy          bool   `json:"load_lazy"`
		MaxDepth          int    `json:"max_depth"`
		Limit             int    `json:"limit"`
	}
	if e := strict(raw, &a, map[string]bool{"id": true, "search": true, "include_references": true, "max_depth": true, "load_lazy": true, "limit": true}); e != nil {
		return nil, e
	}
	if a.Limit == 0 {
		a.Limit = 20
	}
	if a.Limit < 1 || a.Limit > 100 {
		return nil, fmt.Errorf("limit must be 1 to 100")
	}
	if a.ID != "" {
		m, e := s.read(a.ID, a.LoadLazy || !mLazy(s, a.ID))
		if e != nil {
			return nil, e
		}
		out := []memoryData{m}
		if a.IncludeReferences {
			rs, e := s.resolveFrom(ctx, a.ID, a.MaxDepth, true)
			if e != nil {
				return nil, e
			}
			out = rs
		}
		return map[string]any{"memories": out, "total_count": len(out)}, nil
	}
	if a.Search == "" {
		return nil, fmt.Errorf("either id or search is required")
	}
	return s.searchValues(ctx, a.Search, nil, "", "", a.Limit, !a.LoadLazy)
}
func mLazy(s *memoryStore, id string) bool {
	m, e := s.read(id, false)
	return e == nil && m.Metadata.Lazy
}
func (s *memoryStore) search(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Query        string   `json:"query"`
		Category     string   `json:"category"`
		Importance   string   `json:"importance"`
		Tags         []string `json:"tags"`
		Limit        int      `json:"limit"`
		MetadataOnly bool     `json:"metadata_only"`
	}
	if e := strict(raw, &a, map[string]bool{"query": true, "tags": true, "category": true, "importance": true, "limit": true, "metadata_only": true}); e != nil {
		return nil, e
	}
	if a.Limit == 0 {
		a.Limit = 20
	}
	return s.searchValues(ctx, a.Query, a.Tags, a.Category, a.Importance, a.Limit, a.MetadataOnly)
}
func (s *memoryStore) searchValues(ctx context.Context, q string, tags []string, cat, imp string, limit int, meta bool) (any, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be 1 to 100")
	}
	q = strings.ToLower(q)
	out := []memoryData{}
	e := s.walk(ctx, func(_ string, m memoryData) error {
		hay := strings.ToLower(m.Metadata.Title + " " + m.Content + " " + strings.Join(m.Metadata.Tags, " "))
		if q != "" && !strings.Contains(hay, q) {
			return nil
		}
		if cat != "" && m.Metadata.Category != cat {
			return nil
		}
		if imp != "" && m.Metadata.Importance != imp {
			return nil
		}
		for _, t := range tags {
			found := false
			for _, mt := range m.Metadata.Tags {
				if mt == t {
					found = true
				}
			}
			if !found {
				return nil
			}
		}
		if meta || m.Metadata.Lazy {
			m.Content = ""
		}
		out = append(out, m)
		if len(out) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	if errors.Is(e, fs.SkipAll) {
		e = nil
	}
	return map[string]any{"memories": out, "total_count": len(out)}, e
}
func (s *memoryStore) list(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Type, Tag, Category, Importance string
		Limit                           int
	}
	if e := strict(raw, &a, map[string]bool{"type": true, "tag": true, "category": true, "importance": true, "limit": true}); e != nil {
		return nil, e
	}
	tags := []string{}
	if a.Tag != "" {
		tags = []string{a.Tag}
	}
	return s.searchValues(ctx, "", tags, a.Category, a.Importance, defaultLimit(a.Limit), true)
}
func defaultLimit(v int) int {
	if v == 0 {
		return 100
	}
	return v
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

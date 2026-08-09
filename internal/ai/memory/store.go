package memory

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"browser-server/internal/ai/config"
)

// ChatCompleter abstracts the "librarian" sub-agent LLM call used by
// recall_memory synthesize=true. It is wired in from bootstrap so the memory
// package stays decoupled from the provider implementation.
type ChatCompleter interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// CompleterFunc adapts a plain function to ChatCompleter.
type CompleterFunc func(context.Context, CompletionRequest) (CompletionResponse, error)

func (f CompleterFunc) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	return f(ctx, req)
}

type CompletionRequest struct {
	System          string
	User            string
	Temperature     float64
	MaxOutputTokens int
}

type CompletionResponse struct {
	Content string
}

// Store is the process-wide memory graph store. It is a singleton keyed by its
// resolved root path so tools, the chat persona injector and the admin
// endpoint all share one instance and one lock.
type Store struct {
	enabled bool
	root    string
	frags   string // fragments dir
	archive string // .archive dir
	tmp     string // .tmp dir
	idxPath string // index.json path
	cfg     config.MemoryConfig

	idx     *Index
	mu      sync.RWMutex
	writeMu sync.Mutex
	dirty   bool

	completer ChatCompleter
}

var (
	singletonMu sync.Mutex
	singletons  = map[string]*Store{}
)

// ResolveRoot resolves the memory root from config. Relative paths are
// anchored to the executable directory (portable app layout, matching v1).
func ResolveRoot(cfg config.MemoryConfig) string {
	dir := cfg.Directory
	if dir == "" {
		dir = ".memory"
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	return filepath.Join(filepath.Dir(exe), dir)
}

// New returns the process singleton for the given config, bootstrapping it on
// first use. Repeated calls with the same resolved root return the same Store.
func New(cfg config.MemoryConfig) *Store {
	root := ResolveRoot(cfg)
	singletonMu.Lock()
	defer singletonMu.Unlock()
	if s, ok := singletons[root]; ok {
		return s
	}
	s := openStore(root, cfg)
	if s.enabled {
		if err := s.bootstrap(); err != nil {
			log.Printf("memory: bootstrap failed at %s: %v", root, err)
			s.enabled = false
		}
	}
	singletons[root] = s
	return s
}

// Open builds an isolated store rooted at an explicit path (used by tests and
// tooling). It does not consult the singleton map.
func Open(root string, cfg config.MemoryConfig) *Store {
	s := openStore(root, cfg)
	if s.enabled {
		if err := s.bootstrap(); err != nil {
			s.enabled = false
		}
	}
	return s
}

func openStore(root string, cfg config.MemoryConfig) *Store {
	cfg = normalizeCfg(cfg)
	s := &Store{
		enabled: cfg.Enabled,
		root:    root,
		frags:   filepath.Join(root, cfg.FragmentsDir),
		archive: filepath.Join(root, cfg.ArchiveDir),
		tmp:     filepath.Join(root, ".tmp"),
		idxPath: filepath.Join(root, "index.json"),
		cfg:     cfg,
		idx:     newIndex(),
	}
	return s
}

func normalizeCfg(cfg config.MemoryConfig) config.MemoryConfig {
	if cfg.Directory == "" {
		cfg.Directory = ".memory"
	}
	if cfg.FragmentsDir == "" {
		cfg.FragmentsDir = "fragments"
	}
	if cfg.ArchiveDir == "" {
		cfg.ArchiveDir = ".archive"
	}
	if cfg.MaxBodyKB == 0 {
		cfg.MaxBodyKB = 64
	}
	if cfg.MaxLinksPerFragment == 0 {
		cfg.MaxLinksPerFragment = 64
	}
	if cfg.MaxOpsPerCall == 0 {
		cfg.MaxOpsPerCall = 20
	}
	if cfg.MaxResultBytes == 0 {
		cfg.MaxResultBytes = 8192
	}
	if cfg.DefaultDepth == 0 {
		cfg.DefaultDepth = 1
	}
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 3
	}
	if cfg.SpreadFactor == 0 {
		cfg.SpreadFactor = 0.45
	}
	if cfg.PersonaTokenBudget == 0 {
		cfg.PersonaTokenBudget = 900
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 365
	}
	if cfg.SalienceDecayPerWeek == 0 {
		cfg.SalienceDecayPerWeek = 0.985
	}
	if cfg.ArchiveThreshold == 0 {
		cfg.ArchiveThreshold = 0.15
	}
	if cfg.NearDuplicateThreshold == 0 {
		cfg.NearDuplicateThreshold = 0.82
	}
	if cfg.Synthesizer.MaxOutputTokens == 0 {
		cfg.Synthesizer.MaxOutputTokens = 512
	}
	return cfg
}

// SetCompleter wires the librarian sub-agent used by recall synthesize=true.
func (s *Store) SetCompleter(c ChatCompleter) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.completer = c
}

// Enabled reports whether the store is active.
func (s *Store) Enabled() bool { return s.enabled }

// Root returns the resolved memory root directory.
func (s *Store) Root() string { return s.root }

func (s *Store) ensure() error {
	for _, d := range []string{s.frags, s.archive, s.tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) bootstrap() error {
	if err := s.ensure(); err != nil {
		return err
	}
	// v1 -> v2 migration runs once when a v1 memories/ dir exists and no
	// fragments dir exists yet.
	if err := s.migrateIfNeeded(); err != nil {
		return err
	}
	if err := s.load(); err != nil {
		return err
	}
	if err := s.ensureRoot(); err != nil {
		return err
	}
	s.flushSnapshot()
	return nil
}

// load builds the index from disk, restoring the JSON snapshot when it is
// fresh (matching directory fingerprint) and otherwise scanning the fragments
// directory.
func (s *Store) load() error {
	fp, err := s.fingerprint()
	if err == nil {
		if b, e := os.ReadFile(s.idxPath); e == nil {
			var meta struct {
				Fingerprint string `json:"fingerprint"`
				Index       []byte `json:"index"`
			}
			if json.Unmarshal(b, &meta) == nil && meta.Fingerprint == fp {
				ix := newIndex()
				if ix.restore(meta.Index) == nil {
					s.idx = ix
					return nil
				}
			}
		}
	}
	// Rebuild from disk.
	ix := newIndex()
	entries, err := os.ReadDir(s.frags)
	if err != nil {
		return err
	}
	for _, en := range entries {
		if en.IsDir() || filepath.Ext(en.Name()) != ".md" {
			continue
		}
		f, e := s.readFile(filepath.Join(s.frags, en.Name()))
		if e != nil {
			continue
		}
		// ix.add indexes body terms but stores metadata only (bodies stay lazy).
		ix.add(f)
	}
	s.mu.Lock()
	s.idx = ix
	s.mu.Unlock()
	return nil
}

// fingerprint hashes the directory's file names+mtime so the snapshot can be
// skipped when nothing changed on disk (e.g. edits made by hand or git).
func (s *Store) fingerprint() (string, error) {
	entries, err := os.ReadDir(s.frags)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, en := range entries {
		info, e := en.Info()
		if e != nil {
			continue
		}
		fmt.Fprintf(h, "%s|%d|", en.Name(), info.ModTime().UnixNano())
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (s *Store) readFile(path string) (*Fragment, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > int64(s.cfg.MaxBodyKB)*1024+16384 {
		return nil, fmt.Errorf("fragment exceeds size limit")
	}
	return decodeFragment(b)
}

func (s *Store) pathFor(id string) string {
	return filepath.Join(s.frags, id+".md")
}

func (s *Store) archivePathFor(id string) string {
	return filepath.Join(s.archive, id+".md")
}

// flushSnapshot writes index.json (derived data; safe to rebuild).
func (s *Store) flushSnapshot() {
	s.mu.RLock()
	b, err := s.idx.snapshot()
	s.mu.RUnlock()
	if err != nil {
		return
	}
	fp, _ := s.fingerprint()
	meta, err := json.Marshal(map[string]any{"fingerprint": fp, "index": json.RawMessage(b)})
	if err != nil {
		return
	}
	_ = atomicWrite(s.idxPath, meta)
}

func (s *Store) markDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

// touchAccess records one read of id: increments AccessCount, sets Accessed,
// and persists the fragment when its access metadata has not been flushed
// recently (throttled so hot reads do not rewrite the file on every call).
// Returns false when the fragment does not exist.
func (s *Store) touchAccess(id string) bool {
	now := time.Now().UTC()
	s.mu.Lock()
	f, ok := s.idx.byID[id]
	if !ok {
		s.mu.Unlock()
		return false
	}
	f.AccessCount++
	persist := now.Sub(f.Accessed) > 10*time.Minute
	f.Accessed = now
	var cp *Fragment
	if persist {
		cp = cloneFragment(f)
	}
	s.mu.Unlock()
	if cp != nil {
		// The index stores metadata only; reattach the on-disk body so the
		// rewrite does not truncate it.
		if body, err := s.readBody(id); err == nil {
			cp.Body = body
		}
		if b, err := encodeFragment(cp); err == nil {
			_ = writeTempThenRename(s.pathFor(id), b)
		}
	}
	return true
}

// Close flushes any pending snapshot.
func (s *Store) Close() error {
	s.flushSnapshot()
	return nil
}

// Stats returns a small summary used by the admin endpoint.
type Stats struct {
	Enabled   bool   `json:"enabled"`
	Fragments int    `json:"fragments"`
	Root      string `json:"root"`
	IndexFile string `json:"index_file"`
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	n := len(s.idx.byID)
	s.mu.RUnlock()
	return Stats{Enabled: s.enabled, Fragments: n, Root: s.root, IndexFile: s.idxPath}
}

// ---------------------------------------------------------------------------
// Persona bootstrap
// ---------------------------------------------------------------------------

// ensureRoot writes mem_root and its five undeletable children if absent.
func (s *Store) ensureRoot() error {
	now := time.Now().UTC()
	type def struct {
		id      string
		kind    Kind
		title   string
		summary string
		parent  string
	}
	defs := []def{
		{"mem_root", KindPersona, "Persona Root", "Identity anchor: who I am, who the user is, and what we work on.", ""},
		{"mem_self", KindPersona, "Agent Identity", "My identity, capabilities and operating rules.", "mem_root"},
		{"mem_user", KindPersona, "User", "The user: identity, environment, working style and preferences.", "mem_root"},
		{"mem_projects", KindIndex, "Projects", "Index of active projects.", "mem_root"},
		{"mem_glossary", KindIndex, "Glossary", "Shared vocabulary used across projects.", "mem_root"},
		{"mem_inbox", KindIndex, "Inbox", "Unfiled fragments awaiting triage.", "mem_root"},
	}
	for _, d := range defs {
		s.mu.RLock()
		_, exists := s.idx.byID[d.id]
		s.mu.RUnlock()
		if exists {
			continue
		}
		f := &Fragment{
			ID: d.id, Kind: d.kind, Title: d.title, Summary: d.summary,
			Status: StatusActive, Pinned: true, Salience: 1.0,
			Created: now, Updated: now, Source: "system",
		}
		if d.parent != "" {
			f.Links = append(f.Links, Link{Rel: RelChildOf, To: d.parent, Weight: 1.0})
		}
		b, err := encodeFragment(f)
		if err != nil {
			return err
		}
		if err := atomicWrite(s.pathFor(d.id), b); err != nil {
			return err
		}
		s.mu.Lock()
		s.idx.add(f)
		s.mu.Unlock()
	}
	return nil
}

// PersonaBlock returns the auto-injected <memory:persona> + <memory:usage>
// context block, budgeted in approximate tokens. It is appended to the system
// prompt on conversation start so identity, profile and project index survive
// a profile switch.
func (s *Store) PersonaBlock(ctx context.Context, budget int, withUsage bool) string {
	if !s.enabled {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	children := s.idx.childIDs("mem_root")
	sort.Strings(children)

	get := func(id string) *Fragment {
		if f, ok := s.idx.byID[id]; ok {
			cp := *f
			return &cp
		}
		return nil
	}

	var b strings.Builder
	// Summaries are never dropped.
	writeSummary := func(f *Fragment) {
		if f == nil {
			return
		}
		fmt.Fprintf(&b, "# %s\n%s\n", f.Title, f.Summary)
	}

	self := get("mem_self")
	user := get("mem_user")
	writeSummary(self)
	if self != nil && self.Body != "" {
		fmt.Fprintf(&b, "%s\n", self.Body)
	}
	writeSummary(user)

	// Projects index: count descendants per project child.
	b.WriteString("\n# Active projects\n")
	for _, id := range children {
		if id == "mem_self" || id == "mem_user" || id == "mem_projects" || id == "mem_glossary" || id == "mem_inbox" {
			continue
		}
		f := get(id)
		if f == nil {
			continue
		}
		n := len(s.idx.childIDs(id))
		fmt.Fprintf(&b, "- %s — %s (%d fragments)\n", id, f.Summary, n)
	}

	// Pinned non-persona fragments.
	b.WriteString("\n# Pinned\n")
	used := 0
	for _, f := range s.idx.byID {
		if !f.Pinned || f.ID == "mem_root" || f.ID == "mem_self" || f.ID == "mem_user" {
			continue
		}
		fmt.Fprintf(&b, "- %s — %s\n", f.ID, f.Summary)
		used++
		if used > 12 {
			break
		}
	}

	if withUsage {
		b.WriteString(usageGuide)
	}
	return strings.TrimSpace(b.String())
}

const usageGuide = `
<memory:usage>
Two tools: recall_memory (read/search/traverse) and write_memory (batched mutations).
Graph shape: every fragment has one parent (child_of) rooted at mem_root, plus typed cross-links.
- Before answering anything about the user, projects, decisions or history, call recall_memory.
- To save a memory, first recall_memory with the intended title as the query. If a fragment
  already covers it, upsert that id or append to it. Never create a near-duplicate.
- Use stable slug ids (e.g. mem_proj_x_decisions), not random ones.
- Always set a parent; if unsure, omit it (goes to mem_inbox) and file it later.
- Never store secrets, tokens, passwords or API keys.
</memory:usage>`

package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// WriteArgs is the write_memory tool argument set: a validated, atomic batch.
type WriteArgs struct {
	Ops []WriteOp `json:"ops"`
}

// WriteOp is one mutation in a write_memory batch.
type WriteOp struct {
	Op         string   `json:"op"`
	ID         string   `json:"id"`
	Kind       Kind     `json:"kind"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	Tags       []string `json:"tags"`
	Parent     string   `json:"parent"`
	Pinned     *bool    `json:"pinned"`
	Confidence *float32 `json:"confidence"`
	// Status overrides the fragment status on upsert/archive when non-empty.
	Status       Status      `json:"status"`
	Links        []WriteLink `json:"links"`
	From         string      `json:"from"`
	Rel          Rel         `json:"rel"`
	To           string      `json:"to"`
	OnConflict   string      `json:"on_conflict"`
	Note         string      `json:"note"`
	SupersededBy string      `json:"superseded_by"`
	Cascade      bool        `json:"cascade"`
}

// WriteLink is an edge to add in an upsert.
type WriteLink struct {
	Rel  Rel    `json:"rel"`
	To   string `json:"to"`
	Note string `json:"note"`
}

// WriteOpResult describes one applied op.
type WriteOpResult struct {
	Op           string   `json:"op"`
	ID           string   `json:"id"`
	Created      bool     `json:"created"`
	MergedFields []string `json:"merged_fields,omitempty"`
	DuplicateOf  string   `json:"duplicate_of,omitempty"`
	Warning      string   `json:"warning,omitempty"`
}

// WriteResult is the response envelope for write_memory.
type WriteResult struct {
	Applied  int             `json:"applied"`
	Results  []WriteOpResult `json:"results"`
	Warnings []string        `json:"warnings,omitempty"`
}

var validOps = map[string]bool{
	"upsert": true, "append": true, "link": true, "unlink": true,
	"move": true, "archive": true, "delete": true,
}

// Write implements write_memory: validate-all-then-apply atomically.
func (s *Store) Write(ctx context.Context, a WriteArgs) (WriteResult, error) {
	if !s.enabled {
		return WriteResult{}, fmt.Errorf("memory disabled")
	}
	if len(a.Ops) == 0 {
		return WriteResult{}, fmt.Errorf("at least one op is required")
	}
	if len(a.Ops) > s.cfg.MaxOpsPerCall {
		return WriteResult{}, fmt.Errorf("batch exceeds max_ops_per_call (%d)", s.cfg.MaxOpsPerCall)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// Working view with copy-on-write so the live index is untouched until all
	// ops validate. Within-batch references resolve against this view.
	view := map[string]*Fragment{}
	// dirty tracks ids whose file must be (re)written; deleted tracks ids to
	// move to .archive.
	dirty := map[string]bool{}
	deleted := map[string]bool{}
	var warnings []string
	var results []WriteOpResult

	viewGet := func(id string) (*Fragment, bool) {
		if f, ok := view[id]; ok {
			return f, true
		}
		if f, ok := s.idx.byID[id]; ok {
			c := cloneFragment(f)
			view[id] = c
			return c, true
		}
		return nil, false
	}

	for _, op := range a.Ops {
		if err := ctx.Err(); err != nil {
			return WriteResult{}, err
		}
		res, err := s.applyOp(view, viewGet, dirty, deleted, op)
		if err != nil {
			// Validation/planning failed: nothing applied.
			return WriteResult{}, err
		}
		if res.Warning != "" {
			warnings = append(warnings, res.Warning)
		}
		results = append(results, res)
	}

	// Commit phase: write files (staged, then renamed), move deletions to
	// .archive, then update the live index in one lock.
	if err := s.commit(view, dirty, deleted); err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Applied: len(results), Results: results, Warnings: warnings}, nil
}

// applyOp validates and plans one op against the working view. On error,
// nothing has been applied and the caller aborts the whole batch.
func (s *Store) applyOp(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty, deleted map[string]bool, op WriteOp) (WriteOpResult, error) {
	if !validOps[op.Op] {
		return WriteOpResult{}, fmt.Errorf("unknown op %q", op.Op)
	}
	switch op.Op {
	case "upsert":
		return s.opUpsert(view, viewGet, dirty, op)
	case "append":
		return s.opAppend(view, viewGet, dirty, op)
	case "link":
		return s.opLink(view, viewGet, dirty, op)
	case "unlink":
		return s.opUnlink(view, viewGet, dirty, op)
	case "move":
		return s.opMove(view, viewGet, dirty, op)
	case "archive":
		return s.opArchive(view, viewGet, dirty, op)
	case "delete":
		return s.opDelete(view, viewGet, dirty, deleted, op)
	}
	return WriteOpResult{}, fmt.Errorf("unknown op %q", op.Op)
}

func (s *Store) opUpsert(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	id := op.ID
	if id == "" {
		if op.Title == "" {
			return WriteOpResult{}, fmt.Errorf("upsert requires id or title")
		}
		id = slugify(op.Title)
		if !validID(id) {
			return WriteOpResult{}, fmt.Errorf("invalid generated id %q", id)
		}
	}
	onConflict := op.OnConflict
	if onConflict == "" {
		onConflict = "merge"
	}
	if onConflict != "merge" && onConflict != "new" && onConflict != "error" {
		return WriteOpResult{}, fmt.Errorf("invalid on_conflict %q", onConflict)
	}

	existing, exists := viewGet(id)

	// Near-dup detection for brand-new fragments.
	if !exists {
		dup, dupID := s.findNearDuplicate(op.Title, op.Summary)
		if dup != nil && onConflict == "merge" {
			// clone so we never mutate the live index fragment in place
			existing = cloneFragment(dup)
			id = dupID
			exists = true
		}
	}
	if !exists {
		if op.Title == "" {
			return WriteOpResult{}, fmt.Errorf("upsert requires title for new fragment")
		}
		now := time.Now().UTC()
		f := &Fragment{
			ID: id, Kind: op.Kind, Title: op.Title, Summary: truncateStr(op.Summary, 280),
			Body: op.Body, Tags: dedupeStrings(append([]string(nil), op.Tags...)), Status: StatusActive,
			Confidence: floatOf(op.Confidence), Salience: 1.0,
			Created: now, Updated: now, Source: "ai",
		}
		if f.Kind == "" {
			f.Kind = KindNote
		}
		if !validateKind(f.Kind) {
			return WriteOpResult{}, fmt.Errorf("invalid kind %q", f.Kind)
		}
		parent := op.Parent
		if parent == "" {
			parent = "mem_inbox"
		}
		if !validID(parent) {
			return WriteOpResult{}, fmt.Errorf("invalid parent %q", parent)
		}
		if s.wouldCreateCycle(id, parent) {
			return WriteOpResult{}, fmt.Errorf("re-parenting %s under %s would create a cycle", id, parent)
		}
		f.Links = append(f.Links, Link{Rel: RelChildOf, To: parent, Weight: 1.0})
		for _, l := range op.Links {
			if err := s.addPlannedLink(view, viewGet, f, l); err != nil {
				return WriteOpResult{}, err
			}
		}
		if err := s.scan(f); err != nil {
			return WriteOpResult{}, err
		}
		view[id] = f
		dirty[id] = true
		return WriteOpResult{Op: "upsert", ID: id, Created: true}, nil
	}

	// exists: merge non-null fields.
	if onConflict == "error" {
		return WriteOpResult{}, fmt.Errorf("fragment %s already exists", id)
	}
	if onConflict == "new" {
		// find a fresh suffix id
		base := id
		for i := 2; ; i++ {
			cand := fmt.Sprintf("%s_%d", base, i)
			if !validID(cand) {
				cand = base
			}
			if _, ok := viewGet(cand); !ok {
				id = cand
				break
			}
			if !validID(cand) {
				break
			}
		}
		existing = nil
		// create fresh below
		op.ID = id
		return s.opUpsert(view, viewGet, dirty, op)
	}

	// merge
	merged := []string{}
	now := time.Now().UTC()
	if op.Title != "" && op.Title != existing.Title {
		existing.Title = op.Title
		merged = append(merged, "title")
	}
	if op.Summary != "" {
		existing.Summary = truncateStr(op.Summary, 280)
		merged = append(merged, "summary")
	}
	if op.Body != "" {
		existing.Body = op.Body
		merged = append(merged, "body")
	}
	if len(op.Tags) > 0 {
		existing.Tags = dedupeStrings(append(append([]string(nil), existing.Tags...), op.Tags...))
		merged = append(merged, "tags")
	}
	if op.Kind != "" && validateKind(op.Kind) && existing.Kind != op.Kind {
		existing.Kind = op.Kind
		merged = append(merged, "kind")
	}
	if op.Pinned != nil && existing.Pinned != *op.Pinned {
		existing.Pinned = *op.Pinned
		merged = append(merged, "pinned")
	}
	if op.Confidence != nil {
		existing.Confidence = *op.Confidence
		merged = append(merged, "confidence")
	}
	if op.Status != "" && existing.Status != op.Status {
		existing.Status = op.Status
		merged = append(merged, "status")
	}
	if len(op.Links) > 0 {
		for _, l := range op.Links {
			if err := s.addPlannedLink(view, viewGet, existing, l); err != nil {
				return WriteOpResult{}, err
			}
		}
		merged = append(merged, "links")
	}
	if op.Parent != "" && parentOf(existing) != op.Parent {
		if err := s.movePlanned(view, existing, op.Parent); err != nil {
			return WriteOpResult{}, err
		}
		merged = append(merged, "parent")
	}
	if err := s.scan(existing); err != nil {
		return WriteOpResult{}, err
	}
	existing.Updated = now
	view[existing.ID] = existing
	dirty[existing.ID] = true
	res := WriteOpResult{Op: "upsert", ID: existing.ID, Created: false}
	if len(merged) > 0 {
		res.MergedFields = merged
	}
	return res, nil
}

func (s *Store) opAppend(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.ID == "" {
		return WriteOpResult{}, fmt.Errorf("append requires id")
	}
	f, ok := viewGet(op.ID)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.ID)
	}
	if op.Body == "" {
		return WriteOpResult{}, fmt.Errorf("append requires body")
	}
	if err := s.scan(f); err != nil {
		return WriteOpResult{}, err
	}
	now := time.Now().UTC()
	if f.Body != "" {
		f.Body += "\n\n"
	}
	f.Body += op.Body
	f.Updated = now
	view[f.ID] = f
	dirty[f.ID] = true
	return WriteOpResult{Op: "append", ID: f.ID}, nil
}

func (s *Store) opLink(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.From == "" || op.To == "" {
		return WriteOpResult{}, fmt.Errorf("link requires from and to")
	}
	if !validateRel(op.Rel) || op.Rel == RelChildOf {
		return WriteOpResult{}, fmt.Errorf("invalid link rel %q (child_of is set via parent/move)", op.Rel)
	}
	from, ok := viewGet(op.From)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.From)
	}
	to, ok := viewGet(op.To)
	if !ok {
		// auto-create a stub target under mem_inbox
		if !validID(op.To) {
			return WriteOpResult{}, fmt.Errorf("invalid link target %q", op.To)
		}
		now := time.Now().UTC()
		to = &Fragment{
			ID: op.To, Kind: KindStub, Title: "Stub", Summary: "Auto-created link target.",
			Status: StatusActive, Salience: 1.0, Created: now, Updated: now, Source: "ai",
			Links: []Link{{Rel: RelChildOf, To: "mem_inbox", Weight: 1.0}},
		}
		view[op.To] = to
		dirty[op.To] = true
	}
	if from.ID == to.ID {
		return WriteOpResult{}, fmt.Errorf("cannot link a fragment to itself")
	}
	addEdge(from, op.Rel, to.ID, op.Note)
	if symmetricRels[op.Rel] {
		addEdge(to, op.Rel, from.ID, op.Note)
	}
	dirty[from.ID] = true
	dirty[to.ID] = true
	view[from.ID] = from
	view[to.ID] = to
	return WriteOpResult{Op: "link", ID: from.ID}, nil
}

func (s *Store) opUnlink(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.From == "" || op.To == "" {
		return WriteOpResult{}, fmt.Errorf("unlink requires from and to")
	}
	if !validateRel(op.Rel) {
		return WriteOpResult{}, fmt.Errorf("invalid link rel %q", op.Rel)
	}
	from, ok := viewGet(op.From)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.From)
	}
	to, ok := viewGet(op.To)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.To)
	}
	from.Links = removeEdge(from.Links, op.Rel, op.To)
	to.Links = removeEdge(to.Links, op.Rel, op.From)
	dirty[from.ID] = true
	dirty[to.ID] = true
	view[from.ID] = from
	view[to.ID] = to
	return WriteOpResult{Op: "unlink", ID: from.ID}, nil
}

func (s *Store) opMove(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.ID == "" || op.Parent == "" {
		return WriteOpResult{}, fmt.Errorf("move requires id and parent")
	}
	if !validID(op.Parent) {
		return WriteOpResult{}, fmt.Errorf("invalid parent %q", op.Parent)
	}
	f, ok := viewGet(op.ID)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.ID)
	}
	if err := s.movePlanned(view, f, op.Parent); err != nil {
		return WriteOpResult{}, err
	}
	f.Updated = time.Now().UTC()
	view[f.ID] = f
	dirty[f.ID] = true
	return WriteOpResult{Op: "move", ID: f.ID}, nil
}

func (s *Store) opArchive(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.ID == "" {
		return WriteOpResult{}, fmt.Errorf("archive requires id")
	}
	f, ok := viewGet(op.ID)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.ID)
	}
	f.Status = StatusArchived
	if op.SupersededBy != "" {
		if !validID(op.SupersededBy) {
			return WriteOpResult{}, fmt.Errorf("invalid superseded_by")
		}
		addEdge(f, RelSupersedes, op.SupersededBy, "")
		if t, ok := viewGet(op.SupersededBy); ok {
			addEdge(t, "superseded_by", f.ID, "")
			view[t.ID] = t
			dirty[t.ID] = true
		}
	}
	f.Updated = time.Now().UTC()
	view[f.ID] = f
	dirty[f.ID] = true
	return WriteOpResult{Op: "archive", ID: f.ID}, nil
}

func (s *Store) opDelete(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), dirty, deleted map[string]bool, op WriteOp) (WriteOpResult, error) {
	if op.ID == "" {
		return WriteOpResult{}, fmt.Errorf("delete requires id")
	}
	if undeletableIDs[op.ID] {
		return WriteOpResult{}, fmt.Errorf("refusing to delete reserved fragment %s", op.ID)
	}
	_, ok := viewGet(op.ID)
	if !ok {
		return WriteOpResult{}, fmt.Errorf("fragment %s does not exist", op.ID)
	}
	if s.hasChildren(op.ID) && !op.Cascade {
		return WriteOpResult{}, fmt.Errorf("fragment %s has children; set cascade=true to delete them too", op.ID)
	}
	// Snapshot the reverse edges and children under a read lock, then scrub.
	s.mu.RLock()
	inbound := s.idx.inboundEdges(op.ID)
	children := s.idx.childIDs(op.ID)
	s.mu.RUnlock()
	for _, e := range inbound {
		if src, ok := viewGet(e.From); ok {
			src.Links = removeEdge(src.Links, e.Rel, op.ID)
			view[src.ID] = src
			dirty[src.ID] = true
		}
	}
	if op.Cascade {
		for _, c := range children {
			deleted[c] = true
		}
	}
	deleted[op.ID] = true
	delete(dirty, op.ID)
	return WriteOpResult{Op: "delete", ID: op.ID}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (s *Store) addPlannedLink(view map[string]*Fragment, viewGet func(string) (*Fragment, bool), f *Fragment, l WriteLink) error {
	if !validateRel(l.Rel) {
		return fmt.Errorf("invalid link rel %q", l.Rel)
	}
	if l.To == "" || !validID(l.To) {
		return fmt.Errorf("invalid link target %q", l.To)
	}
	// target must exist or a stub is auto-created
	if _, ok := viewGet(l.To); !ok {
		if l.Rel == RelChildOf {
			return fmt.Errorf("parent %s does not exist", l.To)
		}
		now := time.Now().UTC()
		stub := &Fragment{
			ID: l.To, Kind: KindStub, Title: "Stub", Summary: "Auto-created link target.",
			Status: StatusActive, Salience: 1.0, Created: now, Updated: now, Source: "ai",
			Links: []Link{{Rel: RelChildOf, To: "mem_inbox", Weight: 1.0}},
		}
		view[l.To] = stub
	}
	addEdge(f, l.Rel, l.To, l.Note)
	return nil
}

func (s *Store) movePlanned(view map[string]*Fragment, f *Fragment, parent string) error {
	if s.wouldCreateCycle(f.ID, parent, view) {
		return fmt.Errorf("moving %s under %s would create a cycle", f.ID, parent)
	}
	// remove existing child_of, set new
	f.Links = removeEdge(f.Links, RelChildOf, "")
	f.Links = append(f.Links, Link{Rel: RelChildOf, To: parent, Weight: 1.0})
	return nil
}

func (s *Store) findNearDuplicate(title, summary string) (*Fragment, string) {
	if s.cfg.NearDuplicateThreshold <= 0 {
		return nil, ""
	}
	probe := tokenize(title + " " + summary)
	if len(probe) == 0 {
		return nil, ""
	}
	needle := title + " " + summary
	var best *Fragment
	bestSim := s.cfg.NearDuplicateThreshold
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.idx.byID {
		sim := jaccardTrigrams(needle, f.Title+" "+f.Summary)
		if sim >= bestSim {
			best = f
			bestSim = sim
		}
	}
	if best == nil {
		return nil, ""
	}
	return best, best.ID
}

// trigrams returns the set of character trigrams of s.
func trigrams(s string) map[string]bool {
	set := map[string]bool{}
	low := strings.ToLower(strings.TrimSpace(s))
	if len(low) < 3 {
		if low != "" {
			set[low] = true
		}
		return set
	}
	for i := 0; i+3 <= len(low); i++ {
		set[low[i:i+3]] = true
	}
	return set
}

func jaccardTrigrams(a, b string) float64 {
	ta := trigrams(a)
	tb := trigrams(b)
	if len(ta) == 0 && len(tb) == 0 {
		return 1
	}
	inter := 0
	for k := range ta {
		if tb[k] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// scan rejects fragments whose text contains credentials.
func (s *Store) scan(f *Fragment) error {
	if !s.cfg.SecretScan {
		return nil
	}
	text := f.Title + "\n" + f.Summary + "\n" + f.Body
	if span, ok := secretScan(text); ok {
		return fmt.Errorf("refusing to store a likely secret (redacted: %s)", span)
	}
	return nil
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                           // AWS access key
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),                                 // GitHub tokens
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),                                        // OpenAI-style
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),                         // private keys
	regexp.MustCompile(`(?i)password\s*[=:]\s*\S+`),                                  // password=
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`), // JWT
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`),                                      // Google API key
}

func secretScan(text string) (string, bool) {
	for _, re := range secretPatterns {
		m := re.FindString(text)
		if m != "" {
			return redact(m), true
		}
	}
	return "", false
}

func redact(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-2:]
}

// commit writes all dirty fragments (staged to .tmp then renamed), moves
// deletions to .archive, and swaps the live index. All files are staged first
// so a failure anywhere in the batch leaves the fragments dir unchanged.
func (s *Store) commit(view map[string]*Fragment, dirty, deleted map[string]bool) error {
	type staged struct {
		src, dst string
	}
	var files []staged
	now := time.Now().UTC()
	for id := range dirty {
		f, ok := view[id]
		if !ok {
			continue
		}
		f.Updated = now
		b, err := encodeFragment(f)
		if err != nil {
			return err
		}
		if int64(len(b)) > int64(s.cfg.MaxBodyKB)*1024+16384 {
			return fmt.Errorf("fragment %s exceeds max_body_kb", id)
		}
		tmpPath := filepath.Join(s.tmp, id+".md")
		if err := writeTempThenRename(tmpPath, b); err != nil {
			return err
		}
		files = append(files, staged{src: tmpPath, dst: s.pathFor(id)})
	}
	// renames (fragments dir becomes visible only now)
	for _, f := range files {
		if err := os.Rename(f.src, f.dst); err != nil {
			return err
		}
	}
	// deletions: move file to .archive
	for id := range deleted {
		if undeletableIDs[id] {
			continue
		}
		src := s.pathFor(id)
		dst := s.archivePathFor(id)
		_ = os.Rename(src, dst)
	}
	// update live index
	s.mu.Lock()
	for id := range deleted {
		s.idx.remove(id)
	}
	for id := range dirty {
		if f, ok := view[id]; ok {
			s.idx.remove(id) // reindex fully
			s.idx.add(f)     // add stores metadata only; bodies stay lazy
		}
	}
	s.mu.Unlock()
	s.flushSnapshot()
	return nil
}

func cloneFragment(f *Fragment) *Fragment {
	c := *f
	c.Tags = append([]string(nil), f.Tags...)
	c.Links = append([]Link(nil), f.Links...)
	return &c
}

func addEdge(f *Fragment, rel Rel, to, note string) {
	for _, l := range f.Links {
		if l.Rel == rel && l.To == to {
			return
		}
	}
	f.Links = append(f.Links, Link{Rel: rel, To: to, Note: note, Weight: 1.0})
}

func removeEdge(links []Link, rel Rel, to string) []Link {
	out := links[:0]
	for _, l := range links {
		if l.Rel == rel && (to == "" || l.To == to) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// dedupeStrings returns a new slice with blanks and duplicates removed; the
// input slice is never mutated.
func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func floatOf(p *float32) float32 {
	if p == nil {
		return 0
	}
	return *p
}

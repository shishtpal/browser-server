package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MaintainReport summarizes one maintenance run.
type MaintainReport struct {
	Decayed       int      `json:"decayed"`
	Archived      []string `json:"archived"`
	Purged        int      `json:"purged"`
	Orphans       []string `json:"orphans"`
	DanglingEdges int      `json:"dangling_edges"`
	OldStubs      []string `json:"old_stubs"`
	InboxBacklog  int      `json:"inbox_backlog"`
	DedupeHints   []string `json:"dedupe_hints"`
	Reindexed     bool     `json:"reindexed"`
}

// Maintain runs the periodic hygiene job: salience decay, archive sweep,
// purge, verify, reindex, dedupe suggestions. It is called at boot and then on
// maintenance_interval, gated by auto_cleanup.
func (s *Store) Maintain(ctx context.Context) (MaintainReport, error) {
	rep := MaintainReport{}
	if !s.enabled {
		return rep, fmt.Errorf("memory disabled")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// 1) Salience decay
	now := time.Now()
	s.mu.Lock()
	for _, f := range s.idx.byID {
		if err := ctx.Err(); err != nil {
			s.mu.Unlock()
			return rep, err
		}
		if f.Pinned || f.Kind == KindPersona || f.Kind == KindProject || f.Kind == KindIndex {
			continue
		}
		// Fragments never read have no access timestamp; decay only applies
		// to real access history, not to freshly written ones.
		if f.Accessed.IsZero() {
			continue
		}
		weeks := timeSinceWeeks(now, f.Accessed)
		if weeks > 0 && f.Salience > 0 {
			f.Salience = float32(float64(f.Salience) * powF(s.cfg.SalienceDecayPerWeek, weeks))
			if f.Salience < 0.01 {
				f.Salience = 0.01
			}
			rep.Decayed++
		}
	}
	s.mu.Unlock()

	// 2) Archive sweep
	var toArchive []*Fragment
	s.mu.RLock()
	for _, f := range s.idx.byID {
		if f.Status == StatusActive && !f.Pinned &&
			f.Kind != KindPersona && f.Kind != KindProject && f.Kind != KindGlossary && f.Kind != KindIndex &&
			f.Salience < float32(s.cfg.ArchiveThreshold) && time.Since(f.Updated).Hours()/24 > float64(s.cfg.RetentionDays) {
			cp := *f
			toArchive = append(toArchive, &cp)
		}
	}
	s.mu.RUnlock()
	for _, f := range toArchive {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		if _, err := s.archiveOne(ctx, f.ID); err != nil {
			continue
		}
		rep.Archived = append(rep.Archived, f.ID)
	}

	// 3) Purge .archive older than 2x retention
	purged, err := s.purgeArchive(ctx, now)
	if err != nil {
		return rep, err
	}
	rep.Purged = purged

	// 4) Verify
	rep.Orphans = s.orphans()
	rep.DanglingEdges = s.danglingEdges()
	rep.OldStubs = s.oldStubs(now)
	s.mu.RLock()
	rep.InboxBacklog = len(s.idx.childIDs("mem_inbox"))
	s.mu.RUnlock()

	// 5) Reindex
	if err := s.reindex(ctx); err != nil {
		return rep, err
	}
	rep.Reindexed = true

	// 6) Dedupe suggestions -> write task fragments into mem_inbox
	rep.DedupeHints = s.dedupeSuggestions(ctx)

	s.flushSnapshot()
	return rep, nil
}

// archiveOne sets status=archived and moves the file to .archive.
func (s *Store) archiveOne(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	f, ok := s.idx.byID[id]
	if !ok {
		s.mu.Unlock()
		return "", fmt.Errorf("no fragment %s", id)
	}
	f.Status = StatusArchived
	cp := *f
	s.mu.Unlock()
	// The index entry is metadata-only; reattach the on-disk body so the
	// archived file keeps it.
	if body, err := s.readBody(id); err == nil {
		cp.Body = body
	}
	b, err := encodeFragment(&cp)
	if err != nil {
		return "", err
	}
	if err := writeTempThenRename(s.archivePathFor(id), b); err != nil {
		return "", err
	}
	_ = os.Remove(s.pathFor(id))
	s.flushSnapshot()
	return id, nil
}

func (s *Store) purgeArchive(ctx context.Context, now time.Time) (int, error) {
	entries, err := os.ReadDir(s.archive)
	if err != nil {
		return 0, nil
	}
	cutoff := now.Add(-2 * time.Duration(s.cfg.RetentionDays) * 24 * time.Hour)
	n := 0
	for _, en := range entries {
		if err := ctx.Err(); err != nil {
			return n, err
		}
		info, e := en.Info()
		if e != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if os.Remove(filepath.Join(s.archive, en.Name())) == nil {
				n++
			}
		}
	}
	return n, nil
}

func (s *Store) danglingEdges() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, f := range s.idx.byID {
		for _, l := range f.Links {
			if _, ok := s.idx.byID[l.To]; !ok {
				n++
			}
		}
	}
	return n
}

func (s *Store) oldStubs(now time.Time) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []string
	for _, f := range s.idx.byID {
		if f.Kind == KindStub && now.Sub(f.Created) > 30*24*time.Hour {
			out = append(out, f.ID)
		}
	}
	return out
}

func (s *Store) reindex(ctx context.Context) error {
	// Rebuild adjacency from fragments (bodies not held, but postings rebuild
	// from metadata only is lossy for body terms; a full reindex reads files).
	entries, err := os.ReadDir(s.frags)
	if err != nil {
		return err
	}
	s.mu.Lock()
	ix := newIndex()
	s.mu.Unlock()
	for _, en := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if en.IsDir() || filepath.Ext(en.Name()) != ".md" {
			continue
		}
		if f, e := s.readFile(filepath.Join(s.frags, en.Name())); e == nil {
			ix.add(f) // add stores metadata only; bodies stay lazy
		}
	}
	s.mu.Lock()
	s.idx = ix
	s.mu.Unlock()
	return nil
}

func (s *Store) dedupeSuggestions(ctx context.Context) []string {
	// Snapshot metadata under a read lock, then compare outside the lock.
	type pair struct{ id, title, summary string }
	var frags []pair
	s.mu.RLock()
	for id, f := range s.idx.byID {
		if id == "mem_root" || id == "mem_self" || id == "mem_user" {
			continue
		}
		frags = append(frags, pair{id, f.Title, f.Summary})
	}
	s.mu.RUnlock()
	sort.Slice(frags, func(i, j int) bool { return frags[i].id < frags[j].id })
	var hints []string
	for i := 0; i < len(frags); i++ {
		if err := ctx.Err(); err != nil {
			return hints
		}
		for j := i + 1; j < len(frags); j++ {
			if jaccardTrigrams(frags[i].title+" "+frags[i].summary, frags[j].title+" "+frags[j].summary) >= 0.9 {
				hints = append(hints, fmt.Sprintf("%s ~ %s", frags[i].id, frags[j].id))
			}
		}
	}
	// record suggestions as task fragments in mem_inbox
	if len(hints) > 0 {
		now := time.Now().UTC()
		var b string
		for _, h := range hints {
			b += "- " + h + "\n"
		}
		f := &Fragment{
			ID:   "mem_task_dedupe_" + fmt.Sprintf("%d", now.Unix()),
			Kind: KindTask, Title: "Dedupe memory candidates", Summary: "Near-duplicate fragments to review and merge.",
			Body: b, Status: StatusActive, Salience: 1.0, Created: now, Updated: now, Source: "system",
			Links: []Link{{Rel: RelChildOf, To: "mem_inbox", Weight: 1.0}},
		}
		if enc, err := encodeFragment(f); err == nil {
			if writeTempThenRename(s.pathFor(f.ID), enc) == nil {
				s.mu.Lock()
				s.idx.add(f)
				s.mu.Unlock()
			}
		}
	}
	return hints
}

func timeSinceWeeks(now time.Time, t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return now.Sub(t).Hours() / (7 * 24)
}

func powF(base float64, exp float64) float64 {
	r := 1.0
	for i := 0; i < int(exp); i++ {
		r *= base
	}
	return r
}

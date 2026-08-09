package memory

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

// RecallArgs is the read tool argument set. Mode is inferred from which fields
// are present (see mode resolution below).
type RecallArgs struct {
	Query  string   `json:"query"`
	IDs    []string `json:"ids"`
	From   string   `json:"from"`
	Depth  int      `json:"depth"`
	Rels   []Rel    `json:"rels"`
	Kind   Kind     `json:"kind"`
	Tags   []string `json:"tags"`
	Status string   `json:"status"`
	Detail string   `json:"detail"`
	Limit  int      `json:"limit"`
	// Synthesize asks a cheap "librarian" sub-agent to read the matched
	// fragments and return a concise, sourced answer instead of raw graph JSON.
	Synthesize bool `json:"synthesize"`
}

// RecallNode is one returned fragment (body only when detail=full).
type RecallNode struct {
	ID      string   `json:"id"`
	Kind    Kind     `json:"kind"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags,omitempty"`
	Score   float64  `json:"score"`
	Parent  string   `json:"parent"`
	Body    string   `json:"body,omitempty"`
}

// RecallResult is the response envelope for recall_memory.
type RecallResult struct {
	Nodes        []RecallNode     `json:"nodes"`
	Edges        []Edge           `json:"edges,omitempty"`
	TotalMatches int              `json:"total_matches"`
	Returned     int              `json:"returned"`
	Truncated    bool             `json:"truncated"`
	Hint         string           `json:"hint"`
	Synthesized  *SynthesisResult `json:"synthesized,omitempty"`
}

type scoredFrag struct {
	f      *Fragment
	score  float64
	seeded bool
}

// Recall implements recall_memory: search / fetch / traverse, with optional
// librarian synthesis.
func (s *Store) Recall(ctx context.Context, a RecallArgs) (RecallResult, error) {
	if !s.enabled {
		return RecallResult{}, fmt.Errorf("memory disabled")
	}
	depth := a.Depth
	if a.Depth == 0 {
		depth = s.cfg.DefaultDepth
	}
	if depth < 0 {
		depth = 0
	}
	if depth > s.cfg.MaxDepth {
		depth = s.cfg.MaxDepth
	}
	status := Status(a.Status)
	if status == "" || status == "any" {
		status = ""
	}
	detail := a.Detail
	if detail == "" {
		detail = "summary"
	}
	limit := a.Limit
	if limit == 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	s.mu.RLock()
	scored, total := s.score(ctx, a, depth, status, detail)
	// collect edges for traversal results
	edges := s.collectEdges(scored)
	s.mu.RUnlock()
	// Record reads so recency/access priors and salience decay are fed.
	for _, sc := range scored {
		s.touchAccess(sc.f.ID)
	}

	// Dedupe, sort by score desc, keep only allowed status.
	keep := make([]scoredFrag, 0, len(scored))
	for _, sc := range scored {
		if status != "" && sc.f.Status != status {
			continue
		}
		keep = append(keep, sc)
	}
	sort.Slice(keep, func(i, j int) bool {
		if keep[i].score == keep[j].score {
			return keep[i].f.ID < keep[j].f.ID
		}
		return keep[i].score > keep[j].score
	})
	if len(keep) > limit {
		keep = keep[:limit]
	}

	nodes := make([]RecallNode, 0, len(keep))
	usedBytes := 0
	capBytes := s.cfg.MaxResultBytes
	truncated := false
	for i, sc := range keep {
		n := RecallNode{
			ID: sc.f.ID, Kind: sc.f.Kind, Title: sc.f.Title, Summary: sc.f.Summary,
			Tags: sc.f.Tags, Score: math.Round(sc.score*100) / 100,
			Parent: parentOf(sc.f),
		}
		if detail == "full" {
			// lazy body read from disk
			if body, err := s.readBody(sc.f.ID); err == nil {
				n.Body = body
			}
		}
		sz := len(n.ID) + len(n.Title) + len(n.Summary) + len(n.Body)
		if i > 0 && usedBytes+sz > capBytes {
			truncated = true
			continue
		}
		usedBytes += sz
		nodes = append(nodes, n)
	}

	result := RecallResult{
		Nodes: nodes, Edges: edges, TotalMatches: total,
		Returned: len(nodes), Truncated: truncated,
	}
	if len(nodes) > 0 && total > len(nodes) {
		result.Hint = fmt.Sprintf("%d matches; narrow with tags/kind or raise limit", total)
	}
	if a.Synthesize && s.cfg.Synthesizer.Enabled && s.completer != nil {
		syn, err := s.synthesize(ctx, a.Query, nodes, edges)
		if err == nil {
			result.Synthesized = &syn
			result.Nodes = nil
			result.Edges = nil
			result.Hint = ""
		} else if !s.cfg.Synthesizer.FallbackOnError {
			return RecallResult{}, fmt.Errorf("synthesize: %w", err)
		}
	}
	return result, nil
}

func parentOf(f *Fragment) string {
	for _, l := range f.Links {
		if l.Rel == RelChildOf {
			return l.To
		}
	}
	return ""
}

// score resolves the retrieval mode and returns scored candidates plus the
// total match count. Caller holds s.mu.RLock.
func (s *Store) score(_ context.Context, a RecallArgs, depth int, status Status, detail string) ([]scoredFrag, int) {
	// nil relSet means "all edge types". An empty (non-nil) map would filter
	// out every edge, so only build it when the caller restricts rels.
	var relSet map[Rel]bool
	if len(a.Rels) > 0 {
		relSet = map[Rel]bool{}
		for _, r := range a.Rels {
			relSet[r] = true
		}
	}

	switch {
	case len(a.IDs) > 0:
		return s.scoreByIDs(a.IDs, depth, relSet, status)
	case a.From != "" && a.Query == "":
		return s.scoreBrowse(a.From, depth, relSet, status, a.Kind, a.Tags)
	case a.Query != "" && a.From == "":
		return s.scoreQuery(a.Query, depth, relSet, status, a.Kind, a.Tags, nil)
	case a.Query != "" && a.From != "":
		sub := s.reachable(a.From, depth)
		return s.scoreQuery(a.Query, depth, relSet, status, a.Kind, a.Tags, sub)
	default:
		return s.scorePersona()
	}
}

func (s *Store) scoreByIDs(ids []string, depth int, relSet map[Rel]bool, status Status) ([]scoredFrag, int) {
	out := []scoredFrag{}
	seen := map[string]bool{}
	// Emit the requested ids first, in caller order, above any prior-based
	// score: ids mode is a direct fetch, not a ranked search.
	var direct []*Fragment
	for _, id := range ids {
		f, ok := s.idx.byID[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		direct = append(direct, f)
	}
	for i, f := range direct {
		out = append(out, scoredFrag{f: f, score: 1e6 + float64(len(direct)-i), seeded: true})
	}
	// Then expand outward from each in the same order.
	for _, f := range direct {
		for _, n := range s.expand(f, depth, relSet) {
			if !seen[n.ID] {
				seen[n.ID] = true
				out = append(out, scoredFrag{f: n, score: s.priorScore(n)})
			}
		}
	}
	return out, len(out)
}

func (s *Store) scoreBrowse(from string, depth int, relSet map[Rel]bool, status Status, kind Kind, tags []string) ([]scoredFrag, int) {
	f, ok := s.idx.byID[from]
	if !ok {
		return nil, 0
	}
	out := []scoredFrag{{f: f, score: s.priorScore(f), seeded: true}}
	seen := map[string]bool{from: true}
	for _, n := range s.neighbors(f, relSet, kind, tags) {
		if seen[n.ID] {
			continue
		}
		seen[n.ID] = true
		out = append(out, scoredFrag{f: n, score: s.priorScore(n)})
	}
	return out, len(out)
}

func (s *Store) scoreQuery(query string, depth int, relSet map[Rel]bool, status Status, kind Kind, tags []string, restrict map[string]bool) ([]scoredFrag, int) {
	words := tokenize(query)
	lex := map[string]float64{}
	if len(words) > 0 {
		n := float64(len(s.idx.byID))
		for _, w := range words {
			df := float64(s.idx.docFreq[w])
			idf := math.Log(1 + (n-df+0.5)/(df+0.5))
			if idf < 0 {
				idf = 0
			}
			for _, p := range s.idx.postings[w] {
				if restrict != nil && !restrict[p.ID] {
					continue
				}
				f, ok := s.idx.byID[p.ID]
				if !ok {
					continue
				}
				if kind != "" && f.Kind != kind {
					continue
				}
				if status != "" && f.Status != status {
					continue
				}
				if !tagsMatch(f.Tags, tags) {
					continue
				}
				tfSat := float64(p.TF) / (float64(p.TF) + 1.2)
				lex[p.ID] += idf * fieldWeight[p.Field] * tfSat
			}
		}
		// exact phrase bonus
		q := strings.ToLower(strings.TrimSpace(query))
		for id := range lex {
			f := s.idx.byID[id]
			if f != nil && strings.Contains(strings.ToLower(f.Title+" "+f.Summary), q) {
				lex[id] += 2.0 * float64(len(words))
			}
		}
	}
	// build candidates (include seeded structural hits too when no words)
	scored := []scoredFrag{}
	seen := map[string]bool{}
	for id, sc := range lex {
		f := s.idx.byID[id]
		if f == nil {
			continue
		}
		score := sc * s.priorScore(f)
		scored = append(scored, scoredFrag{f: f, score: score, seeded: true})
		seen[id] = true
	}
	if len(words) == 0 {
		// browse-like: fall back to persona context for empty query with from handled elsewhere
		return scored, len(scored)
	}
	// spreading activation across edges (depth <= 2)
	if depth > 0 {
		for _, sd := range scored {
			if sd.score <= 0 {
				continue
			}
			for _, nb := range s.expand(sd.f, 2, relSet) {
				if seen[nb.ID] {
					continue
				}
				seen[nb.ID] = true
				scored = append(scored, scoredFrag{f: nb, score: 0.05 * sd.score, seeded: false})
			}
		}
	}
	return scored, len(scored)
}

func (s *Store) scorePersona() ([]scoredFrag, int) {
	out := []scoredFrag{}
	for _, id := range []string{"mem_root", "mem_self", "mem_user", "mem_projects"} {
		if f, ok := s.idx.byID[id]; ok {
			out = append(out, scoredFrag{f: f, score: s.priorScore(f)})
		}
	}
	return out, len(out)
}

// priorScore applies recency / access / salience / pinned / archived priors.
// The recency factor uses Accessed when the fragment has been recalled at
// least once, falling back to Updated so brand-new fragments still get a
// sensible score before any read occurs.
func (s *Store) priorScore(f *Fragment) float64 {
	var p float64 = 1.0
	p *= 1 + 0.15*math.Log1p(float64(f.AccessCount))
	recency := f.Accessed
	if recency.IsZero() {
		recency = f.Updated
	}
	if !recency.IsZero() {
		days := time.Since(recency).Hours() / 24
		if days < 0 {
			days = 0
		}
		rec := 1.0 - 0.3*math.Min(1.0, days/180.0)
		p *= rec
	}
	if f.Salience > 0 {
		p *= float64(f.Salience)
	}
	if f.Pinned {
		p *= 1.25
	}
	if f.Status == StatusArchived {
		p *= 0.3
	}
	return p
}

// expand returns fragments reachable from f within `depth` hops along edges
// allowed by relSet (empty = all). Uses a visited set; cycles are safe.
func (s *Store) expand(f *Fragment, depth int, relSet map[Rel]bool) []*Fragment {
	seen := map[string]bool{f.ID: true}
	var out []*Fragment
	var visit func(id string, d int)
	visit = func(id string, d int) {
		if d <= 0 {
			return
		}
		cur, ok := s.idx.byID[id]
		if !ok {
			return
		}
		for _, nb := range s.neighbors(cur, relSet, "", nil) {
			if seen[nb.ID] {
				continue
			}
			seen[nb.ID] = true
			out = append(out, nb)
			visit(nb.ID, d-1)
		}
	}
	visit(f.ID, depth)
	return out
}

// neighbors returns the graph neighbours of f (children + outbound targets +
// inbound reverse targets), optionally filtered by rel set / kind / tags.
func (s *Store) neighbors(f *Fragment, relSet map[Rel]bool, kind Kind, tags []string) []*Fragment {
	out := []*Fragment{}
	for _, nb := range s.neighborsWithEdges(f, relSet, kind, tags) {
		out = append(out, nb.frag)
	}
	return out
}

// neighborsWithEdges is like neighbors but also returns the edge actually
// traversed to reach each neighbour, normalized to From=f with the derived
// (inverse) rel for inbound edges so callers never have to guess direction.
type neighborEdge struct {
	frag *Fragment
	edge Edge
}

func (s *Store) neighborsWithEdges(f *Fragment, relSet map[Rel]bool, kind Kind, tags []string) []neighborEdge {
	out := []neighborEdge{}
	seen := map[string]bool{f.ID: true}
	push := func(nb *Fragment, e Edge) {
		if nb == nil || seen[nb.ID] {
			return
		}
		if kind != "" && nb.Kind != kind {
			return
		}
		if !tagsMatch(nb.Tags, tags) {
			return
		}
		seen[nb.ID] = true
		out = append(out, neighborEdge{frag: nb, edge: e})
	}
	for _, c := range s.idx.childIDs(f.ID) {
		if relSet == nil || relSet[RelChildOf] {
			push(s.idx.byID[c], Edge{From: f.ID, Rel: "has_child", To: c})
		}
	}
	for _, l := range f.Links {
		if relSet != nil && !relSet[l.Rel] {
			continue
		}
		push(s.idx.byID[l.To], Edge{From: f.ID, Rel: l.Rel, To: l.To, Note: l.Note, Weight: l.Weight})
	}
	for _, e := range s.idx.inboundEdges(f.ID) {
		if relSet != nil && !relSet[e.Rel] {
			continue
		}
		rel := e.Rel
		if e.Rel == RelChildOf {
			rel = RelChildOf // f is the parent; child_of reads naturally from f
		} else if inv, ok := inverseRels[e.Rel]; ok {
			rel = inv
		}
		push(s.idx.byID[e.From], Edge{From: f.ID, Rel: rel, To: e.From, Note: e.Note, Weight: e.Weight})
	}
	return out
}

func (s *Store) collectEdges(scored []scoredFrag) []Edge {
	out := []Edge{}
	seen := map[string]bool{}
	for _, sc := range scored {
		f := sc.f
		for _, l := range f.Links {
			key := f.ID + "\x00" + string(l.Rel) + "\x00" + l.To
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Edge{From: f.ID, Rel: l.Rel, To: l.To, Note: l.Note, Weight: l.Weight})
		}
	}
	return out
}

// reachable returns the set of IDs reachable from `from` within depth hops
// (used to restrict search to a subgraph).
func (s *Store) reachable(from string, depth int) map[string]bool {
	set := map[string]bool{from: true}
	f, ok := s.idx.byID[from]
	if !ok {
		return set
	}
	for _, nb := range s.expand(f, depth, nil) {
		set[nb.ID] = true
	}
	return set
}

func (s *Store) readBody(id string) (string, error) {
	if s.idx == nil {
		return "", fmt.Errorf("no index")
	}
	b, err := os.ReadFile(s.pathFor(id))
	if err != nil {
		return "", err
	}
	f, err := decodeFragment(b)
	if err != nil {
		return "", err
	}
	return f.Body, nil
}

func tagsMatch(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	haveSet := map[string]bool{}
	for _, t := range have {
		haveSet[t] = true
	}
	for _, w := range want {
		if !haveSet[w] {
			return false
		}
	}
	return true
}

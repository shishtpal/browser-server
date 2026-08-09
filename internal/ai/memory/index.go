package memory

import (
	"encoding/json"
	"strings"
	"sync"
)

// Field weights for lexical scoring (see search.go).
const (
	fieldTitle = iota
	fieldTags
	fieldSummary
	fieldBody
)

var fieldWeight = [4]float64{3.0, 2.0, 1.5, 1.0}

// Posting maps one token to a fragment and the field in which it occurred.
type Posting struct {
	ID    string
	Field int
	TF    int
}

// Index is the in-memory retrieval structure: a metadata map (bodies are
// lazy), a tree of children, a reverse edge index, and an inverted index.
// Reads and writes are guarded by Store.mu.
type Index struct {
	Version  int
	byID     map[string]*Fragment
	children map[string][]string
	inbound  map[string][]Edge
	postings map[string][]Posting
	docFreq  map[string]int
	aliases  map[string]string
	vectors  map[string][]float32
	// terms holds the unique tokens per fragment (title+tags+summary+body).
	// Bodies themselves are never retained in memory; this compact token cache
	// is what lets a later update/delete scrub a fragment's body postings.
	terms map[string][]string

	mu sync.RWMutex
}

func newIndex() *Index {
	return &Index{
		Version:  2,
		byID:     map[string]*Fragment{},
		children: map[string][]string{},
		inbound:  map[string][]Edge{},
		postings: map[string][]Posting{},
		docFreq:  map[string]int{},
		aliases:  map[string]string{},
		vectors:  map[string][]float32{},
		terms:    map[string][]string{},
	}
}

// tokenize lowercases and splits on non-alphanumeric runs.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
}

// add indexes a fragment (postings for title/tags/summary/body), its child
// edge, and its outgoing links. Bodies are indexed for search but never kept
// in memory: the byID entry stores metadata only, while ix.terms keeps the
// compact token list needed to scrub postings on a later update/delete.
func (ix *Index) add(f *Fragment) {
	all := tokenize(f.Title)
	all = append(all, tokenize(f.Summary)...)
	all = append(all, tokenize(strings.Join(f.Tags, " "))...)
	all = append(all, tokenize(f.Body)...)
	uniq := map[string]bool{}
	for _, t := range all {
		if t != "" {
			uniq[t] = true
		}
	}
	termList := make([]string, 0, len(uniq))
	for t := range uniq {
		termList = append(termList, t)
	}
	ix.terms[f.ID] = termList

	tokens := [][]string{
		tokenize(f.Title),
		tokenize(strings.Join(f.Tags, " ")),
		tokenize(f.Summary),
		tokenize(f.Body),
	}
	for field, toks := range tokens {
		seen := map[string]bool{}
		for _, t := range toks {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			ix.addTerm(t, f.ID, field, countOccurrences(toks, t))
		}
	}
	// Store metadata only (bodies stay lazy and off-memory).
	meta := *f
	meta.Body = ""
	ix.byID[f.ID] = &meta
	ix.updateAdjacency(&meta)
}

func countOccurrences(toks []string, t string) int {
	n := 0
	for _, x := range toks {
		if x == t {
			n++
		}
	}
	return n
}

func (ix *Index) addTerm(t, id string, field, tf int) {
	pl := ix.postings[t]
	if len(ix.postings[t]) == 0 {
		ix.docFreq[t] = 0
	}
	pl = append(pl, Posting{ID: id, Field: field, TF: tf})
	ix.postings[t] = pl
	ix.docFreq[t]++
}

// updateAdjacency rebuilds the children and inbound maps for f.
func (ix *Index) updateAdjacency(f *Fragment) {
	for _, c := range ix.children[f.ID] {
		ix.removeChildEdge(f.ID, c)
	}
	ix.children[f.ID] = nil
	ix.inbound[f.ID] = nil
	ix.addEdges(f.ID, f.Links)
}

// addEdges records child_of edges in children and all edges in inbound.
func (ix *Index) addEdges(from string, links []Link) {
	seen := map[string]bool{}
	for _, l := range links {
		key := string(l.Rel) + "\x00" + l.To
		if seen[key] {
			continue
		}
		seen[key] = true
		e := Edge{From: from, Rel: l.Rel, To: l.To, Note: l.Note, Weight: l.Weight}
		if l.Rel == RelChildOf {
			ix.children[l.To] = append(ix.children[l.To], from)
		}
		ix.inbound[l.To] = append(ix.inbound[l.To], e)
	}
}

func (ix *Index) removeChildEdge(parent, child string) {
	out := ix.children[parent][:0]
	for _, c := range ix.children[parent] {
		if c != child {
			out = append(out, c)
		}
	}
	ix.children[parent] = out
}

// remove deletes a fragment's postings and edges from the index.
func (ix *Index) remove(id string) {
	if f, ok := ix.byID[id]; ok {
		for _, t := range ix.terms[id] {
			ix.dropPosting(t, id)
		}
		// remove child edges
		for _, l := range f.Links {
			if l.Rel == RelChildOf {
				ix.removeChildEdge(l.To, id)
			}
		}
	}
	ix.inbound[id] = nil
	delete(ix.byID, id)
	delete(ix.vectors, id)
	delete(ix.terms, id)
}

func (ix *Index) dropPosting(t, id string) {
	pl := ix.postings[t]
	out := pl[:0]
	for _, p := range pl {
		if p.ID != id {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		delete(ix.postings, t)
		delete(ix.docFreq, t)
	} else {
		ix.postings[t] = out
		ix.docFreq[t] = len(out)
	}
}

// childIDs returns the children of parent.
func (ix *Index) childIDs(parent string) []string {
	return append([]string(nil), ix.children[parent]...)
}

// inboundEdges returns the edges pointing at id.
func (ix *Index) inboundEdges(id string) []Edge {
	return append([]Edge(nil), ix.inbound[id]...)
}

// ---------------------------------------------------------------------------
// Snapshot / restore (index.json). Postings and docFreq are persisted so a
// restore does not need to re-read bodies from disk; the metadata map (bodies
// cleared) and edge maps are persisted too.
// ---------------------------------------------------------------------------

type snapPosting struct {
	ID    string `json:"id"`
	Field int    `json:"field"`
	TF    int    `json:"tf"`
}

type snapshotData struct {
	Version   int                      `json:"version"`
	Fragments []*Fragment              `json:"fragments"`
	Aliases   map[string]string        `json:"aliases,omitempty"`
	Vectors   map[string][]float32     `json:"vectors,omitempty"`
	Postings  map[string][]snapPosting `json:"postings"`
	DocFreq   map[string]int           `json:"doc_freq"`
}

// snapshot serializes the index (without holding the lock; callers lock).
func (ix *Index) snapshot() ([]byte, error) {
	frags := make([]*Fragment, 0, len(ix.byID))
	for _, f := range ix.byID {
		cp := *f
		cp.Body = ""
		frags = append(frags, &cp)
	}
	sd := snapshotData{
		Version:   ix.Version,
		Fragments: frags,
		Aliases:   ix.aliases,
		Vectors:   ix.vectors,
		DocFreq:   ix.docFreq,
		Postings:  map[string][]snapPosting{},
	}
	for t, pl := range ix.postings {
		sp := make([]snapPosting, 0, len(pl))
		for _, p := range pl {
			sp = append(sp, snapPosting{ID: p.ID, Field: p.Field, TF: p.TF})
		}
		sd.Postings[t] = sp
	}
	return json.Marshal(sd)
}

// restore replaces the index with persisted data and rebuilds the edge maps
// from fragment links (bodies were not persisted, so byID entries have empty
// bodies and postings come from the snapshot).
func (ix *Index) restore(b []byte) error {
	var sd snapshotData
	if err := json.Unmarshal(b, &sd); err != nil {
		return err
	}
	ix.Version = sd.Version
	ix.byID = map[string]*Fragment{}
	ix.children = map[string][]string{}
	ix.inbound = map[string][]Edge{}
	ix.postings = map[string][]Posting{}
	ix.docFreq = map[string]int{}
	ix.aliases = sd.Aliases
	ix.vectors = sd.Vectors
	ix.terms = map[string][]string{}
	if ix.aliases == nil {
		ix.aliases = map[string]string{}
	}
	if ix.vectors == nil {
		ix.vectors = map[string][]float32{}
	}
	for _, f := range sd.Fragments {
		f.Body = ""
		ix.byID[f.ID] = f
		ix.updateAdjacency(f)
	}
	for t, sp := range sd.Postings {
		pl := make([]Posting, 0, len(sp))
		for _, p := range sp {
			pl = append(pl, Posting{ID: p.ID, Field: p.Field, TF: p.TF})
			ix.terms[p.ID] = append(ix.terms[p.ID], t)
		}
		ix.postings[t] = pl
	}
	// de-duplicate terms lists
	for id, list := range ix.terms {
		seen := map[string]bool{}
		out := list[:0]
		for _, t := range list {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
		ix.terms[id] = out
	}
	ix.docFreq = sd.DocFreq
	if ix.docFreq == nil {
		ix.docFreq = map[string]int{}
	}
	return nil
}

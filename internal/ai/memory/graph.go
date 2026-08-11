package memory

import "sort"

// graph.go holds tree/graph invariants shared by writes and maintenance:
// ancestry, reachability, and cycle guarding. These read only the in-memory
// index and are called with s.mu held (R or W) by the caller.

// ancestors returns the chain of child_of parents from id up to (but not
// including) the root sentinel where parent is "".
func (s *Store) ancestors(id string) []string {
	var out []string
	cur := id
	for {
		f, ok := s.idx.byID[cur]
		if !ok {
			break
		}
		parent := parentOf(f)
		if parent == "" || parent == cur {
			break
		}
		out = append(out, parent)
		cur = parent
	}
	return out
}

// childOf returns the child_of target of id, or "".
func childOf(f *Fragment) string { return parentOf(f) }

// reachableSetFromRoot returns the set of fragment ids reachable from mem_root
// by following child_of edges. Used by verify to detect orphans.
func (s *Store) reachableSetFromRoot() map[string]bool {
	set := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		if set[id] {
			return
		}
		set[id] = true
		for _, c := range s.idx.childIDs(id) {
			visit(c)
		}
	}
	visit("mem_root")
	return set
}

// orphans returns ids that exist but are not reachable from mem_root.
func (s *Store) orphans() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reach := s.reachableSetFromRoot()
	var out []string
	for id := range s.idx.byID {
		if !reach[id] {
			out = append(out, id)
		}
	}
	return out
}

// wouldCreateCycle reports whether re-parenting `id` under `newParent` would
// introduce a child_of cycle (newParent is id itself or one of id's
// descendants). It checks the caller-supplied working view first (so
// within-batch moves see pending re-parents) and falls back to the live
// index under a read lock.
func (s *Store) wouldCreateCycle(id, newParent string, view ...map[string]*Fragment) bool {
	if newParent == id {
		return true
	}
	seen := map[string]bool{id: true}
	cur := newParent
	var lookup func(string) (*Fragment, bool)
	if len(view) > 0 && view[0] != nil {
		v := view[0]
		lookup = func(x string) (*Fragment, bool) {
			if f, ok := v[x]; ok {
				return f, true
			}
			s.mu.RLock()
			f, ok := s.idx.byID[x]
			s.mu.RUnlock()
			return f, ok
		}
	} else {
		lookup = func(x string) (*Fragment, bool) {
			s.mu.RLock()
			f, ok := s.idx.byID[x]
			s.mu.RUnlock()
			return f, ok
		}
	}
	for cur != "" {
		if seen[cur] {
			return true
		}
		seen[cur] = true
		f, ok := lookup(cur)
		if !ok {
			return false
		}
		next := parentOf(f)
		if next == cur {
			return false
		}
		cur = next
	}
	return false
}

// descendants returns the set of ids reachable downward from id via child_of.
func (s *Store) descendants(id string) map[string]bool {
	set := map[string]bool{}
	var visit func(cur string)
	visit = func(cur string) {
		for _, c := range s.idx.childIDs(cur) {
			if set[c] {
				continue
			}
			set[c] = true
			visit(c)
		}
	}
	visit(id)
	return set
}

// hasChildren reports whether id has any child_of descendants. It takes a read
// lock so it is safe to call from the write path.
func (s *Store) hasChildren(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.idx.childIDs(id)) > 0
}

// GraphNode is a node in the admin graph view.
type GraphNode struct {
	ID      string `json:"id"`
	Kind    Kind   `json:"kind"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// GraphView is the payload for GET /api/ai/memory/graph.
type GraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []Edge      `json:"edges"`
}

// Graph returns every stored fragment and every non-dangling relationship for
// the admin viewer. It intentionally does not use MaxDepth: that limit belongs
// to recall, whereas an administrator needs to be able to inspect disconnected
// fragments as well as the tree rooted at mem_root.
func (s *Store) Graph() (GraphView, bool) {
	if !s.enabled {
		return GraphView{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.idx.byID))
	for id := range s.idx.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	g := GraphView{Nodes: make([]GraphNode, 0, len(ids))}
	for _, id := range ids {
		f := s.idx.byID[id]
		g.Nodes = append(g.Nodes, GraphNode{ID: f.ID, Kind: f.Kind, Title: f.Title, Summary: f.Summary})
	}

	// child_of is stored on the child, but emitted parent -> child to match the
	// layout direction used by the graph UI. Other relationships retain their
	// stored direction.
	seenE := map[string]bool{}
	for _, id := range ids {
		parent := s.idx.byID[id]
		for _, l := range parent.Links {
			if _, ok := s.idx.byID[l.To]; !ok || l.To == parent.ID {
				continue
			}
			edge := Edge{From: parent.ID, Rel: l.Rel, To: l.To, Note: l.Note, Weight: l.Weight}
			if l.Rel == RelChildOf {
				edge.From, edge.To = l.To, parent.ID
			}
			key := edge.From + "\x00" + string(edge.Rel) + "\x00" + edge.To
			if seenE[key] {
				continue
			}
			seenE[key] = true
			g.Edges = append(g.Edges, edge)
		}
	}
	return g, true
}

// Get returns a full fragment (including body) by id, for the edit UI. It
// reads the body lazily from disk so the returned fragment is complete, and
// touches the fragment's access metadata.
func (s *Store) Get(id string) (*Fragment, bool) {
	if !s.enabled {
		return nil, false
	}
	if !s.touchAccess(id) {
		return nil, false
	}
	s.mu.RLock()
	meta := cloneFragment(s.idx.byID[id])
	s.mu.RUnlock()
	if body, err := s.readBody(id); err == nil {
		meta.Body = body
	}
	return meta, true
}

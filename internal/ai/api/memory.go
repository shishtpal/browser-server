package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"browser-server/internal/ai/memory"
)

// MemoryStats reports memory store health/status.
func (m *Module) MemoryStats(w http.ResponseWriter, r *http.Request) {
	if m.memory == nil {
		writeError(w, http.StatusNotFound, "memory_unavailable", "memory is disabled")
		return
	}
	writeJSON(w, http.StatusOK, m.memory.Stats())
}

// MemoryMaintain runs the hygiene job (salience decay, archive, purge,
// verify, reindex, dedupe) on demand.
func (m *Module) MemoryMaintain(w http.ResponseWriter, r *http.Request) {
	if m.memory == nil {
		writeError(w, http.StatusNotFound, "memory_unavailable", "memory is disabled")
		return
	}
	rep, err := m.memory.Maintain(context.Background())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "maintain_failed", err.Error())
		return
	}
	if len(rep.Archived) > 0 || rep.Purged > 0 || len(rep.DedupeHints) > 0 {
		log.Printf("AI memory maintenance (manual): archived=%d purged=%d dedupe_hints=%d", len(rep.Archived), rep.Purged, len(rep.DedupeHints))
	}
	writeJSON(w, http.StatusOK, rep)
}

// MemoryGraph returns the visible graph from mem_root for a graph viewer.
func (m *Module) MemoryGraph(w http.ResponseWriter, r *http.Request) {
	if m.memory == nil {
		writeError(w, http.StatusNotFound, "memory_unavailable", "memory is disabled")
		return
	}
	g, ok := m.memory.Graph()
	if !ok {
		writeError(w, http.StatusInternalServerError, "graph_failed", "could not build graph")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// MemoryFragment returns a single full fragment (with body) for editing.
func (m *Module) MemoryFragment(w http.ResponseWriter, r *http.Request) {
	if m.memory == nil {
		writeError(w, http.StatusNotFound, "memory_unavailable", "memory is disabled")
		return
	}
	id := mux.Vars(r)["id"]
	f, ok := m.memory.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "fragment not found")
		return
	}
	writeJSON(w, http.StatusOK, fragmentPayload{ID: f.ID, Kind: string(f.Kind), Title: f.Title, Summary: f.Summary, Body: f.Body, Tags: f.Tags, Status: string(f.Status), Pinned: f.Pinned, Parent: memoryParentOf(f), Links: linksPayload(f.Links)})
}

type fragmentPayload struct {
	ID      string        `json:"id"`
	Kind    string        `json:"kind"`
	Title   string        `json:"title"`
	Summary string        `json:"summary"`
	Body    string        `json:"body"`
	Tags    []string      `json:"tags"`
	Status  string        `json:"status"`
	Pinned  bool          `json:"pinned"`
	Parent  string        `json:"parent"`
	Links   []linkPayload `json:"links"`
}

type linkPayload struct {
	Rel  string `json:"rel"`
	To   string `json:"to"`
	Note string `json:"note"`
}

func memoryParentOf(f *memory.Fragment) string {
	for _, l := range f.Links {
		if l.Rel == "child_of" {
			return l.To
		}
	}
	return ""
}

func linksPayload(links []memory.Link) []linkPayload {
	out := make([]linkPayload, 0, len(links))
	for _, l := range links {
		out = append(out, linkPayload{Rel: string(l.Rel), To: l.To, Note: l.Note})
	}
	return out
}

// MemoryWrite accepts a write_memory batch from the UI and applies it
// atomically via the shared store.
func (m *Module) MemoryWrite(w http.ResponseWriter, r *http.Request) {
	if m.memory == nil {
		writeError(w, http.StatusNotFound, "memory_unavailable", "memory is disabled")
		return
	}
	var req struct {
		Ops []memory.WriteOp `json:"ops"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	res, err := m.memory.Write(r.Context(), memory.WriteArgs{Ops: req.Ops})
	if err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

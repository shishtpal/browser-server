package tools

import (
	"context"
	"encoding/json"

	"browser-server/internal/ai/memory"
)

// v2 memory system exposes exactly two model-facing tools over the shared
// memory.Store: recall_memory (read/search/traverse, with an optional
// synthesize mode that delegates to a cheap "librarian" sub-agent) and
// write_memory (batched, transactional mutations). Everything operational
// (cache, retention, dedupe) is automatic or an admin endpoint.

const recallMemorySchema = `{
  "type": "object",
  "properties": {
    "query":  { "type": "string",  "description": "Free-text query. Omit to browse structurally." },
    "ids":    { "type": "array", "items": { "type": "string" }, "description": "Fetch these fragments directly." },
    "from":   { "type": "string",  "description": "Anchor id to traverse from, e.g. mem_root or mem_projects." },
    "depth":  { "type": "integer", "minimum": 0, "maximum": 3, "default": 1, "description": "Hops to expand from the anchor / search hits." },
    "rels":   { "type": "array", "items": { "enum": ["child_of","relates","depends_on","supersedes","about","contradicts","source"] }, "description": "Restrict traversal to these edge types. Default: all." },
    "kind":   { "type": "string" },
    "tags":   { "type": "array", "items": { "type": "string" } },
    "status": { "type": "string", "enum": ["active","archived","superseded","any"], "default": "active" },
    "detail": { "type": "string", "enum": ["titles","summary","full"], "default": "summary" },
    "limit":  { "type": "integer", "minimum": 1, "maximum": 50, "default": 10 },
    "synthesize": { "type": "boolean", "default": false, "description": "If true, a cheap local model reads the matched fragments and returns a concise, factual summary/answer instead of raw graph JSON. Use for 'what did we decide about X?' or 'give me the history of Y'." }
  },
  "additionalProperties": false
}`

const writeMemorySchema = `{
  "type": "object",
  "properties": {
    "ops": {
      "type": "array",
      "minItems": 1,
      "maxItems": 20,
      "items": {
        "type": "object",
        "properties": {
          "op":     { "enum": ["upsert","append","link","unlink","move","archive","delete"] },
          "id":     { "type": "string" },
          "kind":   { "type": "string" },
          "title":  { "type": "string" },
          "summary":{ "type": "string", "maxLength": 280 },
          "body":   { "type": "string" },
          "tags":   { "type": "array", "items": { "type": "string" } },
          "parent": { "type": "string", "description": "Parent fragment id. Defaults to mem_inbox on create." },
          "pinned": { "type": "boolean" },
          "confidence": { "type": "number", "minimum": 0, "maximum": 1 },
          "links":  { "type": "array", "items": { "type": "object", "properties": {
                        "rel":  { "enum": ["relates","depends_on","supersedes","about","contradicts","source"] },
                        "to":   { "type": "string" },
                        "note": { "type": "string" }
                      }, "required": ["rel","to"], "additionalProperties": false } },
          "from":   { "type": "string", "description": "For link/unlink." },
          "rel":    { "type": "string", "description": "For link/unlink." },
          "to":     { "type": "string", "description": "For link/unlink." },
          "note":   { "type": "string", "description": "For link/unlink." },
          "on_conflict":  { "enum": ["merge","new","error"], "default": "merge" },
          "superseded_by":{ "type": "string", "description": "For archive." },
          "cascade":{ "type": "boolean", "default": false, "description": "For delete: also delete descendants." }
        },
        "required": ["op"],
        "additionalProperties": false
      }
    }
  },
  "required": ["ops"],
  "additionalProperties": false
}`

var recallAllowed = map[string]bool{
	"query": true, "ids": true, "from": true, "depth": true, "rels": true,
	"kind": true, "tags": true, "status": true, "detail": true, "limit": true, "synthesize": true,
}

var writeAllowed = map[string]bool{"ops": true}

// registerMemoryTools registers the two v2 memory tools over the shared store.
func registerMemoryTools(r *Registry, s *memory.Store) {
	if s == nil {
		return
	}
	r.add(Tool{
		Name:        "recall_memory",
		Category:    "Memory",
		Description: "Read the memory graph. Search by text, fetch by id, or walk outward from a fragment. Fragments form a tree rooted at mem_root (persona) plus typed cross-links. Set synthesize=true to have a cheap local model summarize the matched fragments into a concise, sourced answer.",
		Schema:      json.RawMessage(recallMemorySchema),
		Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a memory.RecallArgs
			if err := strict(raw, &a, recallAllowed); err != nil {
				return nil, err
			}
			return s.Recall(ctx, a)
		},
		RawContentFunc: rawRecallMemoryFormatter,
	})
	r.add(Tool{
		Name:        "write_memory",
		Category:    "Memory",
		Description: "Create, update, link, move, archive or delete memory fragments. Operations are validated as a batch and applied atomically. Prefer upsert with a stable slug id over creating new fragments.",
		Schema:      json.RawMessage(writeMemorySchema),
		Execute: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a memory.WriteArgs
			if err := strict(raw, &a, writeAllowed); err != nil {
				return nil, err
			}
			return s.Write(ctx, a)
		},
	})
}

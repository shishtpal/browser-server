package tools

import (
	"context"
	"encoding/json"
)

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

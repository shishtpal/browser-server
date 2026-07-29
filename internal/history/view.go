package history

import "browser-server/internal/models"

// SearchMap renders a History entry as a flat map for AI tool output. It
// mirrors bookmark.SearchMap so the tool layer stays free of storage detail.
func SearchMap(h models.History) map[string]any {
	return map[string]any{
		"id":         h.ID,
		"url":        h.URL,
		"title":      h.Title,
		"domain":     h.Domain,
		"visited_at": h.VisitedAt,
		"duration":   h.Duration,
	}
}

// SearchMaps renders a slice of History entries for AI tool output.
func SearchMaps(entries []models.History) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, h := range entries {
		out = append(out, SearchMap(h))
	}
	return out
}

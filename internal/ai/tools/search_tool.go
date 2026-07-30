package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const SearchToolName = "search_tool"

type SearchToolMatch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Active      bool   `json:"active"`
	Loaded      bool   `json:"loaded"`
	score       int
	order       int
}

type SearchToolResult struct {
	Query   string            `json:"query"`
	Matches []SearchToolMatch `json:"matches"`
	Loaded  []string          `json:"loaded"`
}

func registerSearchTool(r *Registry) {
	r.add(Tool{
		Name:        SearchToolName,
		Category:    "Discovery",
		Description: "Search available AI tools by capability. Up to five matching active tools are loaded for use during the current message.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Capability, tool name, or category to search for"},"limit":{"type":"integer","description":"Maximum matches to return (1-5)","minimum":1,"maximum":5}},"required":["query"],"additionalProperties":false}`),
		Execute: func(_ context.Context, raw json.RawMessage) (any, error) {
			return r.Search(raw)
		},
	})
}

func (r *Registry) Search(raw json.RawMessage) (SearchToolResult, error) {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := strict(raw, &args, map[string]bool{"query": true, "limit": true}); err != nil {
		return SearchToolResult{}, err
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return SearchToolResult{}, fmt.Errorf("query is required")
	}
	if args.Limit == 0 {
		args.Limit = 5
	}
	if args.Limit < 1 || args.Limit > 5 {
		return SearchToolResult{}, fmt.Errorf("limit must be between 1 and 5")
	}

	query := strings.ToLower(args.Query)
	terms := strings.Fields(query)
	names := r.allowed
	if len(names) == 0 {
		names = make([]string, 0, len(r.tools))
		for name := range r.tools {
			names = append(names, name)
		}
		sort.Strings(names)
	}

	matches := make([]SearchToolMatch, 0, args.Limit)
	for order, name := range names {
		if name == SearchToolName {
			continue
		}
		tool, ok := r.tools[name]
		if !ok {
			continue
		}
		score := searchToolScore(tool, query, terms)
		if score == 0 {
			continue
		}
		matches = append(matches, SearchToolMatch{
			Name: tool.Name, Description: tool.Description, Category: tool.Category, score: score, order: order,
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].order < matches[j].order
	})
	if len(matches) > args.Limit {
		matches = matches[:args.Limit]
	}
	return SearchToolResult{Query: args.Query, Matches: matches, Loaded: []string{}}, nil
}

func searchToolScore(tool Tool, query string, terms []string) int {
	name := strings.ToLower(tool.Name)
	category := strings.ToLower(tool.Category)
	description := strings.ToLower(tool.Description)
	score := 0
	if name == query {
		score += 1000
	} else if strings.Contains(name, query) {
		score += 500
	}
	if strings.Contains(category, query) {
		score += 200
	}
	if strings.Contains(description, query) {
		score += 100
	}
	for _, term := range terms {
		switch {
		case name == term:
			score += 100
		case strings.HasPrefix(name, term):
			score += 70
		case strings.Contains(name, term):
			score += 50
		}
		if strings.Contains(category, term) {
			score += 30
		}
		if strings.Contains(description, term) {
			score += 10
		}
	}
	return score
}

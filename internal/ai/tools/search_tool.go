package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	SearchToolName      = "search_tool"
	DefaultSearchLimit  = 5
	MaxSearchLimit      = 20
	MaxLoadBatchSize    = 20
	MaxTotalLoadedTools = 50
)

type SearchToolMatch struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Summary   string `json:"summary"`
	MatchType string `json:"match_type"`
	Score     int    `json:"score"`
	Loaded    bool   `json:"loaded"`
	order     int
}

type LoadResult struct {
	Loaded           []string `json:"loaded"`
	AlreadyLoaded    []string `json:"already_loaded"`
	Unknown          []string `json:"unknown"`
	Unavailable      []string `json:"unavailable"`
	DefinitionsAdded int      `json:"definitions_added"`
}

type ToolCategory struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type DiscoveryResult struct {
	Action     string            `json:"action"`
	Query      string            `json:"query,omitempty"`
	Category   string            `json:"category,omitempty"`
	Matches    []SearchToolMatch `json:"matches,omitempty"`
	Tools      []string          `json:"tools,omitempty"`
	Load       *LoadResult       `json:"load,omitempty"`
	Categories []ToolCategory    `json:"categories,omitempty"`
}

//go:embed schemas/search_tool.json
var searchToolSchema []byte

func registerSearchTool(r *Registry) {
	r.add(Tool{
		Name:        SearchToolName,
		Category:    "Discovery",
		Description: "Discover and explicitly load AI tools. Search by capability, load exact names, list a category, or enumerate categories.",
		Schema:      json.RawMessage(searchToolSchema),
		Execute: func(_ context.Context, raw json.RawMessage) (any, error) {
			return r.Discover(raw, r.allowedSet(), map[string]bool{})
		},
		RawContentFunc: rawDiscoveryResult,
	})
}

func (r *Registry) Discover(raw json.RawMessage, visible, loaded map[string]bool) (DiscoveryResult, error) {
	var args struct {
		Action   string   `json:"action"`
		Query    string   `json:"query"`
		Names    []string `json:"names"`
		Category string   `json:"category"`
		Limit    *int     `json:"limit"`
		Load     *bool    `json:"load"`
	}
	if err := strict(raw, &args, map[string]bool{
		"action": true, "query": true, "names": true, "category": true, "limit": true, "load": true,
	}); err != nil {
		return DiscoveryResult{}, err
	}
	args.Action = strings.ToLower(strings.TrimSpace(args.Action))
	args.Query = strings.TrimSpace(args.Query)
	args.Category = strings.TrimSpace(args.Category)
	limit := DefaultSearchLimit
	limitKnown := args.Limit != nil
	if args.Limit != nil {
		limit = *args.Limit
	}
	if limit < 1 || limit > MaxSearchLimit {
		return DiscoveryResult{}, fmt.Errorf("limit must be between 1 and %d", MaxSearchLimit)
	}

	switch args.Action {
	case "search":
		if args.Query == "" {
			return DiscoveryResult{}, fmt.Errorf("query is required for search")
		}
		if len(args.Names) > 0 || args.Category != "" {
			return DiscoveryResult{}, fmt.Errorf("names and category are not valid for search")
		}
		return r.searchTools(args.Query, limit, args.Load == nil || *args.Load, visible, loaded), nil
	case "load":
		if len(args.Names) == 0 {
			return DiscoveryResult{}, fmt.Errorf("names are required for load")
		}
		if len(args.Names) > MaxLoadBatchSize {
			return DiscoveryResult{}, fmt.Errorf("names must contain at most %d tools", MaxLoadBatchSize)
		}
		if args.Query != "" || args.Category != "" || args.Limit != nil || args.Load != nil {
			return DiscoveryResult{}, fmt.Errorf("query, category, limit, and load are not valid for load")
		}
		result := r.Load(args.Names, visible, loaded)
		return DiscoveryResult{Action: "load", Load: &result}, nil
	case "list":
		if args.Category == "" {
			return DiscoveryResult{}, fmt.Errorf("category is required for list")
		}
		if args.Query != "" || len(args.Names) > 0 || args.Load != nil {
			return DiscoveryResult{}, fmt.Errorf("query, names, and load are not valid for list")
		}
		if !limitKnown {
			// A category enumeration should not silently truncate at the
			// search default; return up to the maximum unless capped.
			limit = MaxSearchLimit
		}
		return DiscoveryResult{Action: "list", Category: normalizeCategory(args.Category), Tools: r.List(args.Category, limit, visible)}, nil
	case "categories":
		if args.Query != "" || len(args.Names) > 0 || args.Category != "" || args.Limit != nil || args.Load != nil {
			return DiscoveryResult{}, fmt.Errorf("categories does not accept query, names, category, limit, or load")
		}
		return DiscoveryResult{Action: "categories", Categories: r.DiscoveryCategories(visible)}, nil
	default:
		return DiscoveryResult{}, fmt.Errorf("action must be search, load, list, or categories")
	}
}

// Search preserves the former registry API for internal callers while routing
// through the explicit search action. New callers should use Discover.
func (r *Registry) Search(raw json.RawMessage) (DiscoveryResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return DiscoveryResult{}, fmt.Errorf("arguments must be a JSON object")
	}
	fields["action"] = json.RawMessage(`"search"`)
	updated, err := json.Marshal(fields)
	if err != nil {
		return DiscoveryResult{}, err
	}
	return r.Discover(updated, r.allowedSet(), map[string]bool{})
}

func (r *Registry) searchTools(query string, limit int, load bool, visible, loaded map[string]bool) DiscoveryResult {
	normalizedQuery := normalizeSearchText(query)
	terms := strings.Fields(normalizedQuery)
	names := sortedToolNames(visible)
	matches := make([]SearchToolMatch, 0, limit)
	for order, name := range names {
		if name == SearchToolName {
			continue
		}
		tool, ok := r.tools[name]
		if !ok || !toolAvailable(tool) {
			continue
		}
		score, matchType := searchToolScore(tool, normalizedQuery, terms)
		if score == 0 {
			continue
		}
		matches = append(matches, SearchToolMatch{
			Name: tool.Name, Category: normalizeCategory(tool.Category), Summary: tool.Description,
			MatchType: matchType, Score: score, Loaded: loaded[tool.Name], order: order,
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].order < matches[j].order
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	result := DiscoveryResult{Action: "search", Query: query, Matches: matches}
	if load {
		names = names[:0]
		for _, match := range matches {
			names = append(names, match.Name)
		}
		loadResult := r.Load(names, visible, loaded)
		result.Load = &loadResult
		loadedNow := make(map[string]bool, len(loadResult.Loaded)+len(loadResult.AlreadyLoaded))
		for _, name := range loadResult.Loaded {
			loadedNow[name] = true
		}
		for _, name := range loadResult.AlreadyLoaded {
			loadedNow[name] = true
		}
		for i := range result.Matches {
			result.Matches[i].Loaded = loadedNow[result.Matches[i].Name]
		}
	}
	return result
}

func (r *Registry) Load(names []string, visible, loaded map[string]bool) LoadResult {
	result := LoadResult{Loaded: []string{}, AlreadyLoaded: []string{}, Unknown: []string{}, Unavailable: []string{}}
	seen := make(map[string]bool, len(names))
	// The cap applies to the per-turn loadedToolSet, which starts empty on
	// each chat submit, so this budget cannot deadlock across turns.
	availableSlots := MaxTotalLoadedTools - len(loaded)
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if name == SearchToolName {
			result.AlreadyLoaded = append(result.AlreadyLoaded, name)
			continue
		}
		tool, ok := r.tools[name]
		if !ok {
			result.Unknown = append(result.Unknown, name)
			continue
		}
		if !visible[name] || !toolAvailable(tool) {
			result.Unavailable = append(result.Unavailable, name)
			continue
		}
		if loaded[name] {
			result.AlreadyLoaded = append(result.AlreadyLoaded, name)
			continue
		}
		if availableSlots <= 0 {
			result.Unavailable = append(result.Unavailable, name)
			continue
		}
		result.Loaded = append(result.Loaded, name)
		availableSlots--
	}
	result.DefinitionsAdded = len(result.Loaded)
	return result
}

func (r *Registry) List(category string, limit int, visible map[string]bool) []string {
	wanted := normalizeCategory(category)
	result := make([]string, 0, limit)
	for _, name := range sortedToolNames(visible) {
		tool, ok := r.tools[name]
		if !ok || !toolAvailable(tool) || name == SearchToolName || normalizeCategory(tool.Category) != wanted {
			continue
		}
		result = append(result, name)
		if len(result) == limit {
			break
		}
	}
	return result
}

func (r *Registry) DiscoveryCategories(visible map[string]bool) []ToolCategory {
	counts := map[string]int{}
	for name := range visible {
		tool, ok := r.tools[name]
		if !ok || !toolAvailable(tool) || name == SearchToolName {
			continue
		}
		counts[normalizeCategory(tool.Category)]++
	}
	categories := make([]ToolCategory, 0, len(counts))
	for name, count := range counts {
		categories = append(categories, ToolCategory{Name: name, Count: count})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Name < categories[j].Name })
	return categories
}

func (r *Registry) allowedSet() map[string]bool {
	visible := make(map[string]bool, len(r.allowed))
	if len(r.allowed) == 0 {
		for name := range r.tools {
			visible[name] = true
		}
		return visible
	}
	for _, name := range r.allowed {
		visible[name] = true
	}
	return visible
}

func sortedToolNames(visible map[string]bool) []string {
	names := make([]string, 0, len(visible))
	for name := range visible {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func searchToolScore(tool Tool, query string, terms []string) (int, string) {
	name := normalizeSearchText(tool.Name)
	category := normalizeSearchText(tool.Category)
	description := normalizeSearchText(tool.Description)
	keywords := normalizeSearchText(strings.Join(tool.Keywords, " "))
	score := 0
	matchType := ""
	if name == query {
		return 1000, "exact_name"
	} else if strings.HasPrefix(name, query) {
		score += 500
		matchType = "name_prefix"
	} else if strings.Contains(name, query) {
		score += 350
		matchType = "name_contains"
	}
	if strings.Contains(keywords, query) {
		score += 300
		matchType = strongerMatchType(matchType, "keyword")
	}
	if strings.Contains(category, query) {
		score += 100
		matchType = strongerMatchType(matchType, "category")
	}
	if strings.Contains(description, query) {
		score += 50
		matchType = strongerMatchType(matchType, "description")
	}
	for _, term := range terms {
		switch {
		case name == term:
			score += 120
			matchType = strongerMatchType(matchType, "name_contains")
		case strings.HasPrefix(name, term):
			score += 80
			matchType = strongerMatchType(matchType, "name_prefix")
		case strings.Contains(name, term):
			score += 60
			matchType = strongerMatchType(matchType, "name_contains")
		}
		if strings.Contains(keywords, term) {
			score += 50
			matchType = strongerMatchType(matchType, "keyword")
		}
		if strings.Contains(category, term) {
			score += 20
			matchType = strongerMatchType(matchType, "category")
		}
		if strings.Contains(description, term) {
			score += 10
			matchType = strongerMatchType(matchType, "description")
		}
	}
	return score, matchType
}

func strongerMatchType(current, candidate string) string {
	priority := map[string]int{"description": 1, "category": 2, "keyword": 3, "name_contains": 4, "name_prefix": 5}
	if priority[candidate] > priority[current] {
		return candidate
	}
	return current
}

func normalizeCategory(value string) string {
	return strings.Join(strings.Fields(normalizeSearchText(value)), "_")
}

func normalizeSearchText(value string) string {
	var out strings.Builder
	var previousLower bool
	for _, r := range value {
		if unicode.IsUpper(r) && previousLower {
			out.WriteByte(' ')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(unicode.ToLower(r))
			previousLower = unicode.IsLower(r) || unicode.IsDigit(r)
		} else {
			out.WriteByte(' ')
			previousLower = false
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func rawDiscoveryResult(value any) ([]byte, bool) {
	result, ok := value.(DiscoveryResult)
	if !ok {
		return nil, false
	}
	var lines []string
	lines = append(lines, "action="+result.Action)
	switch result.Action {
	case "search":
		for _, match := range result.Matches {
			lines = append(lines, fmt.Sprintf("%s|%s|%s|%d|loaded=%t", match.Name, match.Category, match.MatchType, match.Score, match.Loaded))
		}
	// "load" has no header lines of its own; the shared result.Load block
	// below renders the complete compact result for it.
	case "list":
		lines = append(lines, "category="+result.Category, "tools="+strings.Join(result.Tools, ","))
	case "categories":
		categories := make([]string, 0, len(result.Categories))
		for _, category := range result.Categories {
			categories = append(categories, fmt.Sprintf("%s:%d", category.Name, category.Count))
		}
		lines = append(lines, "categories="+strings.Join(categories, ","))
	}
	if result.Load != nil {
		lines = append(lines,
			"loaded="+strings.Join(result.Load.Loaded, ","),
			"already_loaded="+strings.Join(result.Load.AlreadyLoaded, ","),
			"unknown="+strings.Join(result.Load.Unknown, ","),
			"unavailable="+strings.Join(result.Load.Unavailable, ","),
			fmt.Sprintf("definitions_added=%d", result.Load.DefinitionsAdded),
		)
	}
	return []byte(strings.Join(lines, "\n")), true
}

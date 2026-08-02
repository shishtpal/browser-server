package searchengine

import (
	"context"
	"math"
	"sort"
	"strings"
)

// Strategy scores and sorts candidates. Different sources can supply different
// strategies (fuzzy text vs exact regex matches) while sharing the engine.
type Strategy[T any] interface {
	Score(ctx context.Context, terms []string, c Candidate[T]) float64
	Sort(hits []Hit[T], candidates []Candidate[T])
}

// FuzzyStrategy returns the default text-matching strategy.
func FuzzyStrategy[T any]() Strategy[T] {
	return fuzzyStrategy[T]{}
}

type fuzzyStrategy[T any] struct{}

const (
	fuzzyMatchThreshold = 0.75
	minScore            = 0.01
)

func (fuzzyStrategy[T]) Score(_ context.Context, terms []string, c Candidate[T]) float64 {
	if len(terms) == 0 {
		return 0
	}
	if len(c.Fields) == 0 {
		return 0
	}

	maxWeight := 1.0
	for _, field := range c.Fields {
		if field.Weight > maxWeight {
			maxWeight = field.Weight
		}
	}
	quality := 0.0
	matched := 0
	for _, term := range terms {
		bestTermScore := 0.0
		for _, field := range c.Fields {
			normalized := normalizeText(field.Text)
			if normalized == "" {
				continue
			}
			// A zero Weight (the zero value) means "unweighted, searchable at
			// weight 1"; only an explicitly negative weight would exclude a field.
			weight := field.Weight
			if weight <= 0 {
				weight = 1
			}
			fieldScore := scoreField(term, normalized) * weight
			if fieldScore > bestTermScore {
				bestTermScore = fieldScore
			}
		}
		if bestTermScore >= minScore {
			matched++
			quality += math.Min(bestTermScore/maxWeight, 1)
		}
	}
	if matched == 0 {
		// No term matched any field well enough; reject the candidate.
		return 0
	}

	// Coverage occupies the primary score band. Match quality is constrained to
	// a smaller band, so matching every term always outranks matching fewer,
	// regardless of which weighted field supplied the partial match.
	n := float64(len(terms))
	coverage := float64(matched) / n
	quality /= n
	fullQuery := strings.Join(terms, " ")
	for _, field := range c.Fields {
		if normalizeText(field.Text) != fullQuery {
			continue
		}
		weight := field.Weight
		if weight <= 0 {
			weight = 1
		}
		quality = math.Max(quality, math.Min(weight/maxWeight, 1))
	}
	qualityBand := 1 / (n + 1)
	return (coverage + quality*qualityBand) / (1 + qualityBand)
}

func (fuzzyStrategy[T]) Sort(hits []Hit[T], _ []Candidate[T]) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return lessTie(hits[i], hits[j])
	})
}

func lessTie[T any](a, b Hit[T]) bool {
	if a.baseScore != b.baseScore {
		return a.baseScore > b.baseScore
	}
	if a.sourceRank != b.sourceRank {
		return a.sourceRank < b.sourceRank
	}
	return a.key < b.key
}

// scoreField scores one normalized term against one normalized field.
func scoreField(term, field string) float64 {
	if field == term {
		return 1.0
	}
	fields := strings.Fields(field)
	for _, token := range fields {
		if token == term {
			return 0.9
		}
	}
	for _, token := range fields {
		if strings.HasPrefix(token, term) {
			return 0.7
		}
	}
	if strings.Contains(field, term) {
		return 0.5
	}
	bestFuzzy := 0.0
	for _, token := range fields {
		if math.Abs(float64(len(term)-len(token))) > float64(max(len(term), len(token)))*(1-fuzzyMatchThreshold) {
			continue
		}
		sim := tokenSimilarity(term, token)
		if sim > bestFuzzy {
			bestFuzzy = sim
		}
	}
	if bestFuzzy >= fuzzyMatchThreshold {
		return bestFuzzy * 0.6
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// tokenSimilarity computes a rune-based normalized Levenshtein similarity.
func tokenSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0.0
	}
	dist := editDistance(ra, rb)
	maxLen := len(ra)
	if len(rb) > maxLen {
		maxLen = len(rb)
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func editDistance(a, b []rune) int {
	// Ensure a is the shorter slice to bound memory.
	if len(a) > len(b) {
		a, b = b, a
	}
	previous := make([]int, len(a)+1)
	current := make([]int, len(a)+1)
	for i := range previous {
		previous[i] = i
	}
	for j := 1; j <= len(b); j++ {
		current[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := previous[i] + 1
			ins := current[i-1] + 1
			sub := previous[i-1] + cost
			min := del
			if ins < min {
				min = ins
			}
			if sub < min {
				min = sub
			}
			current[i] = min
		}
		previous, current = current, previous
	}
	return previous[len(a)]
}

// normalizeText lowercases, removes leading/trailing whitespace, and collapses
// internal whitespace runs.
func normalizeText(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return b.String()
}

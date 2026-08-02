package searchengine

import (
	"context"
	"sort"
	"strings"
)

// ExactStrategy ranks pre-matched candidates by match quality. It is intended
// for sources that already perform exact, regex, or literal matching (e.g.
// code search) and want deterministic scoring through the shared engine.
//
// Candidates must be emitted with meaningful BaseScore values (e.g. 1 for
// literal, 0.9 for whole-word, 0.7 for prefix, 0.5 for substring, 0.0 for
// no match but included for ordering). The strategy preserves the loader's
// source order for ties.
func ExactStrategy[T any]() Strategy[T] {
	return exactStrategy[T]{}
}

type exactStrategy[T any] struct{}

func (exactStrategy[T]) Score(_ context.Context, terms []string, c Candidate[T]) float64 {
	if c.BaseScore > 0 {
		return c.BaseScore
	}
	if len(terms) == 0 {
		return 0
	}
	// No BaseScore supplied; fall back to exact substring/token matching over
	// concatenated fields so the exact strategy still does something sensible.
	text := ""
	for _, f := range c.Fields {
		text += " " + normalizeText(f.Text)
	}
	if text == "" {
		return 0
	}
	matched := 0
	for _, t := range terms {
		if strings.Contains(text, t) {
			matched++
		}
	}
	if matched == 0 {
		return 0
	}
	return 0.5 + 0.5*float64(matched)/float64(len(terms))
}

func (exactStrategy[T]) Sort(hits []Hit[T], candidates []Candidate[T]) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return lessTie(hits[i], hits[j])
	})
}

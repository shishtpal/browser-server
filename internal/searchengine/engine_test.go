package searchengine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchDefaults(t *testing.T) {
	loader := staticLoader([]string{"a", "b", "c"})
	page, err := Search(context.Background(), Request{Query: "a"}, loader)
	assertNoError(t, err)
	if page.Page != 1 {
		t.Fatalf("expected page 1, got %d", page.Page)
	}
	if page.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", page.PageSize)
	}
	if len(page.Results) != 1 || fmt.Sprint(page.Results[0].Value) != "a" {
		t.Fatalf("expected one result a, got %v", page.Results)
	}
	if page.Total != 1 {
		t.Fatalf("expected total 1, got %d", page.Total)
	}
	if page.HasMore || page.Truncated {
		t.Fatalf("unexpected metadata: %+v", page)
	}
}

func TestSearchPageValidation(t *testing.T) {
	loader := staticLoader([]string{})
	_, err := Search(context.Background(), Request{PageSize: 101}, loader)
	if err == nil || !strings.Contains(err.Error(), "page_size must be between 1 and 100") {
		t.Fatalf("expected page size error, got %v", err)
	}
	_, err = Search(context.Background(), Request{PageSize: 0}, loader)
	if err != nil {
		t.Fatalf("default page size should be valid: %v", err)
	}
	_, err = Search(context.Background(), Request{PageSize: -1}, loader)
	if err == nil || !strings.Contains(err.Error(), "page_size must be between 1 and 100") {
		t.Fatalf("expected negative page size error, got %v", err)
	}
	page, err := Search(context.Background(), Request{Page: int(^uint(0) >> 1), PageSize: 100}, loader)
	if err != nil || len(page.Results) != 0 {
		t.Fatalf("huge page should safely return no results, got page=%+v err=%v", page, err)
	}
}

func TestSearchPagination(t *testing.T) {
	items := make([]string, 25)
	for i := range items {
		items[i] = fmt.Sprintf("item%d", i)
	}
	loader := staticLoader(items)
	page, err := Search(context.Background(), Request{Query: "item", Page: 2, PageSize: 10}, loader)
	assertNoError(t, err)
	if len(page.Results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(page.Results))
	}
	if !page.HasMore {
		t.Fatalf("expected has_more")
	}
	if page.Total != 25 {
		t.Fatalf("expected total 25, got %d", page.Total)
	}
	last, err := Search(context.Background(), Request{Query: "item", Page: 3, PageSize: 10}, loader)
	assertNoError(t, err)
	if len(last.Results) != 5 || last.HasMore {
		t.Fatalf("expected 5 final results, got %+v", last)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	loader := staticLoader([]string{"HELLO World"})
	page, err := Search(context.Background(), Request{Query: "hello"}, loader)
	assertNoError(t, err)
	if len(page.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(page.Results))
	}
	if page.Results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", page.Results[0].Score)
	}
}

func TestSearchFuzzyMatch(t *testing.T) {
	loader := staticLoader([]string{"hello world", "completely unrelated"})
	page, err := Search(context.Background(), Request{Query: "helo wrld"}, loader)
	assertNoError(t, err)
	if len(page.Results) != 1 || fmt.Sprint(page.Results[0].Value) != "hello world" {
		t.Fatalf("expected fuzzy match, got %v", page.Results)
	}
}

func TestSearchEmptyQueryPreservesOrder(t *testing.T) {
	loader := staticLoader([]string{"first", "second", "third"})
	page, err := Search(context.Background(), Request{Query: ""}, loader)
	assertNoError(t, err)
	if len(page.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(page.Results))
	}
	want := []string{"first", "second", "third"}
	for i, r := range page.Results {
		if fmt.Sprint(r.Value) != want[i] {
			t.Fatalf("expected %q at position %d, got %v", want[i], i, r.Value)
		}
	}
}

func TestSearchDeterministicTies(t *testing.T) {
	loader := staticLoader([]string{"a b c", "a b c", "a b c"})
	first, err := Search(context.Background(), Request{Query: "a b c"}, loader)
	assertNoError(t, err)
	for run := 0; run < 5; run++ {
		page, err := Search(context.Background(), Request{Query: "a b c"}, loader)
		assertNoError(t, err)
		if !reflect.DeepEqual(page.Results, first.Results) {
			t.Fatalf("tie ordering changed between runs: first=%v run=%v", first.Results, page.Results)
		}
	}
}

func TestSearchTieBreakers(t *testing.T) {
	loader := func(_ context.Context, _ CandidateRequest) (CandidateSet[string], error) {
		return CandidateSet[string]{Candidates: []Candidate[string]{
			{Key: "z", Value: "lower base", BaseScore: 0.1, SourceRank: 0, Fields: []Field{{Text: "term"}}},
			{Key: "b", Value: "key b", BaseScore: 0.2, SourceRank: 1, Fields: []Field{{Text: "term"}}},
			{Key: "a", Value: "key a", BaseScore: 0.2, SourceRank: 1, Fields: []Field{{Text: "term"}}},
			{Key: "x", Value: "source first", BaseScore: 0.2, SourceRank: 0, Fields: []Field{{Text: "term"}}},
		}}, nil
	}
	page, err := Search(context.Background(), Request{Query: "term"}, loader)
	assertNoError(t, err)
	want := []string{"source first", "key a", "key b", "lower base"}
	for i, hit := range page.Results {
		if hit.Value != want[i] {
			t.Fatalf("result %d = %q, want %q", i, hit.Value, want[i])
		}
	}
}

func TestSearchRejectsNegativePage(t *testing.T) {
	_, err := Search(context.Background(), Request{Page: -1}, staticLoader(nil))
	if err == nil || !strings.Contains(err.Error(), "page must be at least 1") {
		t.Fatalf("expected page validation error, got %v", err)
	}
}

func TestSearchCancellation(t *testing.T) {
	loader := func(_ context.Context, _ CandidateRequest) (CandidateSet[string], error) {
		candidates := make([]Candidate[string], 1000)
		for i := range candidates {
			candidates[i] = Candidate[string]{Key: fmt.Sprintf("k%d", i), Value: fmt.Sprintf("v%d", i), Fields: []Field{{Name: "text", Text: "word"}}}
		}
		return CandidateSet[string]{Candidates: candidates}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Search(ctx, Request{Query: "word"}, loader)
	if err != context.Canceled {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestSearchTruncated(t *testing.T) {
	loader := func(_ context.Context, req CandidateRequest) (CandidateSet[string], error) {
		candidates := make([]Candidate[string], req.MaxCandidates+1)
		for i := range candidates {
			candidates[i] = Candidate[string]{Key: fmt.Sprintf("k%d", i), Value: fmt.Sprintf("v%d", i), Fields: []Field{{Name: "text", Text: "word"}}}
		}
		return CandidateSet[string]{Candidates: candidates[:req.MaxCandidates], Truncated: true}, nil
	}
	page, err := Search(context.Background(), Request{Query: "word"}, loader, WithCandidateCap[string](100))
	assertNoError(t, err)
	if !page.Truncated {
		t.Fatalf("expected truncated flag")
	}
	if page.Total != 100 {
		t.Fatalf("expected total 100, got %d", page.Total)
	}
}

func TestSearchScoreRange(t *testing.T) {
	loader := staticLoader([]string{"exact phrase match"})
	page, err := Search(context.Background(), Request{Query: "exact phrase match"}, loader)
	assertNoError(t, err)
	if len(page.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(page.Results))
	}
	if page.Results[0].Score < 0 || page.Results[0].Score > 1 {
		t.Fatalf("score %f out of range", page.Results[0].Score)
	}
}

func TestSearchWeightedFields(t *testing.T) {
	loader := func(_ context.Context, req CandidateRequest) (CandidateSet[string], error) {
		candidates := []Candidate[string]{
			{Key: "a", Value: "a", Fields: []Field{{Name: "title", Text: "search term", Weight: 10}, {Name: "body", Text: "other text", Weight: 1}}},
			{Key: "b", Value: "b", Fields: []Field{{Name: "title", Text: "other text", Weight: 10}, {Name: "body", Text: "search term", Weight: 1}}},
		}
		return CandidateSet[string]{Candidates: candidates}, nil
	}
	page, err := Search(context.Background(), Request{Query: "search term"}, loader)
	assertNoError(t, err)
	if len(page.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(page.Results))
	}
	if page.Results[0].Value != "a" {
		t.Fatalf("expected weighted title match to rank first, got %v", page.Results[0].Value)
	}
	if page.Results[0].Score <= page.Results[1].Score {
		t.Fatalf("expected higher score for title match: %f <= %f", page.Results[0].Score, page.Results[1].Score)
	}
}

func TestSearchMultiTermCoverage(t *testing.T) {
	loader := staticLoader([]string{"alpha beta", "alpha only", "beta only"})
	page, err := Search(context.Background(), Request{Query: "alpha beta"}, loader)
	assertNoError(t, err)
	if len(page.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(page.Results))
	}
	if fmt.Sprint(page.Results[0].Value) != "alpha beta" {
		t.Fatalf("expected both-terms match first, got %v", page.Results[0].Value)
	}
}

func TestSearchCoverageOutranksFieldWeight(t *testing.T) {
	loader := func(_ context.Context, _ CandidateRequest) (CandidateSet[string], error) {
		return CandidateSet[string]{Candidates: []Candidate[string]{
			{Key: "partial", Value: "partial", Fields: []Field{{Name: "title", Text: "alpha", Weight: 10}, {Name: "content", Text: "", Weight: 2}}},
			{Key: "complete", Value: "complete", Fields: []Field{{Name: "title", Text: "", Weight: 10}, {Name: "content", Text: "alpha beta", Weight: 2}}},
		}}, nil
	}
	page, err := Search(context.Background(), Request{Query: "alpha beta"}, loader)
	assertNoError(t, err)
	if page.Results[0].Value != "complete" {
		t.Fatalf("complete term coverage must rank first: %+v", page.Results)
	}
}

func TestSearchMatchQualityOrdering(t *testing.T) {
	page, err := Search(context.Background(), Request{Query: "hello"}, staticLoader([]string{
		"hxllo", "sayhello", "helloworld", "say hello now", "hello",
	}))
	assertNoError(t, err)
	want := []string{"hello", "say hello now", "helloworld", "sayhello", "hxllo"}
	for i, hit := range page.Results {
		if hit.Value != want[i] {
			t.Fatalf("quality result %d = %q, want %q; all=%+v", i, hit.Value, want[i], page.Results)
		}
	}
}

func TestSearchExactPhraseOutranksSeparatedTerms(t *testing.T) {
	page, err := Search(context.Background(), Request{Query: "alpha beta"}, staticLoader([]string{
		"alpha x beta", "alpha beta",
	}))
	assertNoError(t, err)
	if page.Results[0].Value != "alpha beta" {
		t.Fatalf("exact field phrase must rank first: %+v", page.Results)
	}
}

func TestSearchExactStrategy(t *testing.T) {
	loader := staticLoader([]string{"exact match", "not here"})
	page, err := Search(context.Background(), Request{Query: "exact match"}, loader, WithStrategy[string](ExactStrategy[string]()))
	assertNoError(t, err)
	if len(page.Results) != 1 || fmt.Sprint(page.Results[0].Value) != "exact match" {
		t.Fatalf("expected exact match, got %v", page.Results)
	}
}

func TestSearchPageJSON(t *testing.T) {
	page := Page[string]{Query: "q", Page: 1, PageSize: 10, Total: 0, Results: []Hit[string]{}}
	b, err := json.Marshal(page)
	assertNoError(t, err)
	if !strings.Contains(string(b), `"total"`) {
		t.Fatalf("expected total in JSON, got %s", b)
	}
}

func staticLoader(items []string) Loader[string] {
	return func(_ context.Context, req CandidateRequest) (CandidateSet[string], error) {
		candidates := make([]Candidate[string], 0, len(items))
		for i, it := range items {
			candidates = append(candidates, Candidate[string]{
				Key:        fmt.Sprintf("k%d", i),
				Value:      it,
				SourceRank: i,
				Fields:     []Field{{Name: "text", Text: it, Weight: 1}},
			})
		}
		return CandidateSet[string]{Candidates: candidates}, nil
	}
}

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  Hello   World  ", "hello world"},
		{"\tA\nB\rC", "a b c"},
	}
	for _, c := range cases {
		got := normalizeText(c.in)
		if got != c.want {
			t.Errorf("normalizeText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTokenSimilarity(t *testing.T) {
	if tokenSimilarity("hello", "hello") != 1.0 {
		t.Fatal("expected identical tokens to be 1")
	}
	if tokenSimilarity("hello", "helo") < 0.75 {
		t.Fatalf("expected typo similarity above threshold, got %f", tokenSimilarity("hello", "helo"))
	}
	if tokenSimilarity("hello", "xyz") != 0.0 {
		t.Fatalf("expected unrelated tokens to be 0, got %f", tokenSimilarity("hello", "xyz"))
	}
}

func TestEmptyQueryReturnsZeroScore(t *testing.T) {
	loader := staticLoader([]string{"a"})
	page, err := Search(context.Background(), Request{Query: ""}, loader)
	assertNoError(t, err)
	if len(page.Results) != 1 || page.Results[0].Score != 0 {
		t.Fatalf("expected zero score for empty query, got %+v", page.Results)
	}
}

func TestDeepEqual(t *testing.T) {
	_ = reflect.DeepEqual
}

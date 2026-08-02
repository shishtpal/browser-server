package searchengine

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkFuzzy5000(b *testing.B) {
	loader := func(_ context.Context, req CandidateRequest) (CandidateSet[string], error) {
		n := req.MaxCandidates
		if n > 5000 {
			n = 5000
		}
		candidates := make([]Candidate[string], n)
		for i := range candidates {
			candidates[i] = Candidate[string]{
				Key:    fmt.Sprintf("k%d", i),
				Value:  fmt.Sprintf("value %d", i),
				Fields: []Field{{Name: "text", Text: "benchmark search candidate number", Weight: 1}},
			}
		}
		return CandidateSet[string]{Candidates: candidates}, nil
	}
	for i := 0; i < b.N; i++ {
		_, err := Search(context.Background(), Request{Query: "benchmark candidate"}, loader, WithCandidateCap[string](5000))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenSimilarity(b *testing.B) {
	for b.Loop() {
		_ = tokenSimilarity("hello", "helo")
	}
}

func BenchmarkEditDistance(b *testing.B) {
	for b.Loop() {
		_ = editDistance([]rune("hello"), []rune("helo"))
	}
}

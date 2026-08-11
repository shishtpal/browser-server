package quiz

import (
	"context"
	"fmt"

	"browser-server/internal/db"
	"browser-server/internal/searchengine"
)

// SearchCandidates loads question records for fuzzy ranking. It does not
// apply the final text ranking or page limit; those are the engine's job.
// It is intended for the search_questions AI tool.
func SearchCandidates(ctx context.Context, filter Filter, maxCandidates int) (searchengine.CandidateSet[Record], error) {
	predicate, args := filter.Where()

	query := "SELECT " + Columns + " FROM questions WHERE " + predicate + " ORDER BY created_at DESC, id DESC"
	if maxCandidates > 0 {
		query += " LIMIT ?"
		args = append(args, maxCandidates+1)
	}
	rows, err := db.QuizDB.QueryContext(ctx, query, args...)
	if err != nil {
		return searchengine.CandidateSet[Record]{}, err
	}
	records, err := ScanAll(rows)
	if err != nil {
		return searchengine.CandidateSet[Record]{}, err
	}
	truncated := maxCandidates > 0 && len(records) > maxCandidates
	if truncated {
		records = records[:maxCandidates]
	}
	candidates := make([]searchengine.Candidate[Record], len(records))
	for i, rec := range records {
		candidates[i] = searchengine.Candidate[Record]{
			Key: fmt.Sprintf("question:%d", rec.Question.ID),
			Fields: []searchengine.Field{
				{Name: "question", Text: rec.Question.Question, Weight: 10},
				{Name: "explanation", Text: rec.Question.Explanation, Weight: 3},
				{Name: "tags", Text: rec.TagText(), Weight: 5},
				{Name: "source", Text: rec.Question.Source, Weight: 2},
			},
			Value:      rec,
			SourceRank: i,
		}
	}
	return searchengine.CandidateSet[Record]{Candidates: candidates, Truncated: truncated}, nil
}

// SampleRandom returns up to count questions matching the filter, chosen at
// random. It backs the random mode of the search_questions tool, which skips
// text ranking entirely.
func SampleRandom(ctx context.Context, filter Filter, count int) ([]Record, error) {
	predicate, args := filter.Where()
	args = append(args, count)
	rows, err := db.QuizDB.QueryContext(ctx,
		"SELECT "+Columns+" FROM questions WHERE "+predicate+" ORDER BY RANDOM() LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	return ScanAll(rows)
}

// SearchHitMap renders a scored question record as the map the
// search_questions tool returns. Input questions include expected_answer so
// the answer survives both JSON and raw output modes.
func SearchHitMap(rec Record, score float64) map[string]any {
	q := rec.Question
	out := map[string]any{
		"id":         q.ID,
		"type":       q.Type,
		"difficulty": q.Difficulty,
		"question":   q.Question,
		"tags":       rec.Tags(),
		"subject":    q.Subject,
		"topic":      q.Topic,
		"sub_topic":  q.SubTopic,
		"score":      score,
	}
	switch q.Type {
	case "single_choice", "multiple_choice":
		out["options"] = rec.Options()
		out["correct_answers"] = rec.CorrectAnswers()
	case "input":
		out["expected_answer"] = rec.ExpectedText()
	case "chronology":
		out["chronology_items"] = rec.ChronologyItems()
	}
	return out
}

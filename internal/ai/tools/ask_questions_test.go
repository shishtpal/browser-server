package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateQuestionArguments(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "defaults kind and id", args: `{"questions":[{"prompt":"Pick one"}]}`, want: "q1"},
		{name: "choice requires options", args: `{"questions":[{"prompt":"Pick one","kind":"choice"}]}`, want: "options is required"},
		{name: "duplicate ids", args: `{"questions":[{"id":"same","prompt":"One"},{"id":"same","prompt":"Two"}]}`, want: "duplicated"},
		{name: "unknown field", args: `{"questions":[{"prompt":"One","extra":true}]}`, want: "unknown argument"},
		{name: "accepts twenty", args: twentyQuestionsJSON(20), want: "q1"},
		{name: "too many", args: twentyQuestionsJSON(21), want: "between 1 and 20"},
		{name: "multiple choice alias", args: `{"questions":[{"prompt":"Pick","kind":"multiple_choice","options":["A"]}]}`, want: "multi_choice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := ValidateQuestionArguments(json.RawMessage(tt.args))
			if tt.want == "q1" {
				if err != nil || request.Questions[0].ID != tt.want || request.Questions[0].Kind != "text" {
					t.Fatalf("request=%+v err=%v", request, err)
				}
				return
			}
			if tt.want == "multi_choice" {
				if err != nil || request.Questions[0].Kind != tt.want {
					t.Fatalf("request=%+v err=%v", request, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want substring %q", err, tt.want)
			}
		})
	}
}

func twentyQuestionsJSON(n int) string {
	questions := make([]map[string]string, n)
	for i := range questions {
		questions[i] = map[string]string{"prompt": "Question"}
	}
	b, _ := json.Marshal(map[string]any{"questions": questions})
	return string(b)
}

func TestRawAskQuestionsResult(t *testing.T) {
	raw, ok := rawAskQuestionsResult(map[string]any{
		"status": "answered",
		"answers": []any{
			map[string]any{"id": "q16", "answer": "C", "skipped": false},
			map[string]any{"id": "q17", "answer": "C", "skipped": false},
			map[string]any{"id": "q18", "answer": "C", "skipped": false},
			map[string]any{"id": "q19", "answer": "C", "skipped": false},
			map[string]any{"id": "q20", "answer": "B", "skipped": false},
		},
	})
	if !ok || string(raw) != "status=answered; skipped=false; answers=q16:C,q17:C,q18:C,q19:C,q20:B" {
		t.Fatalf("raw=%q ok=%t", raw, ok)
	}
}

func TestRawAskQuestionsResultUnavailableKeepsHint(t *testing.T) {
	raw, ok := rawAskQuestionsResult(map[string]any{
		"status": "unavailable",
		"hint":   "No interactive answer channel is available; proceed with explicit assumptions.",
	})
	if !ok || !strings.Contains(string(raw), "status=unavailable") || !strings.Contains(string(raw), "hint=No interactive answer channel") {
		t.Fatalf("raw=%q ok=%t", raw, ok)
	}
}

func TestFormatAskQuestionsResultUsesRawMode(t *testing.T) {
	rawMode := true
	r := New()
	result, err := r.FormatResult(WithRawOutputOverride(context.Background(), &rawMode), "ask_questions", map[string]any{
		"status":  "answered",
		"answers": []any{map[string]any{"id": "q1", "answer": "A", "skipped": false}},
	})
	if err != nil || string(result) != "status=answered; skipped=false; answers=q1:A" {
		t.Fatalf("result=%q err=%v", result, err)
	}
}

func TestQuestionDirectExecutionIsUnavailable(t *testing.T) {
	r := New()
	result, err := r.Execute(context.Background(), "ask_questions", []byte(`{"questions":[{"prompt":"What should I use?"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), `"status":"unavailable"`) {
		t.Fatalf("result=%s", result)
	}
}

func TestQuestionIsRegistered(t *testing.T) {
	r := New()
	if got := len(r.Specs([]string{"ask_questions"})); got != 1 {
		t.Fatalf("ask_questions specs=%d, want 1", got)
	}
}

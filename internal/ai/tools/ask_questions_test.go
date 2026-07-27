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
		{name: "too many", args: `{"questions":[{"prompt":"1"},{"prompt":"2"},{"prompt":"3"},{"prompt":"4"},{"prompt":"5"},{"prompt":"6"}]}`, want: "between 1 and 5"},
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
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want substring %q", err, tt.want)
			}
		})
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

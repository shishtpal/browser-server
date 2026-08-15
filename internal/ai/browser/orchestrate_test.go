package browser

import (
	"encoding/json"
	"testing"

	corebrowser "browser-server/internal/browser"
)

func jsonRaw(s string) json.RawMessage { return json.RawMessage(s) }

func TestValidateStepsRejectsEmpty(t *testing.T) {
	if err := validateSteps(nil); err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestValidateStepsRejectsNestedOrchestrate(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionOrchestrate, Params: jsonRaw(`{"steps":[{"action":"wait"}]}`)},
	}
	if err := validateSteps(steps); err == nil {
		t.Fatal("expected nested orchestrate to error")
	}
}

func TestValidateStepsAcceptsPrimitiveFlow(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionNavigate, Params: jsonRaw(`{"url":"https://example.com"}`)},
		{Action: corebrowser.ActionWait, Params: jsonRaw(`{"wait_ms":500}`)},
		{Action: corebrowser.ActionClick, Selector: jsonRaw(`{"css":"#go"}`)},
		{Action: corebrowser.ActionScrape, Params: jsonRaw(`{"extract":["text"]}`)},
	}
	if err := validateSteps(steps); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateStepsAcceptsEvalSteps(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionEval, Params: jsonRaw(`{"expression":"document.title","mode":"inject"}`)},
		{Action: corebrowser.ActionEval, Params: jsonRaw(`{"expression":"document.title","mode":"cdp"}`)},
		{Action: corebrowser.ActionEval, Params: jsonRaw(`{"expression":"document.title"}`)}, // default mode
	}
	if err := validateSteps(steps); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateStepsRejectsBadEvalMode(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionEval, Params: jsonRaw(`{"expression":"document.title","mode":"remote"}`)},
	}
	if err := validateSteps(steps); err == nil {
		t.Fatal("expected invalid eval mode to error")
	}
}

func TestValidateStepsRejectsEvalWithoutExpression(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionEval, Params: jsonRaw(`{"mode":"inject"}`)},
	}
	if err := validateSteps(steps); err == nil {
		t.Fatal("expected eval without expression to error")
	}
}

func TestValidateMode(t *testing.T) {
	for _, mode := range []string{"", "inject", "cdp"} {
		if err := validateMode(mode); err != nil {
			t.Fatalf("mode %q should be valid: %v", mode, err)
		}
	}
	if err := validateMode("remote"); err == nil {
		t.Fatal("expected invalid mode to error")
	}
}

func TestEvalSpecAcceptsModes(t *testing.T) {
	spec := specs[corebrowser.ActionEval]
	for _, mode := range []string{"", "inject", "cdp"} {
		if err := spec.validate(&args{Expression: "document.title", Mode: mode}); err != nil {
			t.Fatalf("mode %q should pass eval validation: %v", mode, err)
		}
	}
	if err := spec.validate(&args{Expression: "document.title", Mode: "remote"}); err == nil {
		t.Fatal("expected invalid eval mode to error")
	}
	if err := spec.validate(&args{Expression: "", Mode: "inject"}); err == nil {
		t.Fatal("expected missing expression to error")
	}
}

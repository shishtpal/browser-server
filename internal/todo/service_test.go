package todo

import "testing"

func TestValidateCreateInputAppliesDefaults(t *testing.T) {
	input := CreateInput{UserID: 1, Title: "Test todo"}

	if err := ValidateCreateInput(&input); err != nil {
		t.Fatalf("ValidateCreateInput returned error: %v", err)
	}
	if input.Priority != "medium" {
		t.Fatalf("expected default priority medium, got %q", input.Priority)
	}
	if input.Status != "pending" {
		t.Fatalf("expected default status pending, got %q", input.Status)
	}
	if input.Color != "" {
		t.Fatalf("expected empty color, got %q", input.Color)
	}
}

func TestValidateCreateInputRejectsInvalidStatus(t *testing.T) {
	input := CreateInput{UserID: 1, Title: "Test todo", Status: "unknown"}

	err := ValidateCreateInput(&input)
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Fields["status"] == "" {
		t.Fatalf("expected status validation error, got %+v", ve.Fields)
	}
}

func TestParseDateSupportsDateOnlyInput(t *testing.T) {
	got := ParseDate("2024-01-02")
	if got == nil {
		t.Fatal("expected parsed date, got nil")
	}
	if got.Format("2006-01-02") != "2024-01-02" {
		t.Fatalf("expected 2024-01-02, got %s", got.Format("2006-01-02"))
	}
}

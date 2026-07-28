package todo

import "testing"

func TestValidateUpdateInputRejectsInvalidPriority(t *testing.T) {
	input := UpdateInput{UserID: 1, ID: 2, Priority: stringPtr("super")}

	err := ValidateUpdateInput(&input)
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Fields["priority"] == "" {
		t.Fatalf("expected priority validation error, got %+v", ve.Fields)
	}
}

func TestValidateUpdateInputRejectsInvalidStatus(t *testing.T) {
	input := UpdateInput{UserID: 1, ID: 2, Status: stringPtr("unknown")}

	err := ValidateUpdateInput(&input)
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Fields["status"] == "" {
		t.Fatalf("expected status validation error, got %+v", ve.Fields)
	}
}

func stringPtr(s string) *string {
	return &s
}

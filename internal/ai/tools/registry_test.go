package tools

import (
	"context"
	"testing"
)

func TestStrictToolArguments(t *testing.T) {
	r := New()
	if _, err := r.Execute(context.Background(), "get_current_time", []byte(`{"unknown":1}`)); err == nil {
		t.Fatal("expected unknown argument rejection")
	}
	if _, err := r.Execute(context.Background(), "missing", []byte(`{}`)); err == nil {
		t.Fatal("expected unknown tool rejection")
	}
}

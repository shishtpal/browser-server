package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAddCalendarEventValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing user_id",
			input:   `{"title":"Meeting","start_date":"2026-08-01"}`,
			wantErr: "user_id",
		},
		{
			name:    "missing title",
			input:   `{"user_id":1,"start_date":"2026-08-01"}`,
			wantErr: "title",
		},
		{
			name:    "missing start_date",
			input:   `{"user_id":1,"title":"Meeting"}`,
			wantErr: "start_date",
		},
		{
			name:    "invalid user_id",
			input:   `{"user_id":0,"title":"Meeting","start_date":"2026-08-01"}`,
			wantErr: "user_id",
		},
		{
			name:    "invalid priority",
			input:   `{"user_id":1,"title":"Meeting","start_date":"2026-08-01","priority":"super"}`,
			wantErr: "priority",
		},
		{
			name:    "invalid status",
			input:   `{"user_id":1,"title":"Meeting","start_date":"2026-08-01","status":"done"}`,
			wantErr: "status",
		},
		{
			name:    "invalid start_date",
			input:   `{"user_id":1,"title":"Meeting","start_date":"not-a-date"}`,
			wantErr: "start_date",
		},
		{
			name:    "title too long",
			input:   `{"user_id":1,"title":"` + repeat("x", 501) + `","start_date":"2026-08-01"}`,
			wantErr: "title",
		},
		{
			name:    "unknown field",
			input:   `{"user_id":1,"title":"Meeting","start_date":"2026-08-01","bogus":true}`,
			wantErr: "bogus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := addCalendarEvent(context.Background(), json.RawMessage(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func repeat(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[0]
	}
	return string(out)
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

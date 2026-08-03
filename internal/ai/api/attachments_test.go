package api

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "normal", in: "photo.png", want: "photo.png"},
		{name: "trims surrounding whitespace", in: "  photo.png\t", want: "photo.png"},
		{name: "strips forward-slash path", in: "photos/photo.png", want: "photo.png"},
		{name: "strips backslash path", in: `photos\photo.png`, want: "photo.png"},
		{name: "strips control characters", in: "pho\nto\t.png", want: "photo.png"},
		{name: "empty", in: "", want: ""},
		{name: "whitespace only", in: " \t\n ", want: ""},
		{name: "dot", in: ".", want: ""},
		{name: "dotdot", in: "..", want: ""},
		{name: "hidden file", in: ".hidden", want: ""},
		{name: "hidden file with extension", in: ".hidden.png", want: ""},
		{name: "hidden file via path", in: "/photos/.secret.png", want: ""},
		{name: "hidden file after trimming", in: "  .hidden  ", want: ""},
		{name: "caps length at 200", in: strings.Repeat("a", 250), want: strings.Repeat("a", 200)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.in); got != tt.want {
				t.Fatalf("sanitizeFilename(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

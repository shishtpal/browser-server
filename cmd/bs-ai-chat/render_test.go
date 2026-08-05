package main

import "testing"

func TestUnstreamedTail(t *testing.T) {
	cases := []struct {
		name     string
		streamed string
		final    string
		want     string
	}{
		{"identical", "hello", "hello", ""},
		{"notice appended after stream", "hello", "hello\n\n---\n*limit*", "\n\n---\n*limit*"},
		{"final is last of several iterations", "first pass. second pass.", "second pass.", ""},
		{"final unrelated to stream", "intermediate", "answer", "answer"},
		{"empty final", "hello", "", ""},
		{"nothing streamed", "", "answer", "answer"},
	}
	for _, c := range cases {
		if got := unstreamedTail(c.streamed, c.final); got != c.want {
			t.Errorf("%s: unstreamedTail(%q, %q) = %q, want %q", c.name, c.streamed, c.final, got, c.want)
		}
	}
}

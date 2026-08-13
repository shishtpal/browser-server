package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMarkdownToText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text unchanged", "Hello there.", "Hello there."},
		{"crlf normalized", "line one\r\nline two", "line one\nline two"},
		{"heading stripped", "## Release notes", "Release notes"},
		{"bold and emphasis", "**bold** and *star* and _under_", "bold and star and under"},
		{"strikethrough", "~~gone~~ kept", "gone kept"},
		{"link keeps label", "[docs](https://example.com/a-b(c)d)", "docs"},
		{"image keeps alt", "see ![diagram](https://x/y.png) here", "see diagram here"},
		{"reference link", "[site][1] now", "site now"},
		{"autolink keeps url", "<https://example.com>", "https://example.com"},
		{"inline code", "run `go test ./...` first", "run go test ./... first"},
		{"bullet list", "- one\n- two", "one\ntwo"},
		{"ordered list", "1. one\n2) two", "one\ntwo"},
		{"checkbox task", "- [x] done\n- [ ] todo", "done\ntodo"},
		{"blockquote", "> quoted **bold** text", "quoted bold text"},
		{"nested quote", "> > deep", "deep"},
		{
			"fenced code spoken as prose",
			"Install:\n```sh\ngo get example.com\n```",
			"Install:\ngo get example.com",
		},
		{
			"tilde fence",
			"~~~\ncode line\n~~~",
			"code line",
		},
		{
			"table rendered as phrases",
			"| Name | Role |\n| --- | --- |\n| ann | **lead** |\n| bob | dev |",
			"Name, Role\nann, lead\nbob, dev",
		},
		{"horizontal rule dropped", "above\n---\nbelow", "above\nbelow"},
		{"inline html stripped", "quote <b class=\"x\">bold</b> end", "quote bold end"},
		{"html entity unescaped", "a &amp; b &lt;3", "a & b <3"},
		{"blank lines collapsed", "first\n\n\n\nsecond", "first\nsecond"},
		{"empty input", "", ""},
		{"whitespace only", "  \n \t \n", ""},
		{"markup only", "##\n- \n> ", ""},
		{
			"real chat message",
			"Here are the **two** options:\n\n1. Use [goldmark](https://github.com/yuin/goldmark) — full CommonMark.\n2. Hand-roll a stripper.\n\n> Trade-off: *maintenance vs dependency*.",
			"Here are the two options:\nUse goldmark — full CommonMark.\nHand-roll a stripper.\nTrade-off: maintenance vs dependency.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := markdownToText(c.in); got != c.want {
				t.Errorf("markdownToText(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}

// TestGenerateConvertsMarkdown verifies the stripper runs inside Generate so
// the provider payload and the stored gallery row hold spoken words, not
// raw markdown.
func TestGenerateConvertsMarkdown(t *testing.T) {
	const want = "Title\nSay hello to world.\nUse goldmark."
	var captured string
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p struct {
			Input string `json:"input"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("payload unmarshal: %v", err)
		}
		captured = p.Input
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3"))
	}, nil)
	sp, err := svc.Generate(context.Background(), GenerateRequest{
		Text: "# Title\n\nSay **hello** to [world](https://x.dev).\n\nUse [goldmark](https://github.com/yuin/goldmark).",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured != want {
		t.Errorf("provider input = %q, want %q", captured, want)
	}
	if sp.Text != want {
		t.Errorf("stored text = %q, want %q", sp.Text, want)
	}
}

// TestMarkdownOnlyMessageRejected guards the empty-after-strip path: a
// message that is entirely markdown syntax must not reach the provider.
func TestMarkdownOnlyMessageRejected(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("provider should not be called")
	}, nil)
	_, err := svc.Generate(context.Background(), GenerateRequest{Text: "##\n- \n>"})
	if err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("err = %v, want text is required", err)
	}
}

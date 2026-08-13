// markdown.go converts chat messages authored in markdown into plain text
// suitable for speech synthesis. It covers the surface the chat UI renders
// (headings, emphasis, links, images, lists, tables, code, blockquotes,
// inline HTML) rather than full CommonMark compliance; anything unrecognised
// passes through as literal prose so nothing a user can hear is dropped
// silently. Hand-rolled deliberately — no new go.mod dependency for what is
// a text-transform of a UI-authored dialect.
package tts

import (
	"html"
	"regexp"
	"strings"
)

var (
	// Link with a balanced-paren-aware destination: `](... )` where the URL
	// may contain one level of inner parens (Wikipedia-style).
	mdLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\((?:\\.|[^()\\])*(?:\([^()]*\)[^()]*)*\)`)
	mdImageRe   = regexp.MustCompile(`!\[([^\]]*)\]\([^()]*\)`)
	mdLinkRefRe = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`)
	mdAutolink  = regexp.MustCompile(`<((?:https?|mailto):[^<>\s]+)>`)
	mdHeading   = regexp.MustCompile(`^#{1,6}\s+`)
	mdListMark  = regexp.MustCompile(`^(?:[-+*]|\d{1,9}[.)])\s+`)
	mdCheckbox  = regexp.MustCompile(`^\[[ xX]\]\s+`)
	mdInlineCD  = regexp.MustCompile("`+([^`]*)`+")
	mdDel       = regexp.MustCompile(`~~([^~]+)~~`)
	mdBoldStar  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdBoldUnder = regexp.MustCompile(`__([^_]+)__`)
	mdEmphStar  = regexp.MustCompile(`\*([^*\n]+)\*`)
	mdEmphUnder = regexp.MustCompile(`\b_([^_]+)_`)
	mdHTMLTag   = regexp.MustCompile(`</?[A-Za-z][^>]*>`)
	mdTableDiv  = regexp.MustCompile(`^:?-{3,}:?$`)
	mdHr        = regexp.MustCompile(`^\s{0,3}(?:\*\s*){3,}$|^\s{0,3}(?:-\s*){3,}$|^\s{0,3}(?:_\s*){3,}$`)
	mdPunctOnly = regexp.MustCompile(`^[\s#*+~_\[\](){}.<>\-]+$`)
)

// markdownToText converts a markdown message to plain text for the speech
// endpoint. Block boundaries become line breaks so voices pause naturally;
// stray pipes/asterisks from tables and emphasis disappear.
func markdownToText(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return mdRenderBlocks(mdUnfence(strings.Split(src, "\n")))
}

// mdUnfence removes fenced code markers and re-indents fence interiors with
// four spaces so the block pass can keep them as quoted prose (the words are
// spoken; the syntax is not).
func mdUnfence(raw []string) []string {
	out := raw[:0]
	inFence := false
	for _, l := range raw {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, "    "+l)
		} else {
			out = append(out, l)
		}
	}
	return out
}

// mdRenderBlocks walks the line stream emitting one spoken line per block,
// merging table rows and skipping markup-only lines.
func mdRenderBlocks(lines []string) string {
	out := make([]string, 0, len(lines))
	inTable := false
	appendLine := func(s string) {
		if s != "" {
			out = append(out, s)
		}
	}
	for _, line := range lines {
		// Tables are the only construct that keeps state across lines, so
		// detect entry and let mdTable decide row by row.
		if row, div, isTbl := mdTable(line, inTable); isTbl {
			inTable = true
			if !div {
				appendLine(row)
			}
			continue
		}
		inTable = false
		t := strings.TrimSpace(line)
		if t == "" || mdHr.MatchString(t) {
			continue
		}
		text := mdInline(mdBlockText(line))
		// Lines that were pure markdown syntax (empty heading, stray quote
		// marker, link/hash punctuation only) must not be spoken.
		if mdPunctOnly.MatchString(text) {
			continue
		}
		appendLine(text)
	}
	return strings.Join(out, "\n")
}

// mdTable handles a pipe line. Returns spoken text and whether the line was a
// divider; isTbl reports the line belonged to a table at all. inTable is the
// state from the previous line, so a table starts on its first pipe row.
func mdTable(line string, inTable bool) (text string, divider, isTbl bool) {
	t := strings.TrimSpace(line)
	if !strings.Contains(t, "|") {
		return "", false, false
	}
	cells := strings.Split(t, "|")
	if len(cells) > 0 && cells[0] == "" {
		cells = cells[1:]
	}
	if n := len(cells); n > 0 && cells[n-1] == "" {
		cells = cells[:n-1]
	}
	if len(cells) == 0 {
		return "", false, false
	}
	div := true
	for _, c := range cells {
		if !mdTableDiv.MatchString(strings.TrimSpace(c)) {
			div = false
			break
		}
	}
	if div {
		if !inTable {
			// Dashes-and-pipes outside a table: horizontal rule. Skip.
			return "", true, true
		}
		return "", true, true
	}
	parts := make([]string, 0, len(cells))
	for _, c := range cells {
		if s := mdInline(strings.TrimSpace(c)); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", "), false, true
}

// mdBlockText strips block-level containers (blockquote markers, list
// markers, checkbox syntax, headings, indented code indent) and returns the
// remaining prose.
func mdBlockText(line string) string {
	t := line
	// Indented code: content is spoken verbatim, still as words.
	if strings.HasPrefix(t, "    ") || strings.HasPrefix(t, "\t") {
		return strings.TrimLeft(t, " \t")
	}
	// Quote markers may nest; strip up to a reasonable depth.
	for i := 0; i < 6; i++ {
		s, ok := strings.CutPrefix(strings.TrimLeft(t, " \t"), ">")
		if !ok {
			break
		}
		t = strings.TrimLeft(s, " \t")
	}
	t = mdListMark.ReplaceAllString(t, "")
	t = mdCheckbox.ReplaceAllString(t, "")
	t = mdHeading.ReplaceAllString(t, "")
	return t
}

// mdInline strips inline markdown to its spoken text.
func mdInline(s string) string {
	s = mdInlineCD.ReplaceAllString(s, "$1")
	s = mdImageRe.ReplaceAllString(s, "$1")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = mdLinkRefRe.ReplaceAllString(s, "$1")
	s = mdAutolink.ReplaceAllString(s, "$1")
	s = mdDel.ReplaceAllString(s, "$1")
	s = mdBoldStar.ReplaceAllString(s, "$1")
	s = mdBoldUnder.ReplaceAllString(s, "$1")
	s = mdEmphStar.ReplaceAllString(s, "$1")
	s = mdEmphUnder.ReplaceAllString(s, "$1")
	s = mdHTMLTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	// Collapse whitespace introduced by removals so the TTS payload is clean.
	return strings.Join(strings.Fields(s), " ")
}

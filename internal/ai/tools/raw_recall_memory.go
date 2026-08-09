package tools

import (
	"fmt"
	"strings"

	"browser-server/internal/ai/memory"
)

// rawRecallMemoryFormatter turns a recall_memory result envelope into a
// compact, token-efficient text representation. It accepts the typed
// memory.RecallResult produced by the tool so JSON and raw rendering share one
// source of truth.
//
// Output format (normal mode):
//
//	QUERY <q> | nodes=N/M truncated=true
//	NODE <id> [kind] score=S parent=<id> tags=a,b
//	  TITLE <title>
//	  SUMMARY <summary>
//	  BODY <body>            (only when detail=full)
//	EDGE <from> <rel> <to> [note]
//	HINT <hint>
//
// Synthesize mode:
//
//	QUERY <q> | SYNTH conf=0.9
//	  ANSWER <answer>
//	  SOURCE <id>
//	  GAP <gap>
//
// The header reflects how recall was invoked: QUERY for free-text search,
// BROWSE for anchor traversal, IDS for direct fetches. Blank fields are
// omitted. Embedded newlines are rendered as the visible glyph ⏎ so the
// one-field-per-rendered-line invariant holds.
//
// Returns (nil, false) when v is not a recall result, so the registry falls
// back to JSON safely.
func rawRecallMemoryFormatter(v any) ([]byte, bool) {
	res, ok := v.(memory.RecallResult)
	if !ok {
		return nil, false
	}

	var b strings.Builder

	if res.Synthesized != nil {
		fmt.Fprintf(&b, "QUERY %s | SYNTH conf=%g\n", res.Hint, res.Synthesized.Confidence)
		fmt.Fprintf(&b, "  ANSWER %s\n", sanitizeLine(res.Synthesized.Answer))
		for _, src := range res.Synthesized.Sources {
			fmt.Fprintf(&b, "  SOURCE %s\n", src)
		}
		for _, gap := range res.Synthesized.Gaps {
			fmt.Fprintf(&b, "  GAP %s\n", sanitizeLine(gap))
		}
		return []byte(b.String()), true
	}

	if len(res.Nodes) == 0 {
		b.WriteString("QUERY | no matches\n")
		return []byte(b.String()), true
	}

	header := fmt.Sprintf("QUERY | nodes=%d/%d", res.Returned, res.TotalMatches)
	if res.Truncated {
		header += " truncated=true"
	}
	fmt.Fprintln(&b, header)

	for _, n := range res.Nodes {
		line := fmt.Sprintf("NODE %s [%s] score=%g", n.ID, n.Kind, n.Score)
		if n.Parent != "" {
			line += fmt.Sprintf(" parent=%s", n.Parent)
		}
		if len(n.Tags) > 0 {
			line += fmt.Sprintf(" tags=%s", strings.Join(n.Tags, ","))
		}
		fmt.Fprintln(&b, line)
		fmt.Fprintf(&b, "  TITLE %s\n", sanitizeLine(n.Title))
		fmt.Fprintf(&b, "  SUMMARY %s\n", sanitizeLine(n.Summary))
		if n.Body != "" {
			fmt.Fprintf(&b, "  BODY %s\n", sanitizeLine(n.Body))
		}
	}

	for _, e := range res.Edges {
		line := fmt.Sprintf("EDGE %s %s %s", e.From, e.Rel, e.To)
		if e.Note != "" {
			line += fmt.Sprintf(" note=%s", sanitizeLine(e.Note))
		}
		fmt.Fprintln(&b, line)
	}

	if res.Hint != "" {
		fmt.Fprintf(&b, "HINT %s\n", sanitizeLine(res.Hint))
	}

	return []byte(b.String()), true
}

package tools

import (
	"fmt"
	"strings"
)

// rawSearchQuestionsFormatter renders the search_questions result envelope
// (a map[string]any produced by fitSearchEnvelope) as compact plain text so
// the LLM keeps everything it reasons over — ids, question text, answers,
// ranking signal, and navigation info — while dropping JSON envelope noise
// (option ids, per-option index fields, braces, quotes).
//
// Output format:
//
//	QUERY q=<q> mode=<random|search> page=<p> size=<n> total=<t> has_more=true truncated=true
//	Q #<id> [<type>/<difficulty>] tags=a,b subject=X topic=Y sub_topic=Z score=<s>
//	  QUESTION <text>
//	  OPTION <n> <text>      (choice types; [*] marks a correct option, n = option index)
//	  ANSWER <text>          (input type; the expected answer text)
//	  CHRONO <n> <text>      (chronology type, in correct order; n = shown order)
//	HINT <hint>              (only when the LLM still has work to do)
//
// Rules:
//   - One field per rendered line; embedded newlines collapse to the visible
//     ⏎ glyph via sanitizeLine so the one-tool-line-per-rendered-line
//     invariant holds.
//   - Blank/empty fields are omitted entirely (never rendered as "field=").
//   - Truncation is never silent: has_more and truncated appear as header
//     flags and imply a trailing HINT line.
//   - The LLM references a question as #<id> when calling manage_question.
//
// Returns (nil, false) when v is not the expected envelope, so the registry
// falls back to JSON safely.
func rawSearchQuestionsFormatter(v any) ([]byte, bool) {
	env, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	results, ok := env["results"].([]map[string]any)
	if !ok {
		return nil, false
	}

	var b strings.Builder

	// Header line: the full navigation/ranking context in one line.
	header := "QUERY"
	if q, ok := env["query"].(string); ok && q != "" {
		header += " q=" + sanitizeLine(q)
	}
	if rand, ok := env["random"].(bool); ok && rand {
		header += " mode=random"
	} else {
		header += " mode=search"
	}
	header += fmt.Sprintf(" page=%d size=%d total=%d",
		rawInt(env, "page"), rawInt(env, "page_size"), rawInt(env, "total"))
	if env["has_more"] == true {
		header += " has_more=true"
	}
	if env["truncated"] == true {
		header += " truncated=true"
	}
	fmt.Fprintln(&b, header)

	if len(results) == 0 {
		b.WriteString("no matches\n")
		return []byte(b.String()), true
	}

	for _, r := range results {
		line := fmt.Sprintf("Q #%d", rawInt(r, "id"))
		if t, ok := r["type"].(string); ok && t != "" {
			line += " [" + t
			if d, ok := r["difficulty"].(string); ok && d != "" {
				line += "/" + d
			}
			line += "]"
		}
		if tags := rawStrings(r["tags"]); len(tags) > 0 {
			line += " tags=" + strings.Join(tags, ",")
		}
		if s, ok := r["subject"].(string); ok && s != "" {
			line += " subject=" + sanitizeLine(s)
		}
		if s, ok := r["topic"].(string); ok && s != "" {
			line += " topic=" + sanitizeLine(s)
		}
		if s, ok := r["sub_topic"].(string); ok && s != "" {
			line += " sub_topic=" + sanitizeLine(s)
		}
		if sc, ok := r["score"].(float64); ok && sc > 0 {
			line += fmt.Sprintf(" score=%g", sc)
		}
		fmt.Fprintln(&b, line)

		if q, ok := r["question"].(string); ok && q != "" {
			fmt.Fprintf(&b, "  QUESTION %s\n", sanitizeLine(q))
		}

		// Type-specific payload, mirroring quiz.SearchHitMap.
		switch r["type"] {
		case "single_choice", "multiple_choice":
			// correct_answers holds option indexes; options are
			// {id, index, text, correct} objects. Rendered with [*] marks so
			// the correct answer survives the JSON→text projection.
			correct := map[int]bool{}
			if ca, ok := r["correct_answers"].([]any); ok {
				for _, a := range ca {
					switch n := a.(type) {
					case int:
						correct[n] = true
					case float64:
						correct[int(n)] = true
					}
				}
			}
			if opts, ok := r["options"].([]any); ok {
				for _, o := range opts {
					opt, ok := o.(map[string]any)
					if !ok {
						continue
					}
					idx := rawInt(opt, "index")
					mark := ""
					if correct[idx] {
						mark = "[*] "
					}
					fmt.Fprintf(&b, "  OPTION %d %s%s\n", idx, mark,
						sanitizeLine(rawMapString(opt, "text")))
				}
			}

		case "input":
			fmt.Fprintf(&b, "  ANSWER %s\n", sanitizeLine(rawMapString(r, "expected_answer")))

		case "chronology":
			// ChronologyItems() yields {id, index, text, correct_order}
			// objects already sorted by correct order; index n is the
			// shuffled shown order kept so the LLM can report both.
			if items, ok := r["chronology_items"].([]any); ok {
				for _, it := range items {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					fmt.Fprintf(&b, "  CHRONO %d %s\n",
						rawInt(m, "index"), sanitizeLine(rawMapString(m, "text")))
				}
			}
		}
	}

	// Navigation hints — only when the LLM still has work to do.
	switch {
	case env["has_more"] == true:
		fmt.Fprintf(&b, "HINT more results available; call again with page=%d\n", rawInt(env, "page")+1)
	case env["truncated"] == true:
		b.WriteString("HINT results truncated by candidate cap or output budget; refine query or filters\n")
	}

	return []byte(b.String()), true
}

// rawInt reads an integer-ish value from a map regardless of whether the
// envelope was produced in-process (int) or decoded from JSON (float64), so
// the formatter survives a JSON round-trip through the registry.
func rawInt(m map[string]any, key string) int {
	switch n := m[key].(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// rawMapString is the string counterpart of rawInt.
func rawMapString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

// rawStrings extracts a []string from a map value that may be []string
// (in-process) or []any (after a JSON round-trip).
func rawStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

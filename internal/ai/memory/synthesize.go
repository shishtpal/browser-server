package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SynthesisResult is the object returned by recall_memory synthesize=true. It
// is the "librarian" sub-agent's distilled answer, replacing the raw graph
// JSON so the parent model does not have to parse and reason over fragments.
type SynthesisResult struct {
	Synthesized bool     `json:"synthesized"`
	Answer      string   `json:"answer"`
	Confidence  float64  `json:"confidence"`
	Sources     []string `json:"sources"`
	Gaps        []string `json:"gaps"`
}

var fencedJSONRe = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

// synthesize asks the cheap librarian model to answer the query from the
// retrieved fragments. On a non-JSON or unparseable response the whole call
// fails so the caller can fall back to raw graph JSON.
func (s *Store) synthesize(ctx context.Context, query string, nodes []RecallNode, edges []Edge) (SynthesisResult, error) {
	prompt := buildSynthesizePrompt(query, nodes, edges)
	req := CompletionRequest{
		System:          "You are the memory assistant for an AI agent. Answer factually and only from the provided context.",
		User:            prompt,
		Temperature:     s.cfg.Synthesizer.Temperature,
		MaxOutputTokens: s.cfg.Synthesizer.MaxOutputTokens,
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Synthesizer.TimeoutMS)*time.Millisecond)
	defer cancel()
	resp, err := s.completer.Complete(ctx, req)
	if err != nil {
		return SynthesisResult{}, err
	}
	return parseSynthesisResult(resp.Content, nodes)
}

func buildSynthesizePrompt(query string, nodes []RecallNode, edges []Edge) string {
	var sb strings.Builder
	sb.WriteString("Answer the query using <only> the provided memory fragments and their relationships.\n")
	sb.WriteString("If the fragments do not contain the answer, return {\"answer\":\"memory gap\",\"confidence\":0.0,\"gaps\":[\"specific missing facts\"]}.\n")
	sb.WriteString("Use ONLY the provided context. Never extrapolate.\n")
	sb.WriteString("For each fragment you rely on, include its id in the \"sources\" array.\n")
	sb.WriteString("Keep the answer under 150 words and actionable.\n\n")
	fmt.Fprintf(&sb, "## Query\n%s\n\n", query)
	sb.WriteString("## Memory fragments\n")
	for _, n := range nodes {
		fmt.Fprintf(&sb, "- %s (%s, %s): %s\n", n.ID, n.Kind, n.Title, n.Summary)
		if n.Body != "" {
			sb.WriteString(n.Body)
			sb.WriteString("\n")
		}
	}
	if len(edges) > 0 {
		sb.WriteString("\n## Edges (relationships)\n")
		for _, e := range edges {
			fmt.Fprintf(&sb, "- %s %s %s\n", e.From, e.Rel, e.To)
		}
	}
	sb.WriteString("\nRespond with valid JSON only:\n{\"answer\":\"...\",\"confidence\":0.0-1.0,\"gaps\":[],\"sources\":[\"mem_id\"]}\n")
	return sb.String()
}

// parseSynthesisResult extracts the JSON object from the model's response,
// tolerating markdown code fences and stray prose.
func parseSynthesisResult(content string, nodes []RecallNode) (SynthesisResult, error) {
	raw := strings.TrimSpace(content)
	var out SynthesisResult
	if m := fencedJSONRe.FindStringSubmatch(raw); len(m) == 2 {
		raw = m[1]
	}
	// Extract the outermost JSON object, tolerating leading/trailing prose.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return SynthesisResult{}, fmt.Errorf("synthesizer returned non-JSON: %v", err)
	}
	out.Synthesized = true
	if len(out.Sources) == 0 {
		// default sources to returned fragment ids
		for _, n := range nodes {
			out.Sources = append(out.Sources, n.ID)
		}
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		out.Confidence = 0
	}
	return out, nil
}

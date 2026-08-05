package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"browser-server/internal/ai/chat"
	"browser-server/internal/ai/provider"
)

// ANSI color constants. The repo has no color dependency, so a tiny local
// helper lives here. Color is disabled when NO_COLOR is set, when --json is
// on, or when stderr is not a terminal.
const (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// renderer turns chat.Service events into terminal output. Stream separation is
// deliberate: the assistant's answer goes to stdout (so `bs-ai-chat "..." >
// answer.txt` captures exactly the answer), and every trace line goes to
// stderr. In --json mode all streaming is suppressed and one object is printed
// at the end.
type renderer struct {
	jsonMode bool
	verbose  bool
	color    bool

	answer       strings.Builder // assistant answer
	reasoning    strings.Builder // accumulated reasoning (--verbose only)
	reasonHeader bool

	jsonTools []jsonToolCall
	toolIndex map[string]int
	toolStart map[string]time.Time
}

// jsonToolCall is one entry in the --json output's tool_calls array. Arguments
// and result are raw JSON when they parse, omitted otherwise.
type jsonToolCall struct {
	Name       string          `json:"name"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	Status     string          `json:"status,omitempty"`
	DurationMS int64           `json:"duration_ms,omitempty"`
}

func newRenderer(opts options) *renderer {
	return &renderer{
		jsonMode:  opts.json,
		verbose:   opts.verbose,
		color:     opts.verbose && !opts.json && isTerminal(os.Stderr) && os.Getenv("NO_COLOR") == "",
		toolIndex: map[string]int{},
		toolStart: map[string]time.Time{},
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (r *renderer) paint(code, s string) string {
	if !r.color {
		return s
	}
	return code + s + colorReset
}

// Emit implements the chat.Event callback. It never returns an error: the CLI
// is not interactive, so there is no consumer to stop the stream early.
func (r *renderer) Emit(ev chat.Event) error {
	switch ev.Type {
	case "delta":
		r.answer.WriteString(ev.Content)
		if !r.jsonMode {
			fmt.Print(ev.Content)
		}
	case "reasoning":
		if !r.verbose {
			return nil
		}
		r.reasoning.WriteString(ev.Content)
		if r.jsonMode {
			return nil
		}
		if !r.reasonHeader {
			fmt.Fprintln(os.Stderr, r.paint(colorDim, "── reasoning ──"))
			r.reasonHeader = true
		}
		fmt.Fprint(os.Stderr, r.paint(colorDim, ev.Content))
	case "tool_call":
		if ev.ToolCall == nil {
			return nil
		}
		call := ev.ToolCall
		if r.jsonMode {
			if r.verbose {
				r.startJSONTool(call)
			}
			return nil
		}
		r.toolStart[call.ID] = time.Now()
		if r.verbose {
			fmt.Fprintf(os.Stderr, "%s %s %s\n", r.paint(colorCyan, "⚙ "+call.Name), r.paint(colorDim, prettyJSON(call.Arguments)), statusLabel(r, ev.Status))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", r.paint(colorCyan, "⚙ "+call.Name))
		}
	case "tool_result":
		if ev.ToolCall == nil {
			return nil
		}
		call := ev.ToolCall
		if r.jsonMode {
			if r.verbose {
				r.finishJSONTool(call, ev)
			}
			return nil
		}
		if ev.Status != "error" && !r.verbose {
			return nil
		}
		duration := time.Since(r.toolStart[call.ID]).Round(time.Millisecond)
		if ev.Status == "error" {
			fmt.Fprintf(os.Stderr, "%s %s\n", r.paint(colorRed, "✗ "+call.Name), r.paint(colorDim, prettyResult(ev.Content)))
			return nil
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", r.paint(colorGreen, "✔ "+call.Name), r.paint(colorDim, duration.String()))
		if ev.Content != "" {
			fmt.Fprintf(os.Stderr, "%s\n", r.paint(colorDim, prettyResult(ev.Content)))
		}
	case "append_window":
		if r.verbose && !r.jsonMode && ev.Status == "open" {
			fmt.Fprintln(os.Stderr, r.paint(colorDim, "── iteration ──"))
		}
	}
	return nil
}

// Finish writes the final output: the JSON object in --json mode, otherwise a
// trailing newline so the streamed answer ends cleanly.
//
// The authoritative answer is resp.AssistantMessage.Content, not the
// accumulated deltas: across a multi-iteration tool loop the service persists
// only the final iteration's text, and it appends the tool-iteration-limit
// notice after streaming has already ended.
func (r *renderer) Finish(resp chat.SubmitResponse, providerName, modelID string) {
	final := resp.AssistantMessage.Content
	if final == "" {
		final = r.answer.String()
	}
	if r.jsonMode {
		out := struct {
			ConversationID string         `json:"conversation_id"`
			Provider       string         `json:"provider"`
			Model          string         `json:"model"`
			Response       string         `json:"response"`
			Reasoning      string         `json:"reasoning,omitempty"`
			ToolCalls      []jsonToolCall `json:"tool_calls,omitempty"`
			Usage          provider.Usage `json:"usage"`
		}{
			ConversationID: resp.ConversationID,
			Provider:       providerName,
			Model:          modelID,
			Response:       final,
			Reasoning:      r.reasoning.String(),
			ToolCalls:      r.jsonTools,
			Usage:          resp.Usage,
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: encode JSON output: %v\n", err)
			return
		}
		fmt.Println(string(b))
		return
	}
	streamed := r.answer.String()
	if tail := unstreamedTail(streamed, final); tail != "" {
		fmt.Print(tail)
		streamed += tail
	}
	if streamed != "" && !strings.HasSuffix(streamed, "\n") {
		fmt.Println()
	}
}

// unstreamedTail returns the part of the authoritative final content that was
// never emitted as a delta. The service appends text after streaming ends (the
// tool-iteration-limit notice), and in a multi-iteration run the streamed text
// holds every iteration while final holds only the last, so the overlap is
// found as the longest prefix of final that is already a suffix of streamed.
func unstreamedTail(streamed, final string) string {
	if final == "" || strings.HasSuffix(streamed, final) {
		return ""
	}
	for i := len(final); i > 0; i-- {
		if strings.HasSuffix(streamed, final[:i]) {
			return final[i:]
		}
	}
	return final
}

func (r *renderer) startJSONTool(call *provider.ToolCall) {
	var args json.RawMessage
	if call.Arguments != "" {
		args = json.RawMessage(call.Arguments)
	}
	r.toolIndex[call.ID] = len(r.jsonTools)
	r.jsonTools = append(r.jsonTools, jsonToolCall{Name: call.Name, Arguments: args, Status: "pending"})
	r.toolStart[call.ID] = time.Now()
}

func (r *renderer) finishJSONTool(call *provider.ToolCall, ev chat.Event) {
	idx, ok := r.toolIndex[call.ID]
	if !ok {
		return
	}
	r.jsonTools[idx].Status = ev.Status
	r.jsonTools[idx].DurationMS = time.Since(r.toolStart[call.ID]).Milliseconds()
	if ev.Content != "" {
		var payload struct {
			Result json.RawMessage `json:"result"`
		}
		if json.Unmarshal([]byte(ev.Content), &payload) == nil && payload.Result != nil {
			r.jsonTools[idx].Result = payload.Result
		}
	}
}

func statusLabel(r *renderer, status string) string {
	switch status {
	case "approved":
		return r.paint(colorGreen, "[approved]")
	case "pending":
		return r.paint(colorYellow, "[pending]")
	case "error":
		return r.paint(colorRed, "[error]")
	}
	return ""
}

// prettyJSON indents raw JSON for display, falling back to the raw string when
// it is not valid JSON.
func prettyJSON(raw string) string {
	if raw == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err != nil {
		return raw
	}
	return buf.String()
}

// prettyResult extracts and indents the "result" field from a stored tool
// message ({"tool":..., "args":..., "result":..., "decision":...}).
func prettyResult(content string) string {
	var payload struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal([]byte(content), &payload) != nil || payload.Result == nil {
		return content
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, payload.Result, "", "  "); err != nil {
		return string(payload.Result)
	}
	return buf.String()
}

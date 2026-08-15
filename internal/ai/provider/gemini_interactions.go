package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// GeminiInteractionsClient talks to the Gemini Interactions API
// (POST {base}/interactions, with "stream": true for SSE). It operates
// statelessly: the full conversation history is re-sent as an input step array
// on every turn, so the chat service's branching/regeneration/editing pipeline
// keeps working without previous_interaction_id chaining. store=false keeps
// Google from retaining interactions.
type GeminiInteractionsClient struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	retryAttempts int
	retryDelay    time.Duration
}

// NewGeminiInteractionsClient builds a client against baseURL (e.g.
// https://generativelanguage.googleapis.com/v1beta). Model IDs may carry the
// "models/" prefix; it is stripped before the request is built.
func NewGeminiInteractionsClient(baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration) *GeminiInteractionsClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &GeminiInteractionsClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		apiKey:        apiKey,
		retryAttempts: retryAttempts,
		retryDelay:    retryDelay,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// geminiPart is one input/output block (text, inline image, ...). Image parts
// carry base64 bytes in Data (no RFC 2397 prefix).
type geminiPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	Data     string `json:"data,omitempty"`
}

// geminiStep is an entry in the Interactions input/step timeline.
type geminiStep struct {
	Type      string          `json:"type"`
	Content   []geminiPart    `json:"content,omitempty"`
	Name      string          `json:"name,omitempty"`
	ID        string          `json:"id,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    []geminiPart    `json:"result,omitempty"`
}

type geminiTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type geminiRequest struct {
	Model             string          `json:"model"`
	Input             []geminiStep    `json:"input,omitempty"`
	SystemInstruction string          `json:"system_instruction,omitempty"`
	Tools             []geminiTool    `json:"tools,omitempty"`
	GenerationConfig  json.RawMessage `json:"generation_config,omitempty"`
	Stream            bool            `json:"stream"`
	Store             bool            `json:"store"`
}

type geminiUsage struct {
	TotalInputTokens  *int `json:"total_input_tokens"`
	TotalOutputTokens *int `json:"total_output_tokens"`
	TotalTokens       *int `json:"total_tokens"`
}

func (u geminiUsage) toUsage() Usage {
	return Usage{PromptTokens: u.TotalInputTokens, CompletionTokens: u.TotalOutputTokens, TotalTokens: u.TotalTokens}
}

func (c *GeminiInteractionsClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	payload, err := json.Marshal(c.payload(req, false))
	if err != nil {
		return ChatResponse{}, err
	}
	started := time.Now()
	for attempt := 0; ; attempt++ {
		response, requestErr := c.completeOnce(ctx, payload)
		response.Latency = time.Since(started)
		if requestErr == nil || attempt >= c.retryAttempts || !isRetryable(requestErr) {
			return response, requestErr
		}
		log.Printf("[AI] Gemini Complete request failed (attempt %d/%d): %v — retrying", attempt+1, c.retryAttempts+1, requestErr)
		if err := waitForRetry(ctx, c.retryDelay, attempt); err != nil {
			return response, classify(err, 0)
		}
	}
}

func (c *GeminiInteractionsClient) completeOnce(ctx context.Context, raw []byte) (ChatResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/interactions", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{RawRequest: raw}, classify(err, 0)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{RawRequest: raw}, classify(err, 0)
	}
	defer resp.Body.Close()

	rawResponse, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		return ChatResponse{RawRequest: raw, HTTPStatus: resp.StatusCode}, classify(readErr, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{RawRequest: raw, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, classify(fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, geminiErrorMessage(rawResponse, 256)), resp.StatusCode)
	}

	var parsed struct {
		Steps []struct {
			Type      string          `json:"type"`
			Content   []geminiPart    `json:"content"`
			Summary   []geminiPart    `json:"summary"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"steps"`
		// output is the pre-May-2026 convenience field; some revisions still
		// include it, so treat it as a fallback when steps are absent.
		Output []geminiPart `json:"output"`
		Usage  geminiUsage  `json:"usage"`
	}
	if err := json.Unmarshal(rawResponse, &parsed); err != nil {
		return ChatResponse{RawRequest: raw, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, &Error{Code: "malformed_provider_response", Status: 502, Retryable: true, Diagnostic: "failed to parse response JSON"}
	}

	out := ChatResponse{
		HTTPStatus:  resp.StatusCode,
		RawRequest:  raw,
		RawResponse: rawResponse,
		Usage:       parsed.Usage.toUsage(),
	}
	if len(parsed.Steps) == 0 && len(parsed.Output) > 0 {
		parsed.Steps = []struct {
			Type      string          `json:"type"`
			Content   []geminiPart    `json:"content"`
			Summary   []geminiPart    `json:"summary"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}{{Type: "model_output", Content: parsed.Output}}
	}
	for _, step := range parsed.Steps {
		switch step.Type {
		case "model_output":
			for _, part := range step.Content {
				out.Content += part.Text
			}
		case "thought":
			text := textParts(step.Content, step.Summary)
			if out.Reasoning != "" && text != "" {
				out.Reasoning += "\n\n"
			}
			out.Reasoning += text
		case "function_call":
			args := ""
			if err := json.Unmarshal(step.Arguments, &args); err != nil {
				args = string(step.Arguments)
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:        step.ID,
				Name:      step.Name,
				Arguments: args,
			})
		}
	}
	if len(parsed.Steps) == 0 {
		return ChatResponse{RawRequest: raw, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, &Error{Code: "malformed_provider_response", Status: 502, Retryable: true, Diagnostic: "response contained no steps"}
	}
	return out, nil
}

func (c *GeminiInteractionsClient) Stream(ctx context.Context, req ChatRequest, emit func(Event) error) (ChatResponse, error) {
	raw, err := json.Marshal(c.payload(req, true))
	if err != nil {
		return ChatResponse{}, err
	}
	started := time.Now()
	for attempt := 0; ; attempt++ {
		emitted := false
		response, requestErr := c.streamOnce(ctx, raw, func(event Event) error {
			emitted = true
			if emit != nil {
				return emit(event)
			}
			return nil
		})
		response.Latency = time.Since(started)
		if requestErr == nil || emitted || attempt >= c.retryAttempts || !isRetryable(requestErr) {
			return response, requestErr
		}
		log.Printf("[AI] Gemini Stream request failed (attempt %d/%d): %v — retrying", attempt+1, c.retryAttempts+1, requestErr)
		if err := waitForRetry(ctx, c.retryDelay, attempt); err != nil {
			return response, classify(err, 0)
		}
	}
}

func (c *GeminiInteractionsClient) streamOnce(ctx context.Context, raw []byte, emit func(Event) error) (ChatResponse, error) {
	h, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/interactions", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, classify(err, 0)
	}
	h.Header.Set("Content-Type", "application/json")
	h.Header.Set("x-goog-api-key", c.apiKey)
	start := time.Now()
	res, err := c.httpClient.Do(h)
	if err != nil {
		return ChatResponse{RawRequest: raw, Latency: time.Since(start)}, classify(err, 0)
	}
	defer res.Body.Close()
	out := ChatResponse{RawRequest: raw, HTTPStatus: res.StatusCode}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		out.RawResponse = body
		return out, classify(fmt.Errorf("upstream HTTP %d: %s", res.StatusCode, geminiErrorMessage(body, 256)), res.StatusCode)
	}

	// The Gemini Interactions SSE protocol nests the incremental payload in a
	// top-level "delta" object (referenced by "index"), while "step" only
	// appears on step.start to describe the step at that index (its type and,
	// for function calls, its name + initial arguments). Deltas therefore live
	// at ev.Delta, never at ev.Step.Delta.
	type event struct {
		EventType string `json:"event_type"`
		Index     int    `json:"index"`
		Step      *struct {
			Type      string          `json:"type"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"step"`
		Delta *struct {
			Type             string `json:"type"`
			Text             string `json:"text"`
			Thinking         string `json:"thinking"`
			FunctionName     string `json:"function_name"`
			PartialArguments string `json:"arguments"`
			// thought_summary deltas carry the visible reasoning text nested
			// in a content object rather than in the flat "text" field.
			Content *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"delta"`
		Interaction *struct {
			ID     string      `json:"id"`
			Status string      `json:"status"`
			Usage  geminiUsage `json:"usage"`
		} `json:"interaction"`
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	calls := map[int]*ToolCall{}
	// stepNameByIndex records the function name announced in step.start so a
	// later arguments_delta (which may omit the name) can be attributed.
	stepNameByIndex := map[int]string{}
	// Reasoning accumulates the model's thinking (thought_summary deltas)
	// across the stream so it can be persisted on the assistant message.
	var reasoning strings.Builder
	sawTerminal := false
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var ev event
		if json.Unmarshal([]byte(data), &ev) != nil {
			continue
		}
		switch ev.EventType {
		case "step.start":
			// Announces the step type (and, for function calls, the name and
			// any already-present arguments) for the index referenced by the
			// subsequent step.delta events.
			if ev.Step == nil {
				continue
			}
			if ev.Step.Type == "function_call" {
				v := &ToolCall{ID: ev.Step.ID, Name: ev.Step.Name}
				// Seed only when arguments are non-empty; an empty "{}" from
				// step.start would otherwise corrupt the arguments_delta
				// fragments that accumulate on this call below.
				if a := strings.TrimSpace(string(ev.Step.Arguments)); a != "" && a != "{}" && a != "null" {
					v.Arguments = a
				}
				calls[ev.Index] = v
			}
			if ev.Step.Name != "" {
				stepNameByIndex[ev.Index] = ev.Step.Name
			}
		case "step.delta":
			delta := ev.Delta
			if delta == nil {
				continue
			}
			switch delta.Type {
			case "text":
				if delta.Text != "" {
					out.Content += delta.Text
					if err := emit(Event{Type: "text_delta", Text: delta.Text}); err != nil {
						return out, err
					}
				}
			case "arguments_delta":
				v := calls[ev.Index]
				if v == nil {
					v = &ToolCall{}
					calls[ev.Index] = v
				}
				if delta.FunctionName != "" {
					v.Name = delta.FunctionName
				} else if stepNameByIndex[ev.Index] != "" {
					v.Name = stepNameByIndex[ev.Index]
				}
				v.Arguments += delta.PartialArguments
			case "thought_summary":
				// Visible reasoning text. It is nested in delta.content.text
				// for thought_summary deltas, falling back to the flat text
				// field when the nested form is absent.
				text := ""
				if delta.Content != nil {
					text = delta.Content.Text
				}
				if text == "" {
					text = delta.Text
				}
				if text != "" {
					reasoning.WriteString(text)
					if err := emit(Event{Type: "reasoning_delta", Text: text}); err != nil {
						return out, err
					}
				}
			default:
				// thought_signature and other non-textual deltas are
				// received-only signals with no user-visible content.
			}
		case "step.stop":
			// Marks the end of a step; nothing to extract.
		case "interaction.status_update":
			// Function-calling turns may end with status "requires_action"
			// (the model has emitted its function_call steps and is waiting
			// for results) before interaction.completed arrives.
			if ev.Status == "completed" || ev.Status == "requires_action" {
				sawTerminal = true
			}
		case "interaction.completed":
			if ev.Interaction != nil {
				out.Usage = ev.Interaction.Usage.toUsage()
			}
			sawTerminal = true
		case "error":
			msg := "Gemini stream error"
			if ev.Error != nil && ev.Error.Message != "" {
				msg = ev.Error.Message
			}
			return out, classify(fmt.Errorf("provider error: %s", msg), 502)
		}
		if sawTerminal {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return out, classify(err, res.StatusCode)
	}
	if !sawTerminal {
		return out, &Error{Code: "malformed_provider_stream", Status: 502, Retryable: true, Diagnostic: "stream ended without terminal event"}
	}
	out.Reasoning = reasoning.String()
	// Step indices are global across the whole response timeline (thoughts,
	// function calls, ...), so the tool-call keys do not necessarily form a
	// dense 0..n-1 sequence. Iterate up to the maximum observed index rather
	// than len(calls); otherwise any call whose index >= len(calls) is
	// silently dropped and the turn would end having executed no tools.
	maxIndex := -1
	for idx := range calls {
		if idx > maxIndex {
			maxIndex = idx
		}
	}
	for i := 0; i <= maxIndex; i++ {
		if calls[i] != nil {
			out.ToolCalls = append(out.ToolCalls, *calls[i])
			_ = emit(Event{Type: "tool_call", ToolCall: calls[i]})
		}
	}
	out.Latency = time.Since(start)
	_ = emit(Event{Type: "done", Usage: out.Usage})
	return out, nil
}

func (c *GeminiInteractionsClient) payload(req ChatRequest, stream bool) geminiRequest {
	p := geminiRequest{
		Model:  strings.TrimPrefix(req.Model, "models/"),
		Stream: stream,
		Store:  false,
	}
	// generation_config mirrors the reference curl: max_output_tokens, top_p,
	// and thinking_level alongside temperature.
	gc := map[string]any{
		"max_output_tokens": req.MaxOutputTokens,
		"temperature":       req.Temperature,
		"top_p":             0.95,
		"thinking_level":    "medium",
	}
	p.GenerationConfig, _ = json.Marshal(gc)

	callNameByID := map[string]string{}
	for _, message := range req.Messages {
		switch message.Role {
		case "system":
			if p.SystemInstruction == "" {
				p.SystemInstruction = message.Content
			} else if message.Content != "" {
				p.SystemInstruction += "\n\n" + message.Content
			}
		case "user":
			parts := textAndImageParts(message)
			if len(parts) > 0 {
				p.Input = append(p.Input, geminiStep{Type: "user_input", Content: parts})
			}
		case "assistant":
			if len(message.ToolCalls) > 0 {
				if message.Content != "" {
					p.Input = append(p.Input, geminiStep{Type: "model_output", Content: []geminiPart{{Type: "text", Text: message.Content}}})
				}
				for _, call := range message.ToolCalls {
					callNameByID[call.ID] = call.Name
					p.Input = append(p.Input, geminiStep{Type: "function_call", ID: call.ID, Name: call.Name, Arguments: functionCallArgumentsJSON(call.Arguments)})
				}
			} else if message.Content != "" {
				p.Input = append(p.Input, geminiStep{Type: "model_output", Content: []geminiPart{{Type: "text", Text: message.Content}}})
			}
		case "tool":
			if message.ToolCallID == "" {
				continue
			}
			p.Input = append(p.Input, geminiStep{
				Type:   "function_result",
				CallID: message.ToolCallID,
				Name:   callNameByID[message.ToolCallID],
				Result: []geminiPart{{Type: "text", Text: message.Content}},
			})
		}
	}

	for _, t := range req.Tools {
		p.Tools = append(p.Tools, geminiTool{Type: "function", Name: t.Name, Description: t.Description, Parameters: t.Parameters})
	}
	if req.GoogleSearch {
		p.Tools = append(p.Tools, geminiTool{Type: "google_search"})
	}
	return p
}

// functionCallArgumentsJSON renders a tool call's raw arguments for the
// Interactions request. The API transports function arguments as a
// JSON-encoded *string* (mirroring the arguments_delta fragments streamed over
// SSE), not as an inline object, so the wire value is e.g. "{\"tz\":\"UTC\"}"
// rather than {"tz":"UTC"}. Empty or "null" arguments become the string "{}"
// (a no-arg tool call).
func functionCallArgumentsJSON(raw string) json.RawMessage {
	if raw == "" || raw == "null" {
		raw = "{}"
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		// json.Marshal on a Go string cannot actually fail; keep a safe
		// fallback so a malformed tool call degrades to a no-arg call.
		return json.RawMessage(`"{}"`)
	}
	return json.RawMessage(encoded)
}

// textAndImageParts converts a provider message into Gemini input parts.
// ImageParts arrive as RFC 2397 data URLs and are decoded into Gemini's
// {type:image, mime_type, data} form (raw base64, no prefix).
func textAndImageParts(message Message) []geminiPart {
	var parts []geminiPart
	if message.Content != "" {
		parts = append(parts, geminiPart{Type: "text", Text: message.Content})
	}
	for _, part := range message.ImageParts {
		mimeType, data := splitDataURL(part.DataURL)
		if mimeType == "" || data == "" {
			continue
		}
		parts = append(parts, geminiPart{Type: "image", MIMEType: mimeType, Data: data})
	}
	return parts
}

func splitDataURL(dataURL string) (mimeType, data string) {
	rest, ok := strings.CutPrefix(dataURL, "data:")
	if !ok {
		return "", ""
	}
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return "", ""
	}
	meta := rest[:comma]
	if strings.HasSuffix(meta, ";base64") {
		return strings.TrimSuffix(meta, ";base64"), rest[comma+1:]
	}
	return meta, rest[comma+1:]
}

func textParts(content, summary []geminiPart) string {
	var b strings.Builder
	for _, lists := range [][]geminiPart{content, summary} {
		for _, part := range lists {
			if part.Type == "text" && part.Text != "" {
				b.WriteString(part.Text)
			}
		}
	}
	return b.String()
}

// geminiErrorMessage pulls the human-readable reason out of a Google API error
// envelope so operators see "API key not valid" instead of a bare 400.
func geminiErrorMessage(b []byte, maxLen int) string {
	var v struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &v) == nil && v.Error.Message != "" {
		if v.Error.Status != "" {
			return v.Error.Status + ": " + v.Error.Message
		}
		return v.Error.Message
	}
	return truncateBody(b, maxLen)
}

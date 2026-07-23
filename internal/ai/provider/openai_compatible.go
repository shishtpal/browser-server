package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatibleClient struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	retryAttempts int
	retryDelay    time.Duration
}

func NewOpenAICompatibleClient(baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		apiKey:        apiKey,
		retryAttempts: retryAttempts,
		retryDelay:    retryDelay,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type chatCompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []wireMessage    `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
	Tools       []map[string]any `json:"tools,omitempty"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
}

type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     *int `json:"prompt_tokens"`
		CompletionTokens *int `json:"completion_tokens"`
		TotalTokens      *int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (c *OpenAICompatibleClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	payload := c.payload(req, false)
	rawRequest, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, err
	}
	started := time.Now()
	for attempt := 0; ; attempt++ {
		response, requestErr := c.completeOnce(ctx, rawRequest)
		response.Latency = time.Since(started)
		if requestErr == nil || attempt >= c.retryAttempts || !isRetryable(requestErr) {
			return response, requestErr
		}
		delay := c.retryDelay * time.Duration(1<<uint(attempt))
		log.Printf("[AI] Complete request failed (attempt %d/%d): %v — retrying in %v", attempt+1, c.retryAttempts+1, requestErr, delay)
		if err := waitForRetry(ctx, c.retryDelay, attempt); err != nil {
			return response, classify(err, 0)
		}
	}
}

func (c *OpenAICompatibleClient) completeOnce(ctx context.Context, rawRequest []byte) (ChatResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(rawRequest))
	if err != nil {
		return ChatResponse{RawRequest: rawRequest}, classify(err, 0)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "http://localhost")
	httpReq.Header.Set("X-Title", "browser-server")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{RawRequest: rawRequest}, classify(err, 0)
	}
	defer resp.Body.Close()

	rawResponse, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		return ChatResponse{RawRequest: rawRequest, HTTPStatus: resp.StatusCode}, classify(readErr, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, classify(fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, truncateBody(rawResponse, 256)), resp.StatusCode)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(rawResponse, &parsed); err != nil {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, &Error{Code: "malformed_provider_response", Status: 502, Retryable: true, Diagnostic: "failed to parse response JSON"}
	}
	// Some providers (e.g. OpenRouter) return HTTP 200 with an error body
	if parsed.Error != nil && parsed.Error.Message != "" {
		errStatus := classifyErrorBody(parsed.Error.Code)
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, classify(fmt.Errorf("provider error: %s", parsed.Error.Message), errStatus)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode}, &Error{Code: "malformed_provider_response", Status: 502, Retryable: true, Diagnostic: "response contained no choices"}
	}

	result := ChatResponse{
		Content:     parsed.Choices[0].Message.Content,
		Usage:       Usage(parsed.Usage),
		HTTPStatus:  resp.StatusCode,
		RawRequest:  rawRequest,
		RawResponse: rawResponse,
	}
	for _, call := range parsed.Choices[0].Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
	}
	return result, nil
}

func (c *OpenAICompatibleClient) payload(req ChatRequest, stream bool) chatCompletionRequest {
	p := chatCompletionRequest{Model: req.Model, Temperature: req.Temperature, MaxTokens: req.MaxOutputTokens, Stream: stream}
	for _, message := range req.Messages {
		wire := wireMessage{Role: message.Role, Content: message.Content, ToolCallID: message.ToolCallID}
		for _, call := range message.ToolCalls {
			item := wireToolCall{ID: call.ID, Type: "function"}
			item.Function.Name = call.Name
			item.Function.Arguments = call.Arguments
			wire.ToolCalls = append(wire.ToolCalls, item)
		}
		p.Messages = append(p.Messages, wire)
	}
	for _, t := range req.Tools {
		p.Tools = append(p.Tools, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": json.RawMessage(t.Parameters)}})
	}
	return p
}

func (c *OpenAICompatibleClient) Stream(ctx context.Context, req ChatRequest, emit func(Event) error) (ChatResponse, error) {
	raw, _ := json.Marshal(c.payload(req, true))
	started := time.Now()
	for attempt := 0; ; attempt++ {
		emitted := false
		response, requestErr := c.streamOnce(ctx, raw, func(event Event) error {
			emitted = true
			return emit(event)
		})
		response.Latency = time.Since(started)
		if requestErr == nil || emitted || attempt >= c.retryAttempts || !isRetryable(requestErr) {
			return response, requestErr
		}
		delay := c.retryDelay * time.Duration(1<<uint(attempt))
		log.Printf("[AI] Stream request failed (attempt %d/%d): %v — retrying in %v", attempt+1, c.retryAttempts+1, requestErr, delay)
		if err := waitForRetry(ctx, c.retryDelay, attempt); err != nil {
			return response, classify(err, 0)
		}
	}
}

func (c *OpenAICompatibleClient) streamOnce(ctx context.Context, raw []byte, emit func(Event) error) (ChatResponse, error) {
	h, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, classify(err, 0)
	}
	h.Header.Set("Content-Type", "application/json")
	h.Header.Set("Authorization", "Bearer "+c.apiKey)
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
		return out, classify(fmt.Errorf("upstream HTTP %d", res.StatusCode), res.StatusCode)
	}
	type chunk struct {
		Choices []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	calls := map[int]*ToolCall{}
	sawTerminal := false
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			sawTerminal = true
			break
		}
		var ch chunk
		if json.Unmarshal([]byte(data), &ch) != nil {
			return out, &Error{Code: "malformed_provider_stream", Status: 502, Retryable: true, Diagnostic: "failed to parse stream chunk"}
		}
		if ch.Usage.TotalTokens != nil {
			out.Usage = ch.Usage
			_ = emit(Event{Type: "usage", Usage: ch.Usage})
		}
		for _, choice := range ch.Choices {
			if choice.Delta.Content != "" {
				out.Content += choice.Delta.Content
				if err := emit(Event{Type: "text_delta", Text: choice.Delta.Content}); err != nil {
					return out, err
				}
			}
			for _, tc := range choice.Delta.ToolCalls {
				v := calls[tc.Index]
				if v == nil {
					v = &ToolCall{}
					calls[tc.Index] = v
				}
				if tc.ID != "" {
					v.ID = tc.ID
				}
				v.Name += tc.Function.Name
				v.Arguments += tc.Function.Arguments
			}
			// OpenAI-compatible providers are allowed to terminate a completion
			// with finish_reason and omit or delay the optional [DONE] sentinel.
			// Returning here is especially important for tool calls: the chat
			// service cannot register the pending approval until Stream returns.
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				sawTerminal = true
			}
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
	for i := 0; i < len(calls); i++ {
		if calls[i] != nil {
			out.ToolCalls = append(out.ToolCalls, *calls[i])
			_ = emit(Event{Type: "tool_call", ToolCall: calls[i]})
		}
	}
	out.Latency = time.Since(start)
	_ = emit(Event{Type: "done", Usage: out.Usage})
	return out, nil
}

func classify(err error, status int) error {
	code := "provider_error"
	retry := status == 0 || status == 429 || status >= 500
	safeStatus := 502
	if status == 429 {
		code = "rate_limited"
		safeStatus = 429
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code = "provider_timeout"
		safeStatus = 504
		retry = true
	}
	if errors.Is(err, context.Canceled) {
		retry = false
	}
	return &Error{Code: code, Status: safeStatus, Retryable: retry, Diagnostic: err.Error()}
}

// classifyErrorBody maps a provider's inline error code to a pseudo-HTTP status
// so that classify() can determine retryability. Providers like OpenRouter can
// return HTTP 200 with an error body; the inline code tells us the real failure.
func classifyErrorBody(code any) int {
	switch v := code.(type) {
	case float64:
		return int(v)
	case string:
		switch v {
		case "rate_limit_exceeded", "429":
			return 429
		case "server_error", "503", "502":
			return 503
		case "timeout", "504":
			return 504
		}
	}
	// Treat unknown error codes as transient (server-side) so they get retried
	return 503
}

func isRetryable(err error) bool {
	var providerErr *Error
	return errors.As(err, &providerErr) && providerErr.Retryable
}

func waitForRetry(ctx context.Context, baseDelay time.Duration, attempt int) error {
	// Exponential backoff: delay * 2^attempt, capped at 60s
	delay := baseDelay * time.Duration(1<<uint(attempt))
	const maxDelay = 60 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// truncateBody returns up to maxLen bytes of the response body for diagnostics.
func truncateBody(body []byte, maxLen int) string {
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "..."
}

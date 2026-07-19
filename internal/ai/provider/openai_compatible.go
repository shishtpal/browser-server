package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAICompatibleClient(baseURL, apiKey string, timeout time.Duration) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(rawRequest))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "http://localhost")
	httpReq.Header.Set("X-Title", "browser-server")

	started := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	latency := time.Since(started)
	if err != nil {
		return ChatResponse{RawRequest: rawRequest, Latency: latency}, classify(err, 0)
	}
	defer resp.Body.Close()

	rawResponse, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		return ChatResponse{RawRequest: rawRequest, HTTPStatus: resp.StatusCode, Latency: latency}, classify(readErr, resp.StatusCode)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(rawResponse, &parsed); err != nil {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode, Latency: latency}, &Error{Code: "malformed_provider_response", Status: 502}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode, Latency: latency}, classify(fmt.Errorf("upstream HTTP %d", resp.StatusCode), resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{RawRequest: rawRequest, RawResponse: rawResponse, HTTPStatus: resp.StatusCode, Latency: latency}, &Error{Code: "malformed_provider_response", Status: 502}
	}

	result := ChatResponse{
		Content:     parsed.Choices[0].Message.Content,
		Usage:       Usage(parsed.Usage),
		HTTPStatus:  resp.StatusCode,
		Latency:     latency,
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
			return out, &Error{Code: "malformed_provider_stream", Status: 502}
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
		return out, &Error{Code: "malformed_provider_stream", Status: 502}
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
	retry := status == 429 || status >= 500
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
	return &Error{Code: code, Status: safeStatus, Retryable: retry, Diagnostic: err.Error()}
}

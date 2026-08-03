package mcp

import (
	"encoding/json"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The generic tool registry applies the operator-configured final output
// limit. This preflight cap only prevents an MCP result from consuming an
// unbounded amount of memory before it reaches that shared policy gate.
const maxResultBytes = 512 * 1024

func normalizeResult(result *sdk.CallToolResult) (any, error) {
	if result == nil { return nil, errors.New("MCP tool returned no result") }
	envelope := map[string]any{"content": normalizeContent(result.Content)}
	if result.StructuredContent != nil { envelope["structured_content"] = result.StructuredContent }
	encoded, err := json.Marshal(envelope)
	if err != nil { return nil, errors.New("MCP tool returned an invalid result") }
	if len(encoded) > maxResultBytes { return nil, errors.New("MCP tool result exceeds 512 KiB") }
	if result.IsError {
		message := "MCP tool reported an error"
		for _, c := range result.Content { if text, ok := c.(*sdk.TextContent); ok && text.Text != "" { message = text.Text; break } }
		if len(message) > 1024 { message = message[:1024] }
		return nil, errors.New(message)
	}
	return envelope, nil
}

func normalizeContent(content []sdk.Content) []any {
	out := make([]any, 0, len(content))
	for _, item := range content {
		b, err := json.Marshal(item); if err != nil { continue }
		var value map[string]any; if json.Unmarshal(b, &value) != nil { continue }
		// Binary content is intentionally represented by metadata only.
		if typ, _ := value["type"].(string); typ == "image" || typ == "audio" { delete(value, "data") }
		out = append(out, value)
	}
	return out
}

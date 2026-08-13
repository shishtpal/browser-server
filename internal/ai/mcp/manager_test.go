package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExecuteWithInMemoryTransport(t *testing.T) {
	ctx := context.Background()
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	server := sdk.NewServer(&sdk.Implementation{Name: "test-server", Version: "1"}, nil)
	sdk.AddTool(server, &sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}},
		func(_ context.Context, _ *sdk.CallToolRequest, args map[string]any) (*sdk.CallToolResult, any, error) {
			return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: args["value"].(string)}}}, nil, nil
		})
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{
		routes:  map[string]route{"mcp_test_echo": {server: "test", original: "echo", session: clientSession, timeout: time.Second}},
		servers: []*managedServer{{session: clientSession}},
	}
	args, _ := json.Marshal(map[string]any{"value": "hello"})
	got, err := m.Execute(ctx, "mcp_test_echo", args)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got)
	if !strings.Contains(string(b), "hello") {
		t.Fatalf("result=%s", b)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderSafeNamesAreStableAndDistinct(t *testing.T) {
	a := publicName("ServerName", strings.Repeat("tool!", 30))
	b := publicName("ServerName", strings.Repeat("tool?", 30))
	if a == b || len(a) > 64 || len(b) > 64 {
		t.Fatalf("invalid names %q %q", a, b)
	}
	if a != publicName("ServerName", strings.Repeat("tool!", 30)) {
		t.Fatal("name is unstable")
	}
}

func TestNormalizeResultErrorAndBinaryMetadata(t *testing.T) {
	_, err := normalizeResult(&sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: "tool failed"}}})
	if err == nil || err.Error() != "tool failed" {
		t.Fatalf("error=%v", err)
	}
	got, err := normalizeResult(&sdk.CallToolResult{Content: []sdk.Content{&sdk.ImageContent{MIMEType: "image/png", Data: []byte("secret binary")}}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got)
	if strings.Contains(string(b), "secret") || !strings.Contains(string(b), "image/png") {
		t.Fatalf("result=%s", b)
	}
}

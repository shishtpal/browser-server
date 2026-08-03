package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testJSON(value string) string {
	return strings.ReplaceAll(value, "'", string(rune(34)))
}

func TestLoadOptionalAndDefaults(t *testing.T) {
	t.Setenv("BS_AI_MCP_PATH", "")
	c, err := Load(t.TempDir())
	if err != nil || c.Configured || len(c.MCPServers) != 0 {
		t.Fatalf("Load absent: config=%+v err=%v", c, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, configName)
	content := testJSON(`{'mcpServers':{'local':{'command':'tool','cwd':'work'}}}`)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil { t.Fatal(err) }
	c, err = Load(dir)
	if err != nil { t.Fatal(err) }
	s := c.MCPServers["local"]
	if !c.Configured || s.ConnectTimeoutSeconds != 15 || s.CallTimeoutSeconds != 60 { t.Fatalf("defaults not applied: %+v", s) }
	if s.Cwd != filepath.Join(dir, "work") { t.Fatalf("cwd=%q", s.Cwd) }
}

func TestLoadStrictAndSecretResolution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configName)
	t.Setenv("MCP_TEST_SECRET", "super-secret-value")
	content := testJSON(`{'mcpServers':{'remote':{'url':'https://example.test/mcp?token=hidden','headers':{'Authorization':'env:MCP_TEST_SECRET'}}}}`)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil { t.Fatal(err) }
	c, err := Load(dir)
	if err != nil { t.Fatal(err) }
	if got := c.MCPServers["remote"].Headers["Authorization"]; got != "super-secret-value" { t.Fatalf("resolved header=%q", got) }
	if err := os.WriteFile(path, []byte(testJSON(`{'mcpServers':{},'unknown':true}`)), 0600); err != nil { t.Fatal(err) }
	if _, err := Load(dir); err == nil { t.Fatal("unknown field was accepted") }
}

func TestLoadMissingEnvironmentDoesNotLeakValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MCP_EMPTY_SECRET", "")
	content := testJSON(`{'mcpServers':{'remote':{'url':'https://example.test','headers':{'X-Key':'env:MCP_EMPTY_SECRET'}}}}`)
	if err := os.WriteFile(filepath.Join(dir, configName), []byte(content), 0600); err != nil { t.Fatal(err) }
	_, err := Load(dir)
	if err == nil || strings.Contains(err.Error(), "super-secret") { t.Fatalf("unexpected error: %v", err) }
}

func TestRemoteURLPolicy(t *testing.T) {
	for _, good := range []string{"https://example.test/mcp", "http://localhost:8080/mcp", "http://127.0.0.1/mcp"} {
		if err := validateRemoteURL(good); err != nil { t.Errorf("%q: %v", good, err) }
	}
	for _, bad := range []string{"http://example.test/mcp", "https://user:pass@example.test/mcp"} {
		if err := validateRemoteURL(bad); err == nil { t.Errorf("accepted %q", bad) }
	}
}

func TestLoadRejectsInvalidServerSettings(t *testing.T) {
	tests := []string{
		`{'mcpServers':{'bad name':{'command':'tool'}}}`,
		`{'mcpServers':{'both':{'command':'tool','url':'https://example.test'}}}`,
		`{'mcpServers':{'remote':{'url':'http://example.test'}}}`,
		`{'mcpServers':{'remote':{'url':'https://example.test','env':{'KEY':'value'}}}}`,
		`{'mcpServers':{'local':{'command':'tool','headers':{'X-Key':'value'}}}}`,
		`{'mcpServers':{'local':{'command':'tool','allowed_tools':['search','search']}}}`,
		`{'mcpServers':{'local':{'command':'tool','connect_timeout_seconds':121}}}`,
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			content := testJSON(input)
			if err := os.WriteFile(filepath.Join(dir, configName), []byte(content), 0600); err != nil { t.Fatal(err) }
			if _, err := Load(dir); err == nil { t.Fatalf("accepted invalid config %s", content) }
		})
	}
}

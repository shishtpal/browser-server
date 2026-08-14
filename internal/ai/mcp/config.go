// Package mcp owns configuration and connections to external MCP servers.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const configName = "bs-ai-mcp.json"

var serverNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// Config is the validated MCP configuration. An absent file produces an
// unconfigured, empty Config rather than an error.
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
	Configured bool                    `json:"-"`
	path       string
}

// ServerConfig describes one stdio or Streamable HTTP MCP server.
type ServerConfig struct {
	Enabled               *bool             `json:"enabled,omitempty"`
	Command               string            `json:"command,omitempty"`
	Args                  []string          `json:"args,omitempty"`
	Cwd                   string            `json:"cwd,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	URL                   string            `json:"url,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	AllowedTools          []string          `json:"allowed_tools,omitempty"`
	ConnectTimeoutSeconds int               `json:"connect_timeout_seconds,omitempty"`
	CallTimeoutSeconds    int               `json:"call_timeout_seconds,omitempty"`
}

func (s ServerConfig) enabled() bool { return s.Enabled == nil || *s.Enabled }

// Load reads BS_AI_MCP_PATH, or bs-ai-mcp.json in configDir. Relative override
// paths are interpreted from the process working directory.
func Load(configDir string) (*Config, error) {
	path := os.Getenv("BS_AI_MCP_PATH")
	if path == "" {
		path = filepath.Join(configDir, configName)
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && os.Getenv("BS_AI_MCP_PATH") == "" {
		return &Config{MCPServers: map[string]ServerConfig{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read MCP configuration: %w", err)
	}
	return parseConfig(b, path)
}

// ValidateBytes applies the same strict parsing, environment resolution, and
// semantic validation as Load without starting any MCP processes.
func ValidateBytes(content []byte, path string) error {
	_, err := parseConfig(content, path)
	return err
}

func parseConfig(content []byte, path string) (*Config, error) {
	var config Config
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("parse MCP configuration: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	config.Configured = true
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve MCP configuration path: %w", err)
	}
	config.path = absolutePath
	if err := config.validateAndResolve(); err != nil {
		return nil, err
	}
	return &config, nil
}

func ensureEOF(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("parse MCP configuration: multiple JSON values")
		}
		return fmt.Errorf("parse MCP configuration: %w", err)
	}
	return nil
}

func (c *Config) validateAndResolve() error {
	if c.MCPServers == nil {
		return errors.New("MCP configuration requires mcpServers")
	}
	if len(c.MCPServers) > 20 {
		return errors.New("MCP configuration exceeds 20 servers")
	}
	base := filepath.Dir(c.path)
	for name, s := range c.MCPServers {
		if !serverNamePattern.MatchString(name) {
			return fmt.Errorf("MCP server name %q must match [A-Za-z0-9_-]{1,32}", name)
		}
		if !s.enabled() {
			c.MCPServers[name] = s
			continue
		}
		if (s.Command == "") == (s.URL == "") {
			return fmt.Errorf("MCP server %q must specify exactly one of command or url", name)
		}
		if s.Command != "" && strings.TrimSpace(s.Command) == "" {
			return fmt.Errorf("MCP server %q command must not be blank", name)
		}
		if strings.ContainsRune(s.Command, 0) {
			return fmt.Errorf("MCP server %q command contains a NUL byte", name)
		}
		for _, arg := range s.Args {
			if strings.ContainsRune(arg, 0) {
				return fmt.Errorf("MCP server %q argument contains a NUL byte", name)
			}
		}
		if len(s.AllowedTools) > maxTools {
			return fmt.Errorf("MCP server %q allowed_tools exceeds %d entries", name, maxTools)
		}
		allowed := make(map[string]bool, len(s.AllowedTools))
		for _, toolName := range s.AllowedTools {
			if strings.TrimSpace(toolName) == "" {
				return fmt.Errorf("MCP server %q allowed_tools contains an empty name", name)
			}
			if allowed[toolName] {
				return fmt.Errorf("MCP server %q allowed_tools contains duplicate %q", name, toolName)
			}
			allowed[toolName] = true
		}
		if s.ConnectTimeoutSeconds == 0 {
			s.ConnectTimeoutSeconds = 15
		}
		if s.CallTimeoutSeconds == 0 {
			s.CallTimeoutSeconds = 60
		}
		if s.ConnectTimeoutSeconds < 1 || s.ConnectTimeoutSeconds > 120 || s.CallTimeoutSeconds < 1 || s.CallTimeoutSeconds > 600 {
			return fmt.Errorf("MCP server %q has an invalid timeout", name)
		}
		if s.Command != "" {
			if len(s.Headers) != 0 {
				return fmt.Errorf("MCP server %q: headers are only valid with url", name)
			}
			if s.Cwd == "" {
				s.Cwd = base
			} else if !filepath.IsAbs(s.Cwd) {
				s.Cwd = filepath.Join(base, s.Cwd)
			}
		} else {
			if len(s.Args) != 0 || s.Cwd != "" || len(s.Env) != 0 {
				return fmt.Errorf("MCP server %q: args, cwd, and env are only valid with command", name)
			}
			if err := validateRemoteURL(s.URL); err != nil {
				return fmt.Errorf("MCP server %q: %w", name, err)
			}
		}
		var err error
		s.Env, err = resolveMap(s.Env)
		if err != nil {
			return fmt.Errorf("MCP server %q env: %w", name, err)
		}
		for envName := range s.Env {
			if strings.ContainsRune(envName, '=') {
				return fmt.Errorf("MCP server %q has invalid environment key %q", name, envName)
			}
		}
		s.Headers, err = resolveMap(s.Headers)
		if err != nil {
			return fmt.Errorf("MCP server %q headers: %w", name, err)
		}
		for header := range s.Headers {
			if textproto.CanonicalMIMEHeaderKey(header) == "" {
				return fmt.Errorf("MCP server %q has invalid HTTP header %q", name, header)
			}
		}
		c.MCPServers[name] = s
	}
	return nil
}

func resolveMap(in map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(in))
	for key, value := range in {
		if key == "" || strings.ContainsAny(key, "\r\n") {
			return nil, errors.New("contains an invalid key")
		}
		if strings.HasPrefix(value, "env:") {
			ref := strings.TrimPrefix(value, "env:")
			resolved, ok := os.LookupEnv(ref)
			if ref == "" || !ok || resolved == "" {
				return nil, fmt.Errorf("environment variable %q is missing or empty", ref)
			}
			value = resolved
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return nil, fmt.Errorf("value for %q contains an invalid control character", key)
		}
		out[key] = value
	}
	return out, nil
}

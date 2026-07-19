// Package auth manages the single operator-level API token used to protect the
// server's API routes. The token is an opaque, long-lived secret stored in a
// .bs-token file alongside the binary and presented by clients via the
// Authorization: Bearer <token> header.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const tokenFileName = ".bs-token"

// tokenBytes is the number of random bytes in a generated token (hex-encoded to
// twice this many characters).
const tokenBytes = 32

var (
	mu      sync.RWMutex
	current string // the in-memory expected token, loaded at startup
)

// TokenPath returns the path to the .bs-token file. It honors the
// SERVER_TOKEN_PATH environment variable, otherwise it resolves the file next
// to the running binary (consistent with how DATA_PATH defaults work).
func TokenPath() (string, error) {
	if p := os.Getenv("SERVER_TOKEN_PATH"); p != "" {
		return p, nil
	}
	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(ex), tokenFileName), nil
}

// generate creates a cryptographically random hex token.
func generate() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// write saves the token to path with restrictive (0600) permissions.
func write(path, token string) error {
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

// Generate creates a new token and saves it, refusing to overwrite an existing
// token file. Returns the generated token and the path it was written to.
func Generate() (token, path string, err error) {
	path, err = TokenPath()
	if err != nil {
		return "", "", err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return "", path, fmt.Errorf("token already exists at %s (use 'token refresh' to rotate it)", path)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", path, fmt.Errorf("failed to check token file: %w", statErr)
	}
	token, err = generate()
	if err != nil {
		return "", path, err
	}
	if err = write(path, token); err != nil {
		return "", path, err
	}
	return token, path, nil
}

// Refresh regenerates the token and overwrites any existing token file. Returns
// the new token and the path it was written to.
func Refresh() (token, path string, err error) {
	path, err = TokenPath()
	if err != nil {
		return "", "", err
	}
	token, err = generate()
	if err != nil {
		return "", path, err
	}
	if err = write(path, token); err != nil {
		return "", path, err
	}
	return token, path, nil
}

// Load reads the token from disk into memory so the middleware can validate
// requests against it. Returns os.ErrNotExist (wrapped) if no token file exists.
func Load() error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return fmt.Errorf("token file %s is empty", path)
	}
	mu.Lock()
	current = token
	mu.Unlock()
	return nil
}

// Configured reports whether a non-empty token has been loaded into memory.
func Configured() bool {
	mu.RLock()
	defer mu.RUnlock()
	return current != ""
}

// Valid reports whether the supplied token matches the loaded token using a
// constant-time comparison to avoid timing attacks. Returns false if no token
// is configured.
func Valid(token string) bool {
	mu.RLock()
	expected := current
	mu.RUnlock()
	if expected == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1
}

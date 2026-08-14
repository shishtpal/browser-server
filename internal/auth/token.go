// Package auth manages the disjoint operator and administrator API tokens used
// to protect the server's API routes. Tokens are opaque, long-lived secrets
// stored beside the binary unless their paths are overridden by environment.
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
	current string // the in-memory expected operator token, loaded at startup
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

// generateToken creates a cryptographically random hex token.
func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// writeTokenFile saves a token with restrictive permissions. OpenFile is used
// instead of relying solely on WriteFile so an existing token's mode is also
// tightened on Unix-like systems when it is rotated.
func writeTokenFile(path, token string) error {
	return writeTokenFileWithFlags(path, token, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}

func writeNewTokenFile(path, token string) error {
	err := writeTokenFileWithFlags(path, token, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil && !errors.Is(err, os.ErrExist) {
		// Clean up the partial file so a failed create does not leave behind an
		// unusable token that later blocks `token generate` with 'exists'.
		_ = os.Remove(path)
	}
	return err
}

func writeTokenFileWithFlags(path, token string, flags int) error {
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to secure token file: %w", err)
	}
	if _, err := file.WriteString(token + "\n"); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to write token file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to sync token file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close token file: %w", err)
	}
	return nil
}

func generateAt(path, existsMessage string) (string, string, error) {
	if _, statErr := os.Stat(path); statErr == nil {
		return "", path, errors.New(existsMessage)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", path, fmt.Errorf("failed to check token file: %w", statErr)
	}
	token, err := generateToken()
	if err != nil {
		return "", path, err
	}
	if err := writeNewTokenFile(path, token); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", path, errors.New(existsMessage)
		}
		return "", path, err
	}
	return token, path, nil
}

func refreshAt(path string) (string, string, error) {
	token, err := generateToken()
	if err != nil {
		return "", path, err
	}
	if err := writeTokenFile(path, token); err != nil {
		return "", path, err
	}
	return token, path, nil
}

func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return token, nil
}

func tokenMatches(expected, supplied string) bool {
	if expected == "" || supplied == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) == 1
}

// Generate creates a new operator token and saves it, refusing to overwrite an
// existing token file. Returns the generated token and its path.
func Generate() (token, path string, err error) {
	path, err = TokenPath()
	if err != nil {
		return "", "", err
	}
	return generateAt(path, fmt.Sprintf("token already exists at %s (use 'token refresh' to rotate it)", path))
}

// Refresh regenerates the operator token and overwrites any existing token
// file. Returns the new token and its path.
func Refresh() (token, path string, err error) {
	path, err = TokenPath()
	if err != nil {
		return "", "", err
	}
	return refreshAt(path)
}

// Load reads the operator token from disk into memory so middleware can
// validate requests against it.
func Load() error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	token, err := readTokenFile(path)
	if err != nil {
		mu.Lock()
		current = ""
		mu.Unlock()
		return err
	}
	mu.Lock()
	current = token
	mu.Unlock()
	return nil
}

// Configured reports whether a non-empty operator token has been loaded.
func Configured() bool {
	mu.RLock()
	defer mu.RUnlock()
	return current != ""
}

// Valid reports whether the supplied token matches the loaded operator token
// using a constant-time comparison.
func Valid(token string) bool {
	mu.RLock()
	expected := current
	mu.RUnlock()
	return tokenMatches(expected, token)
}

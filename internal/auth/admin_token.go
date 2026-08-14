package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const adminTokenFileName = ".bs-token-admin"

var (
	adminMu      sync.RWMutex
	adminCurrent string
)

// AdminTokenPath returns the administrator token path. The
// SERVER_ADMIN_TOKEN_PATH override mirrors SERVER_TOKEN_PATH; otherwise the
// token lives beside the running executable.
func AdminTokenPath() (string, error) {
	if path := os.Getenv("SERVER_ADMIN_TOKEN_PATH"); path != "" {
		return path, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(executable), adminTokenFileName), nil
}

// AdminGenerate creates the opt-in administrator token without overwriting an
// existing one.
func AdminGenerate() (token, path string, err error) {
	path, err = AdminTokenPath()
	if err != nil {
		return "", "", err
	}
	return generateAt(path, fmt.Sprintf("admin token already exists at %s (use 'token admin-refresh' to rotate it)", path))
}

// AdminRefresh rotates (or creates) the administrator token.
func AdminRefresh() (token, path string, err error) {
	path, err = AdminTokenPath()
	if err != nil {
		return "", "", err
	}
	return refreshAt(path)
}

// AdminDelete removes the administrator token file. A running server retains
// its startup-loaded credential until it is restarted; clearing the local copy
// here keeps in-process callers consistent.
func AdminDelete() (string, error) {
	path, err := AdminTokenPath()
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return path, fmt.Errorf("failed to remove admin token file: %w", err)
	}
	adminMu.Lock()
	adminCurrent = ""
	adminMu.Unlock()
	return path, nil
}

// AdminLoad loads the administrator token. The missing file is returned as
// os.ErrNotExist so startup can report that the optional admin API is disabled.
func AdminLoad() error {
	path, err := AdminTokenPath()
	if err != nil {
		return err
	}
	token, err := readTokenFile(path)
	if err != nil {
		adminMu.Lock()
		adminCurrent = ""
		adminMu.Unlock()
		return err
	}
	adminMu.Lock()
	adminCurrent = token
	adminMu.Unlock()
	return nil
}

// AdminConfigured reports whether the optional admin tier was loaded.
func AdminConfigured() bool {
	adminMu.RLock()
	defer adminMu.RUnlock()
	return adminCurrent != ""
}

// AdminValid compares only against the admin token. Admin and operator tokens
// deliberately do not grant access to each other's routes.
func AdminValid(token string) bool {
	adminMu.RLock()
	expected := adminCurrent
	adminMu.RUnlock()
	return tokenMatches(expected, token)
}

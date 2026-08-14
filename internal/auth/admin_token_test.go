package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdminTokenLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bs-token-admin")
	t.Setenv("SERVER_ADMIN_TOKEN_PATH", path)
	_, _ = AdminDelete()

	token, generatedPath, err := AdminGenerate()
	if err != nil {
		t.Fatal(err)
	}
	if generatedPath != path || token == "" {
		t.Fatalf("AdminGenerate() = %q, %q", token, generatedPath)
	}
	if _, _, err := AdminGenerate(); err == nil {
		t.Fatal("AdminGenerate overwrote an existing token")
	}
	if err := AdminLoad(); err != nil {
		t.Fatal(err)
	}
	if !AdminConfigured() || !AdminValid(token) || AdminValid("wrong") {
		t.Fatal("loaded admin token validation is incorrect")
	}

	rotated, _, err := AdminRefresh()
	if err != nil {
		t.Fatal(err)
	}
	if rotated == token {
		t.Fatal("AdminRefresh did not rotate the token")
	}
	if err := AdminLoad(); err != nil || !AdminValid(rotated) {
		t.Fatalf("rotated token was not loaded: %v", err)
	}
	if _, err := AdminDelete(); err != nil {
		t.Fatal(err)
	}
	if AdminConfigured() {
		t.Fatal("AdminDelete left the in-memory tier configured")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("admin token still exists: %v", err)
	}
}

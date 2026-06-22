package main

import "testing"

func TestResolveServerPortDefault(t *testing.T) {
	t.Setenv("PORT", "")

	port, err := resolveServerPort(nil)
	if err != nil {
		t.Fatalf("resolveServerPort returned error: %v", err)
	}
	if port != defaultPort {
		t.Fatalf("resolveServerPort() = %q, want %q", port, defaultPort)
	}
}

func TestResolveServerPortFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")

	port, err := resolveServerPort(nil)
	if err != nil {
		t.Fatalf("resolveServerPort returned error: %v", err)
	}
	if port != "9090" {
		t.Fatalf("resolveServerPort() = %q, want %q", port, "9090")
	}
}

func TestResolveServerPortFromFlag(t *testing.T) {
	t.Setenv("PORT", "")

	port, err := resolveServerPort([]string{"--port", "7070"})
	if err != nil {
		t.Fatalf("resolveServerPort returned error: %v", err)
	}
	if port != "7070" {
		t.Fatalf("resolveServerPort() = %q, want %q", port, "7070")
	}
}

func TestResolveServerPortFlagOverridesEnv(t *testing.T) {
	t.Setenv("PORT", "9090")

	port, err := resolveServerPort([]string{"--port", "7070"})
	if err != nil {
		t.Fatalf("resolveServerPort returned error: %v", err)
	}
	if port != "7070" {
		t.Fatalf("resolveServerPort() = %q, want %q", port, "7070")
	}
}

func TestResolveServerPortRejectsInvalidValues(t *testing.T) {
	t.Setenv("PORT", "")

	tests := []struct {
		name string
		args []string
	}{
		{name: "non numeric", args: []string{"--port", "abc"}},
		{name: "zero", args: []string{"--port", "0"}},
		{name: "negative", args: []string{"--port", "-1"}},
		{name: "too high", args: []string{"--port", "65536"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := resolveServerPort(tt.args); err == nil {
				t.Fatal("resolveServerPort returned nil error")
			}
		})
	}
}

func TestResolveServerPortRejectsInvalidEnvValue(t *testing.T) {
	t.Setenv("PORT", "abc")

	if _, err := resolveServerPort(nil); err == nil {
		t.Fatal("resolveServerPort returned nil error")
	}
}

package config

import (
	"os"
	"path/filepath"
)

// ExecutableDir returns the directory containing the running binary. It is
// the anchor for every JSON config file so the tools work no matter which
// directory the user happens to be in when they invoke the binary.
func ExecutableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Resolve symlinks so a binary invoked through a link still finds the
	// configs that sit next to the real file.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
}

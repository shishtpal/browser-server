package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxPathsAdditionalDirs = 20
	maxPathsBinaries       = 30
)

// validatePaths validates the paths configuration section. Empty values are
// allowed (no-op), and relative paths are resolved relative to the config
// file's directory via cfg.ResolvePath.
func validatePaths(cfg PathsConfig, base *Config) error {
	if len(cfg.AdditionalDirs) > maxPathsAdditionalDirs {
		return fmt.Errorf("paths.additional_dirs exceeds %d entries", maxPathsAdditionalDirs)
	}
	for _, dir := range cfg.AdditionalDirs {
		if strings.TrimSpace(dir) == "" {
			return fmt.Errorf("paths.additional_dirs contains an empty entry")
		}
		resolved := base.ResolvePath(dir)
		info, err := os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("paths.additional_dirs %q: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("paths.additional_dirs %q is not a directory", dir)
		}
	}
	if len(cfg.Binaries) > maxPathsBinaries {
		return fmt.Errorf("paths.binaries exceeds %d entries", maxPathsBinaries)
	}
	for name, full := range cfg.Binaries {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("paths.binaries has an empty name")
		}
		if strings.ContainsAny(name, `/\`) || strings.Contains(name, string(filepath.Separator)) {
			return fmt.Errorf("paths.binaries key %q must be a simple name without path separators", name)
		}
		if strings.TrimSpace(full) == "" {
			return fmt.Errorf("paths.binaries %q has an empty path", name)
		}
		resolved := full
		if !filepath.IsAbs(resolved) {
			resolved = base.ResolvePath(full)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("paths.binaries %q: %w", name, err)
		}
		if info.IsDir() {
			return fmt.Errorf("paths.binaries %q points to a directory, not a file", name)
		}
	}
	return nil
}

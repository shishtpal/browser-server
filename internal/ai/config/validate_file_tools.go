package config

import (
	"fmt"
	"path"
	"path/filepath"
)

func validateFileToolsGlobs(patterns []string) error {
	for _, p := range patterns {
		if _, err := path.Match(filepath.ToSlash(p), ""); err != nil {
			return fmt.Errorf("invalid glob %q: %w", p, err)
		}
	}
	return nil
}

func validateFileTools(cfg FileToolsConfig) error {
	if cfg.MaxReadBytes < 4096 || cfg.MaxReadBytes > 512*1024 {
		return fmt.Errorf("file_tools.max_read_bytes must be between 4096 and 524288")
	}
	if cfg.MaxLineReadBytes < 4096 || cfg.MaxLineReadBytes > 1024*1024 {
		return fmt.Errorf("file_tools.max_line_read_bytes must be between 4096 and 1048576")
	}
	if cfg.MaxLineCount < 100 || cfg.MaxLineCount > 50000 {
		return fmt.Errorf("file_tools.max_line_count must be between 100 and 50000")
	}
	if cfg.MaxFileSizeWarnMB < 1 || cfg.MaxFileSizeWarnMB > 10000 {
		return fmt.Errorf("file_tools.max_file_size_warn_mb must be between 1 and 10000")
	}
	if err := validateFileToolsGlobs(cfg.BlockedPathPatterns); err != nil {
		return fmt.Errorf("file_tools: %w", err)
	}
	return nil
}

// Package ocr wraps the pdftoppm CLI for rasterizing PDFs into page images.
// It follows the project's pattern of shell-free exec.CommandContext
// invocation and config-driven binary resolution (configured dir > PATH).
package ocr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"browser-server/internal/ai/config"
)

// runCommand is replaceable in tests.
var runCommand = defaultRunCommand

func defaultRunCommand(ctx context.Context, exe string, args ...string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

var pdftoppmPagePattern = regexp.MustCompile(`^page-\d+\.(png|ppm|jpeg|jpg)$`)

// Resolve returns the path to the pdftoppm executable. It checks config dir,
// supplementary dirs, PATH, and known poppler-windows install shapes.
func Resolve(cfg config.OCRPopplerConfig, additionalDirs []string) (string, error) {
	dirs := make([]string, 0, len(additionalDirs)+1)
	if cfg.Dir != "" {
		dirs = append(dirs, cfg.Dir)
	}
	dirs = append(dirs, additionalDirs...)
	for _, dir := range dirs {
		candidate := filepath.Join(dir, pdftoppmName())
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath(pdftoppmName()); err == nil {
		return p, nil
	}
	return "", errors.New("poppler not found; set ocr.poppler.dir or add pdftoppm to PATH")
}

func pdftoppmName() string {
	if runtime.GOOS == "windows" {
		return "pdftoppm.exe"
	}
	return "pdftoppm"
}

// Convert rasterizes firstPage..lastPage (1-indexed, 0=unset) of pdfPath into
// PNG/ppm/jpeg images in outDir, enforcing timing constraints and the page
// count cap from cfg. It returns the sorted list of generated image paths.
func Convert(ctx context.Context, cfg config.OCRPopplerConfig, pdfPath, outDir string, firstPage, lastPage int, additionalDirs []string) ([]string, error) {
	exe, err := Resolve(cfg, additionalDirs)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir %q: %w", outDir, err)
	}

	// Defaults are applied once at config load time (config.applyOCRDefaults).
	dpi := cfg.DPI
	format := cfg.Format
	maxPages := cfg.MaxPages
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second

	actualFirst := firstPage
	if actualFirst <= 0 {
		actualFirst = 1
	}
	actualLast := lastPage
	if actualLast <= 0 {
		actualLast = actualFirst + maxPages - 1
	}
	if actualLast < actualFirst {
		return nil, fmt.Errorf("last_page %d is before first_page %d", actualLast, actualFirst)
	}
	if actualLast-actualFirst+1 > maxPages {
		return nil, fmt.Errorf("page range %d-%d exceeds ocr.poppler.max_pages %d; narrow first_page/last_page or raise the cap", actualFirst, actualLast, maxPages)
	}

	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"-r", fmt.Sprint(dpi),
		"-" + format,
		"-f", fmt.Sprint(actualFirst),
		"-l", fmt.Sprint(actualLast),
		pdfPath,
		filepath.Join(outDir, "page"),
	}
	if err := runCommand(ctx2, exe, args...); err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("pdftoppm timed out after %s", timeout)
		}
		return nil, fmt.Errorf("run pdftoppm: %w", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, fmt.Errorf("read output dir: %w", err)
	}
	var pages []string
	for _, e := range entries {
		name := e.Name()
		if pdftoppmPagePattern.MatchString(name) {
			pages = append(pages, name)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdftoppm reported success but produced no pages in %q", outDir)
	}
	sort.Strings(pages)
	for i, p := range pages {
		pages[i] = filepath.Join(outDir, p)
	}
	return pages, nil
}

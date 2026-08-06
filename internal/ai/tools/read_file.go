package tools

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"browser-server/internal/ai/config"
)

//go:embed schemas/read_file.json
var readFileSchema []byte

var errBinary = fmt.Errorf("file appears to be binary (contains null bytes). read_file only supports UTF-8 encoded text files.")

type fileReadTool struct {
	maxReadBytes      int
	maxLineReadBytes  int
	maxLineCount      int
	maxFileSizeWarnMB int
	allowedExts       []string
	blockedPatterns   []string
}

func registerReadFile(r *Registry, cfg config.FileToolsConfig) {
	ft := &fileReadTool{
		maxReadBytes:      clamp(cfg.MaxReadBytes, 4096, 512*1024, 32*1024),
		maxLineReadBytes:  clamp(cfg.MaxLineReadBytes, 4096, 1024*1024, 64*1024),
		maxLineCount:      clamp(cfg.MaxLineCount, 100, 50000, 5000),
		maxFileSizeWarnMB: clamp(cfg.MaxFileSizeWarnMB, 1, 10000, 100),
	}
	for _, p := range cfg.BlockedPathPatterns {
		ft.blockedPatterns = append(ft.blockedPatterns, filepath.ToSlash(p))
	}
	for _, e := range cfg.AllowedExtensions {
		e = strings.TrimSpace(e)
		if e != "" {
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			ft.allowedExts = append(ft.allowedExts, strings.ToLower(e))
		}
	}

	r.add(Tool{
		Name:        "read_file",
		Category:    "File Operations",
		Description: "Read a UTF-8 text file from the server filesystem (maximum " + formatBytes(int64(ft.maxReadBytes)) + " for full reads, configurable in the AI configuration). Supports reading specific line ranges and prefixing lines with their line numbers.",
		Schema:      json.RawMessage(readFileSchema),
		Execute:        ft.readFile,
		RawContentFunc: rawMapField("content"),
	})

	// read_files reuses the same configured limits and security checks via ft.
	registerReadFiles(r, ft)
}

type lineRange struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (ft *fileReadTool) readFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path        string      `json:"path"`
		Offset      int         `json:"offset"`
		Limit       int         `json:"limit"`
		Ranges      []lineRange `json:"ranges"`
		LineNumbers bool        `json:"line_numbers"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "offset": true, "limit": true, "ranges": true, "line_numbers": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if len(a.Ranges) > 0 && (a.Offset != 0 || a.Limit != 0) {
		return nil, fmt.Errorf("ranges is mutually exclusive with offset and limit")
	}
	if a.Offset < 0 || a.Limit < 0 {
		return nil, fmt.Errorf("offset and limit must be non-negative")
	}
	if a.Offset == 0 {
		a.Offset = 1
	}
	for i, r := range a.Ranges {
		if r.Offset < 1 {
			return nil, fmt.Errorf("ranges[%d].offset must be >= 1", i)
		}
		if r.Limit < 1 {
			return nil, fmt.Errorf("ranges[%d].limit must be >= 1", i)
		}
	}

	resolvedPath, err := ft.resolvePath(a.Path)
	if err != nil {
		return nil, err
	}

	// Security: check blocked path patterns
	if ft.isBlocked(resolvedPath) {
		return nil, fmt.Errorf("reading %q is not allowed (blocked by path pattern)", a.Path)
	}

	// Security: check allowed extensions
	if len(ft.allowedExts) > 0 && !ft.hasAllowedExt(resolvedPath) {
		return nil, fmt.Errorf("reading %q is not allowed: file extension not in allowed list", a.Path)
	}

	// Check file size before opening (warn for very large files even with ranges)
	if ft.maxFileSizeWarnMB > 0 {
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return nil, friendlyFileError(a.Path, err)
		}
		if info.Size() > int64(ft.maxFileSizeWarnMB)*1024*1024 {
			return nil, fmt.Errorf("file %q is %.1f MB, which exceeds the %d MB limit", a.Path, float64(info.Size())/(1024*1024), ft.maxFileSizeWarnMB)
		}
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, friendlyFileError(a.Path, err)
	}
	defer file.Close()

	// Case: full-file read (no offset/limit/ranges specified)
	if len(a.Ranges) == 0 && a.Offset <= 1 && a.Limit == 0 {
		data, err := io.ReadAll(io.LimitReader(file, int64(ft.maxReadBytes)+1))
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		if len(data) > ft.maxReadBytes {
			return nil, fmt.Errorf("file exceeds %s read limit. Use offset/limit or ranges to read specific portions.", formatBytes(int64(ft.maxReadBytes)))
		}
		if err := validateUTF8(data); err != nil {
			return nil, err
		}
		lines := splitLines(string(data))
		return ft.buildResult(a.Path, lines, nil, a.LineNumbers), nil
	}

	// Case: range-limited read — scan line-by-line with a bufio.Scanner
	lines, lineNums, err := ft.readLines(file, a.Offset, a.Limit, a.Ranges)
	if err != nil {
		return nil, err
	}
	return ft.buildResult(a.Path, lines, lineNums, a.LineNumbers), nil
}

// resolvePath normalises the raw path input to an absolute path on the host OS.
func (ft *fileReadTool) resolvePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("path is required")
	}
	// On Windows, strip leading forward slash from /D:/foo style paths
	if runtime.GOOS == "windows" && len(raw) > 1 && raw[0] == '/' && raw[1] != '/' {
		raw = raw[1:]
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %v", raw, err)
	}
	return abs, nil
}

// isBlocked checks if the resolved path matches any blocked pattern.
func (ft *fileReadTool) isBlocked(resolved string) bool {
	slashed := filepath.ToSlash(resolved)
	base := filepath.Base(slashed)
	for _, pattern := range ft.blockedPatterns {
		// Match against full path
		if ok, _ := path.Match(pattern, slashed); ok {
			return true
		}
		// Match against filename only
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
		// Support patterns like **/.env* by matching the suffix against any path segment.
		if strings.HasPrefix(pattern, "**/") {
			suffix := strings.TrimPrefix(pattern, "**/")
			for _, segment := range strings.Split(slashed, "/") {
				if ok, _ := path.Match(suffix, segment); ok {
					return true
				}
			}
		}
		// Support patterns like **/.git/** by matching segments and descendants.
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(slashed, prefix) || strings.Contains(slashed, "/"+prefix+"/") {
				return true
			}
		}
	}
	return false
}

// hasAllowedExt checks if the resolved path has an allowed extension.
func (ft *fileReadTool) hasAllowedExt(resolved string) bool {
	ext := strings.ToLower(filepath.Ext(resolved))
	for _, allowed := range ft.allowedExts {
		if ext == allowed {
			return true
		}
	}
	return false
}

// readLines reads specific line ranges from a file using a bufio.Scanner.
func (ft *fileReadTool) readLines(file *os.File, offset, limit int, ranges []lineRange) ([]string, []int, error) {
	// Build a set of line numbers we care about
	wanted := make(map[int]bool)
	ordered := make([]int, 0)

	switch {
	case len(ranges) > 0:
		for _, r := range ranges {
			for i := 0; i < r.Limit; i++ {
				ln := r.Offset + i
				if !wanted[ln] {
					wanted[ln] = true
					ordered = append(ordered, ln)
				}
			}
		}
	default:
		if limit == 0 {
			limit = ft.maxLineCount
		}
		for i := 0; i < limit; i++ {
			ln := offset + i
			wanted[ln] = true
			ordered = append(ordered, ln)
		}
	}

	scanner := bufio.NewScanner(file)
	// Set a custom split function that limits the buffer size
	scanner.Buffer(make([]byte, 0, 64*1024), ft.maxLineReadBytes)

	var lines []string
	var lineNums []int
	lineNo := 0
	maxLines := ft.maxLineCount

	for scanner.Scan() {
		lineNo++
		if len(lines) >= maxLines {
			break
		}
		if wanted[lineNo] {
			lines = append(lines, scanner.Text())
			lineNums = append(lineNums, lineNo)
			delete(wanted, lineNo)
		}
		// Stop scanning if we've found all wanted lines and no more are needed
		if len(wanted) == 0 && lineNo >= ordered[len(ordered)-1] {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Validate UTF-8 on the collected content
	content := strings.Join(lines, "\n")
	if !utf8.ValidString(content) {
		return nil, nil, fmt.Errorf("file is not valid UTF-8 text. Only UTF-8 encoded files are supported.")
	}

	return lines, lineNums, nil
}

// buildResult constructs the final result map.
func (ft *fileReadTool) buildResult(path string, lines []string, lineNums []int, showLineNumbers bool) map[string]any {
	content := strings.Join(lines, "\n")
	if showLineNumbers && lineNums == nil {
		lineNums = make([]int, len(lines))
		for i := range lineNums {
			lineNums[i] = i + 1
		}
	}
	if showLineNumbers {
		var sb strings.Builder
		for i, line := range lines {
			if i > 0 {
				sb.WriteByte('\n')
			}
			fmt.Fprintf(&sb, "%d: %s", lineNums[i], line)
		}
		content = sb.String()
	}
	return map[string]any{
		"content":      content,
		"path":         path,
		"line_count":   len(lines),
		"line_numbers": showLineNumbers,
	}
}

// splitLines splits a string into lines, removing a trailing empty line.
func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// validateUTF8 checks that data is valid UTF-8 and not binary.
func validateUTF8(data []byte) error {
	if !utf8.Valid(data) {
		if bytes.IndexByte(data, 0) != -1 {
			return errBinary
		}
		// Check if it's likely binary (high ratio of non-printable chars)
		if isLikelyBinary(data) {
			return errBinary
		}
		return fmt.Errorf("file is not valid UTF-8 text. Only UTF-8 encoded files are supported.")
	}
	return nil
}

// isLikelyBinary checks if the data appears to be binary content.
func isLikelyBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	nonPrintable := 0
	sample := data
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	for _, b := range sample {
		if b == 0 {
			return true // null byte = definitely binary
		}
		if b != '\n' && b != '\r' && b != '\t' && (b < 0x20 || b > 0x7E) {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(sample)) > 0.30
}

// friendlyFileError translates OS file errors to user-friendly messages.
func friendlyFileError(path string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if os.IsPermission(err) {
		return fmt.Errorf("permission denied: %s", path)
	}
	return fmt.Errorf("cannot read file %q: %v", path, err)
}

// clamp returns value clamped to [min, max]. If value is 0, returns defaultVal.
func clamp(value, minVal, maxVal, defaultVal int) int {
	if value == 0 {
		return defaultVal
	}
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

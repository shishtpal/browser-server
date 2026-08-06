package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

//go:embed schemas/read_files.json
var readFilesSchema []byte

func registerReadFiles(r *Registry, ft *fileReadTool) {
	r.add(Tool{
		Name:     "read_files",
		Category: "File Operations",
		Description: "Read multiple UTF-8 text files from the server filesystem in a single call. " +
			"Each entry in files is a path with an optional line-range suffix: \"path\" reads the whole file " +
			"(up to the configured read limit), \"path:offset:limit\" reads limit lines starting at the 1-based " +
			"offset, and \"path:start-end\" reads an inclusive line range. Set line_numbers to true to prefix " +
			"each line with its 1-based line number.",
		Schema:         json.RawMessage(readFilesSchema),
		Execute:        ft.readFiles,
		RawContentFunc: rawMapField("content"),
	})
}

// fileSpec is one parsed entry from the read_files files array: a path plus an
// optional line range. Ranges are stored as offset+limit so the existing
// readLines scanner can be reused unchanged.
type fileSpec struct {
	path    string
	offset  int // 1-based; 0 means full-file read
	limit   int // 0 means all remaining lines
	hasRng  bool
	rngDisp string // canonical range display for the header, e.g. "10:12" or "10-15"
}

// parseFileSpec splits a "path[:range]" spec. The range suffix is detected by
// splitting on ':' after the last path separator, so Windows drive letters
// (D:/...) are never mistaken for a range. Supported suffixes:
//
//	path:OFFSET:LIMIT  → LIMIT lines starting at 1-based OFFSET
//	path:START-END     → lines START through END inclusive
func parseFileSpec(raw string) (fileSpec, error) {
	var s fileSpec
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s, fmt.Errorf("files entries must not be empty")
	}

	lastSep := strings.LastIndexAny(raw, "/\\")
	tail := raw[lastSep+1:]
	parts := strings.Split(tail, ":")

	// No colon (or a colon only in the drive-letter portion) → plain path.
	if len(parts) == 1 {
		s.path = raw
		return s, nil
	}

	switch {
	case len(parts) == 2 && strings.Count(parts[1], "-") == 1:
		// path:START-END
		kv := strings.SplitN(parts[1], "-", 2)
		start, errA := strconv.Atoi(kv[0])
		end, errB := strconv.Atoi(kv[1])
		if errA != nil || errB != nil || start < 1 || end < start {
			return s, fmt.Errorf("invalid range in %q: want start-end with start >= 1 and end >= start", raw)
		}
		s.offset, s.limit = start, end-start+1
		s.rngDisp = fmt.Sprintf("%d-%d", start, end)
	case len(parts) == 3:
		// path:OFFSET:LIMIT
		offset, errA := strconv.Atoi(parts[1])
		limit, errB := strconv.Atoi(parts[2])
		if errA != nil || errB != nil || offset < 1 || limit < 1 {
			return s, fmt.Errorf("invalid range in %q: want offset >= 1 and limit >= 1", raw)
		}
		s.offset, s.limit = offset, limit
		s.rngDisp = fmt.Sprintf("%d:%d", offset, limit)
	default:
		return s, fmt.Errorf("invalid range in %q: use :offset:limit or :start-end", raw)
	}

	s.path = raw[:len(raw)-(len(tail)-len(parts[0]))]
	s.hasRng = true
	if strings.TrimSpace(s.path) == "" {
		return s, fmt.Errorf("missing path before range in %q", raw)
	}
	return s, nil
}

// readFiles reads each requested file and concatenates the results with
// "# File: <path>[:range]" headers. A failure on one file does not abort the
// remaining files; the error is rendered inline in that file's section so the
// caller still gets everything that could be read.
func (ft *fileReadTool) readFiles(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Files       []string `json:"files"`
		LineNumbers bool     `json:"line_numbers"`
	}
	if err := strict(raw, &a, map[string]bool{"files": true, "line_numbers": true}); err != nil {
		return nil, err
	}
	if len(a.Files) == 0 {
		return nil, fmt.Errorf("files is required and must not be empty")
	}
	if len(a.Files) > 20 {
		return nil, fmt.Errorf("too many files: %d (maximum 20 per call)", len(a.Files))
	}

	var sb strings.Builder
	read := 0
	for i, specStr := range a.Files {
		spec, err := parseFileSpec(specStr)
		header := "# File: " + strings.TrimSpace(specStr)
		if err == nil && spec.hasRng {
			header = "# File: " + spec.path + ":" + spec.rngDisp
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(header)
		sb.WriteByte('\n')
		if err != nil {
			fmt.Fprintf(&sb, "error: %v", err)
			continue
		}
		content, err := ft.readOneFile(spec, a.LineNumbers)
		if err != nil {
			fmt.Fprintf(&sb, "error: %v", err)
			continue
		}
		sb.WriteString(content)
		read++
	}

	return map[string]any{
		"content":      sb.String(),
		"files_read":   read,
		"files_total":  len(a.Files),
		"line_numbers": a.LineNumbers,
	}, nil
}

// readOneFile resolves, validates, and reads a single file spec, reusing the
// same security checks (blocked patterns, allowed extensions, size limits)
// and line scanner as read_file.
func (ft *fileReadTool) readOneFile(spec fileSpec, lineNumbers bool) (string, error) {
	resolvedPath, err := ft.resolvePath(spec.path)
	if err != nil {
		return "", err
	}
	if ft.isBlocked(resolvedPath) {
		return "", fmt.Errorf("reading %q is not allowed (blocked by path pattern)", spec.path)
	}
	if len(ft.allowedExts) > 0 && !ft.hasAllowedExt(resolvedPath) {
		return "", fmt.Errorf("reading %q is not allowed: file extension not in allowed list", spec.path)
	}
	if ft.maxFileSizeWarnMB > 0 {
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return "", friendlyFileError(spec.path, err)
		}
		if info.Size() > int64(ft.maxFileSizeWarnMB)*1024*1024 {
			return "", fmt.Errorf("file %q is %.1f MB, which exceeds the %d MB limit", spec.path, float64(info.Size())/(1024*1024), ft.maxFileSizeWarnMB)
		}
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return "", friendlyFileError(spec.path, err)
	}
	defer file.Close()

	var lines []string
	var lineNums []int
	if !spec.hasRng {
		data, err := readAllLimited(file, ft.maxReadBytes)
		if err != nil {
			return "", err
		}
		if err := validateUTF8(data); err != nil {
			return "", err
		}
		lines = splitLines(string(data))
	} else {
		lines, lineNums, err = ft.readLines(file, spec.offset, spec.limit, nil)
		if err != nil {
			return "", err
		}
	}

	if lineNumbers && lineNums == nil {
		lineNums = make([]int, len(lines))
		for i := range lineNums {
			lineNums[i] = i + 1
		}
	}
	if !lineNumbers {
		return strings.Join(lines, "\n"), nil
	}
	var sb strings.Builder
	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%d: %s", lineNums[i], line)
	}
	return sb.String(), nil
}

// readAllLimited reads up to maxBytes from r, erroring when the file is
// larger rather than silently truncating.
func readAllLimited(r *os.File, maxBytes int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("file exceeds %s read limit. Use a line-range suffix (:offset:limit or :start-end) to read a portion.", formatBytes(int64(maxBytes)))
	}
	return data, nil
}

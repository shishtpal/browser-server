package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func registerEditFile(r *Registry) {
	r.add(Tool{
		Name:        "edit_file",
		Category:    "File Operations",
		Description: "Apply a unified diff patch to an existing file. The patch must contain --- and +++ file headers and at least one @@ hunk. Context lines must match exactly; use read_file first to inspect the file. Use write_file for new files.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Absolute path to the file to edit"},"patch":{"type":"string","description":"Unified diff text with --- / +++ headers and @@ hunks. Context lines (space prefix) must match the file exactly."},"dry_run":{"type":"boolean","description":"Validate without writing and return a preview of the resulting content."}},"required":["path","patch"],"additionalProperties":false}`),
		Execute:     editFile,
	})
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []diffLine
}

type diffLine struct {
	op      byte
	content string
}

var hunkHeaderRE = regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@`)

func editFile(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Path   string  `json:"path"`
		Patch  *string `json:"patch"`
		DryRun bool    `json:"dry_run"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "patch": true, "dry_run": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if a.Patch == nil || strings.TrimSpace(*a.Patch) == "" {
		return nil, fmt.Errorf("patch is required")
	}

	info, err := os.Lstat(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("cannot edit a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("path must be a regular file")
	}

	content, err := os.ReadFile(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	hunks, err := parseUnifiedDiff(*a.Patch)
	if err != nil {
		return nil, fmt.Errorf("invalid patch: %w", err)
	}
	result, linesAdded, linesRemoved, err := applyHunks(splitFileLines(string(content)), hunks)
	if err != nil {
		return nil, fmt.Errorf("patch apply failed: %w", err)
	}
	lineEnding := "\n"
	if strings.Contains(string(content), "\r\n") {
		lineEnding = "\r\n"
	}
	newContent := joinFileLines(result, lineEnding)

	response := map[string]any{
		"path":          a.Path,
		"success":       true,
		"hunks_applied": len(hunks),
		"lines_added":   linesAdded,
		"lines_removed": linesRemoved,
	}
	if a.DryRun {
		response["dry_run"] = true
		response["preview"] = newContent
		return response, nil
	}
	if err := os.WriteFile(a.Path, []byte(newContent), info.Mode().Perm()); err != nil {
		return nil, fmt.Errorf("failed to write patched file: %w", err)
	}
	return response, nil
}

func parseUnifiedDiff(patch string) ([]hunk, error) {
	scanner := bufio.NewScanner(strings.NewReader(patch))
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var hunks []hunk
	var current *hunk
	seenOldHeader := false
	seenNewHeader := false
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case current == nil && strings.HasPrefix(line, "diff "):
			continue
		case current == nil && strings.HasPrefix(line, "--- "):
			seenOldHeader = true
			continue
		case current == nil && strings.HasPrefix(line, "+++ "):
			seenNewHeader = true
			continue
		case strings.HasPrefix(line, "@@"):
			if !seenOldHeader || !seenNewHeader {
				return nil, fmt.Errorf("missing --- or +++ file header")
			}
			if current != nil {
				if err := validateHunk(*current); err != nil {
					return nil, err
				}
				hunks = append(hunks, *current)
			}
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			current = &h
			continue
		}

		if current == nil {
			if strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("unexpected patch line %q", line)
			}
			continue
		}
		if line == `\ No newline at end of file` {
			continue
		}
		if line == "" {
			current.lines = append(current.lines, diffLine{op: ' '})
			continue
		}
		switch line[0] {
		case ' ', '+', '-':
			current.lines = append(current.lines, diffLine{op: line[0], content: line[1:]})
		default:
			return nil, fmt.Errorf("invalid hunk line %q", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("patch contains no hunks")
	}
	if err := validateHunk(*current); err != nil {
		return nil, err
	}
	return append(hunks, *current), nil
}

func parseHunkHeader(line string) (hunk, error) {
	m := hunkHeaderRE.FindStringSubmatch(line)
	if m == nil {
		return hunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}
	oldStart, _ := strconv.Atoi(m[1])
	oldCount := 1
	if m[2] != "" {
		oldCount, _ = strconv.Atoi(m[2])
	}
	newStart, _ := strconv.Atoi(m[3])
	newCount := 1
	if m[4] != "" {
		newCount, _ = strconv.Atoi(m[4])
	}
	return hunk{oldStart: oldStart, oldCount: oldCount, newStart: newStart, newCount: newCount}, nil
}

func validateHunk(h hunk) error {
	oldCount, newCount := 0, 0
	for _, line := range h.lines {
		if line.op != '+' {
			oldCount++
		}
		if line.op != '-' {
			newCount++
		}
	}
	if oldCount != h.oldCount || newCount != h.newCount {
		return fmt.Errorf("hunk at old line %d has %d old/%d new lines, header declares %d old/%d new", h.oldStart, oldCount, newCount, h.oldCount, h.newCount)
	}
	return nil
}

func applyHunks(lines []string, hunks []hunk) ([]string, int, int, error) {
	const fuzz = 3
	result := append([]string(nil), lines...)
	offset, totalAdded, totalRemoved := 0, 0, 0

	for i, h := range hunks {
		oldLines := make([]string, 0, h.oldCount)
		for _, line := range h.lines {
			if line.op != '+' {
				oldLines = append(oldLines, line.content)
			}
		}
		target := h.oldStart - 1 + offset
		if h.oldCount == 0 {
			// For an insertion-only hunk, oldStart identifies the line after
			// which new content is inserted (zero means the start of the file).
			target = h.oldStart + offset
		}
		match := -1
		if matchAt(result, oldLines, target) {
			match = target
		} else {
			for delta := 1; delta <= fuzz; delta++ {
				if matchAt(result, oldLines, target-delta) {
					match = target - delta
					break
				}
				if matchAt(result, oldLines, target+delta) {
					match = target + delta
					break
				}
			}
		}
		if match < 0 {
			expected := oldLines
			if len(expected) > 3 {
				expected = expected[:3]
			}
			return nil, 0, 0, fmt.Errorf("hunk %d: context mismatch at line %d (expected: %q)", i+1, h.oldStart, expected)
		}

		newLines := make([]string, 0, h.newCount)
		added, removed := 0, 0
		for _, line := range h.lines {
			switch line.op {
			case ' ', '+':
				newLines = append(newLines, line.content)
				if line.op == '+' {
					added++
				}
			case '-':
				removed++
			}
		}
		tail := append([]string(nil), result[match+len(oldLines):]...)
		result = append(result[:match], append(newLines, tail...)...)
		offset += len(newLines) - len(oldLines)
		totalAdded += added
		totalRemoved += removed
	}
	return result, totalAdded, totalRemoved, nil
}

func matchAt(lines, expected []string, pos int) bool {
	if pos < 0 || pos+len(expected) > len(lines) {
		return false
	}
	for i := range expected {
		if lines[pos+i] != expected[i] {
			return false
		}
	}
	return true
}

func splitFileLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinFileLines(lines []string, lineEnding string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, lineEnding) + lineEnding
}

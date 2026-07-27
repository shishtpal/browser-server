package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
)

//go:embed schemas/multi_edit.json
var multiEditSchema []byte

const (
	maxMultiEditFiles = 50
	maxMultiEditOps   = 200
	maxEditFileSize   = 2 << 20
)

var multiEditMu sync.Mutex

// strictBool is a bool that rejects JSON null during unmarshalling.
// It ensures callers pass an explicit true/false rather than null or omission-with-null.
type strictBool bool

func (b *strictBool) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("must be a boolean")
	}
	return json.Unmarshal(data, (*bool)(b))
}

type multiEditOp struct {
	Path    string `json:"path"`
	Find    string `json:"find"`
	Replace string `json:"replace"`
	All     bool   `json:"all"`
}

type multiEditSnapshot struct {
	path         string
	original     []byte
	text         string
	newline      string
	mode         os.FileMode
	info         os.FileInfo
	edits        int
	replacements int
}

func registerMultiEdit(r *Registry) {
	r.add(Tool{
		Name:     "multi_edit",
		Category: "File Operations",
		Description: "Apply literal find/replace edits across existing files in one atomic transaction. " +
			"Each find must match exactly once unless all is true. If any edit fails, no file is modified. " +
			"Use read_file first to obtain exact content and write_file to create new files.",
		Schema:  json.RawMessage(multiEditSchema),
		Execute: multiEdit,
	})
}

func multiEdit(ctx context.Context, raw json.RawMessage) (any, error) {
	multiEditMu.Lock()
	defer multiEditMu.Unlock()

	var a struct {
		Edits  []json.RawMessage `json:"edits"`
		DryRun strictBool        `json:"dry_run"`
	}
	if err := strict(raw, &a, map[string]bool{"edits": true, "dry_run": true}); err != nil {
		return nil, err
	}
	if len(a.Edits) == 0 {
		return nil, fmt.Errorf("edits array must not be empty")
	}
	if len(a.Edits) > maxMultiEditOps {
		return nil, fmt.Errorf("too many edits: %d (max %d)", len(a.Edits), maxMultiEditOps)
	}

	edits := make([]multiEditOp, 0, len(a.Edits))
	for i, rawEdit := range a.Edits {
		var parsed struct {
			Path    string     `json:"path"`
			Find    string     `json:"find"`
			Replace *string    `json:"replace"`
			All     strictBool `json:"all"`
		}
		if err := strict(rawEdit, &parsed, map[string]bool{"path": true, "find": true, "replace": true, "all": true}); err != nil {
			return nil, fmt.Errorf("edit #%d: %w", i, err)
		}
		if parsed.Replace == nil {
			return nil, fmt.Errorf("edit #%d: replace is required", i)
		}
		op := multiEditOp{Path: parsed.Path, Find: parsed.Find, Replace: *parsed.Replace, All: bool(parsed.All)}
		if strings.TrimSpace(op.Path) == "" {
			return nil, fmt.Errorf("edit #%d: path is required", i)
		}
		if !filepath.IsAbs(op.Path) {
			return nil, fmt.Errorf("edit #%d: path must be absolute", i)
		}
		op.Path = filepath.Clean(op.Path)
		if op.Find == "" {
			return nil, fmt.Errorf("edit #%d: find must not be empty", i)
		}
		if op.Find == op.Replace {
			return nil, fmt.Errorf("edit #%d: find and replace are identical (no-op)", i)
		}
		edits = append(edits, op)
	}

	files := make(map[string]*multiEditSnapshot)
	ordered := make([]*multiEditSnapshot, 0)
	for i, op := range edits {
		key := multiEditPathKey(op.Path)
		if _, ok := files[key]; ok {
			continue
		}
		if len(files) >= maxMultiEditFiles {
			return nil, fmt.Errorf("too many distinct files: max %d", maxMultiEditFiles)
		}
		snapshot, err := loadForMultiEdit(op.Path, i)
		if err != nil {
			return nil, err
		}
		for _, existing := range ordered {
			if os.SameFile(existing.info, snapshot.info) {
				return nil, fmt.Errorf("edit #%d: path refers to a file already listed through another hard link", i)
			}
		}
		files[key] = snapshot
		ordered = append(ordered, snapshot)
	}

	for i, op := range edits {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		snapshot := files[multiEditPathKey(op.Path)]
		find := normalizeNewlines(op.Find)
		replacement := normalizeNewlines(op.Replace)
		matches := strings.Count(snapshot.text, find)
		if matches == 0 {
			return multiEditMatchError("no_match", i, op.Path, find, snapshot.text)
		}
		if matches > 1 && !op.All {
			return multiEditMatchError("ambiguous_match", i, op.Path, find, snapshot.text)
		}
		replacements := 1
		if op.All {
			replacements = matches
		}
		if multiEditResultTooLarge(len(snapshot.text), len(find), len(replacement), replacements) {
			return nil, fmt.Errorf("edit #%d: resulting file exceeds 2 MiB limit", i)
		}
		if op.All {
			snapshot.text = strings.ReplaceAll(snapshot.text, find, replacement)
			snapshot.replacements += matches
		} else {
			snapshot.text = strings.Replace(snapshot.text, find, replacement, 1)
			snapshot.replacements++
		}
		snapshot.edits++
	}

	summaries := make([]map[string]any, 0, len(ordered))
	totalReplacements := 0
	for _, snapshot := range ordered {
		if !snapshot.changed() {
			continue
		}
		summaries = append(summaries, map[string]any{
			"path": snapshot.path, "edits": snapshot.edits, "replacements": snapshot.replacements,
		})
		totalReplacements += snapshot.replacements
	}

	response := map[string]any{
		"ok":                 true,
		"files_changed":      len(summaries),
		"total_replacements": totalReplacements,
		"files":              summaries,
	}
	if bool(a.DryRun) {
		diffs := make([]string, 0, len(summaries))
		for _, snapshot := range ordered {
			if snapshot.changed() {
				diffs = append(diffs, buildMultiEditDiff(snapshot))
			}
		}
		response["dry_run"] = true
		response["diff"] = strings.Join(diffs, "")
		encoded, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}
		if len(encoded) > maxOutput {
			return map[string]any{
				"ok": false, "error": "diff_too_large",
				"message": fmt.Sprintf("Dry-run diff exceeds the %d-byte tool output limit.", maxOutput),
			}, nil
		}
		return response, nil
	}
	if err := commitMultiEditFiles(ctx, ordered); err != nil {
		return nil, err
	}
	return response, nil
}

func multiEditPathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func multiEditResultTooLarge(current, find, replacement, count int) bool {
	if replacement <= find {
		return false
	}
	growth := replacement - find
	return current > maxEditFileSize || count > (maxEditFileSize-current)/growth
}

func loadForMultiEdit(path string, opIndex int) (*multiEditSnapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("edit #%d: file not found: %s", opIndex, path)
		}
		return nil, fmt.Errorf("edit #%d: failed to inspect %s: %w", opIndex, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("edit #%d: cannot edit a symbolic link", opIndex)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("edit #%d: path must be a regular file", opIndex)
	}
	if info.Size() > maxEditFileSize {
		return nil, fmt.Errorf("edit #%d: file exceeds 2 MiB limit", opIndex)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("edit #%d: failed to read %s: %w", opIndex, path, err)
	}
	if len(data) > maxEditFileSize {
		return nil, fmt.Errorf("edit #%d: file exceeds 2 MiB limit", opIndex)
	}
	prefix := data
	if len(prefix) > 8192 {
		prefix = prefix[:8192]
	}
	if bytes.IndexByte(prefix, 0) >= 0 {
		return nil, fmt.Errorf("edit #%d: file appears to be binary", opIndex)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("edit #%d: file is not valid UTF-8", opIndex)
	}

	text := string(data)
	crlf := strings.Count(text, "\r\n")
	lf := strings.Count(text, "\n") - crlf
	newline := "\n"
	if crlf > lf {
		newline = "\r\n"
	}
	return &multiEditSnapshot{
		path: path, original: data, text: normalizeNewlines(text), newline: newline, mode: info.Mode().Perm(), info: info,
	}, nil
}

func normalizeNewlines(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func (s *multiEditSnapshot) output() []byte {
	text := s.text
	if s.newline == "\r\n" {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	return []byte(text)
}

func (s *multiEditSnapshot) changed() bool {
	return !bytes.Equal(s.original, s.output())
}

func multiEditMatchError(code string, opIndex int, path, find, fileText string) (any, error) {
	response := map[string]any{"ok": false, "error": code, "op": opIndex, "path": path}
	switch code {
	case "no_match":
		response["message"] = fmt.Sprintf("Text not found in %s. Re-read the file and copy exact text.", path)
		response["hint"] = "Use read_file to get the current content, then copy the exact text including indentation."
		if nearest := findNearestMultiEditLine(fileText, find); nearest != nil {
			response["nearest"] = nearest
		}
	case "ambiguous_match":
		matches := strings.Count(fileText, find)
		response["found"] = matches
		response["message"] = fmt.Sprintf("Found %d matches in %s; expected exactly 1.", matches, path)
		response["hint"] = "Add surrounding context lines to make `find` unique, or set \"all\": true."
	}
	return response, nil
}

func findNearestMultiEditLine(fileText, find string) map[string]any {
	target := strings.TrimSpace(strings.Split(find, "\n")[0])
	if target == "" {
		return nil
	}
	bestLine, bestText, bestScore := 0, "", -1.0
	for i, line := range strings.Split(fileText, "\n") {
		score := diceSimilarity(target, strings.TrimSpace(line))
		if score > bestScore {
			bestLine, bestText, bestScore = i+1, line, score
		}
	}
	if bestLine == 0 {
		return nil
	}
	bestText = truncateRunes(bestText, 256)
	return map[string]any{"line": bestLine, "content": bestText, "similarity": bestScore}
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}

func diceSimilarity(a, b string) float64 {
	if a == b {
		return 1
	}
	ar, br := []rune(a), []rune(b)
	if len(ar) < 2 || len(br) < 2 {
		return 0
	}
	pairs := make(map[string]int, len(ar)-1)
	for i := 0; i < len(ar)-1; i++ {
		pairs[string(ar[i:i+2])]++
	}
	intersection := 0
	for i := 0; i < len(br)-1; i++ {
		pair := string(br[i : i+2])
		if pairs[pair] > 0 {
			intersection++
			pairs[pair]--
		}
	}
	return 2 * float64(intersection) / float64(len(ar)+len(br)-2)
}

func buildMultiEditDiff(snapshot *multiEditSnapshot) string {
	oldLines, oldTrailing := multiEditDiffLines(normalizeNewlines(string(snapshot.original)))
	newLines, newTrailing := multiEditDiffLines(snapshot.text)
	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	trailingOnlyChange := prefix == len(oldLines) && prefix == len(newLines) && oldTrailing != newTrailing
	if trailingOnlyChange && prefix > 0 {
		prefix--
	}
	suffix := 0
	for !trailingOnlyChange && suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}
	start := prefix - 3
	if start < 0 {
		start = 0
	}
	oldEnd, newEnd := len(oldLines)-suffix+3, len(newLines)-suffix+3
	if oldEnd > len(oldLines) {
		oldEnd = len(oldLines)
	}
	if newEnd > len(newLines) {
		newEnd = len(newLines)
	}

	var diff strings.Builder
	fmt.Fprintf(&diff, "--- %s\n+++ %s\n", snapshot.path, snapshot.path)
	fmt.Fprintf(&diff, "@@ -%s +%s @@\n", multiEditDiffRange(start, oldEnd-start), multiEditDiffRange(start, newEnd-start))
	commonPrefixEnd := prefix
	if commonPrefixEnd > oldEnd || commonPrefixEnd > newEnd {
		commonPrefixEnd = min(oldEnd, newEnd)
	}
	for _, line := range oldLines[start:commonPrefixEnd] {
		fmt.Fprintf(&diff, " %s\n", line)
	}
	for _, line := range oldLines[commonPrefixEnd : len(oldLines)-suffix] {
		fmt.Fprintf(&diff, "-%s\n", line)
	}
	if !oldTrailing && len(oldLines)-suffix == len(oldLines) && len(oldLines) > commonPrefixEnd {
		diff.WriteString("\\ No newline at end of file\n")
	}
	for _, line := range newLines[commonPrefixEnd : len(newLines)-suffix] {
		fmt.Fprintf(&diff, "+%s\n", line)
	}
	if !newTrailing && len(newLines)-suffix == len(newLines) && len(newLines) > commonPrefixEnd {
		diff.WriteString("\\ No newline at end of file\n")
	}
	oldSuffixStart, newSuffixStart := len(oldLines)-suffix, len(newLines)-suffix
	for oldSuffixStart < oldEnd && newSuffixStart < newEnd {
		fmt.Fprintf(&diff, " %s\n", oldLines[oldSuffixStart])
		if oldSuffixStart == len(oldLines)-1 && !oldTrailing && newSuffixStart == len(newLines)-1 && !newTrailing {
			diff.WriteString("\\ No newline at end of file\n")
		}
		oldSuffixStart++
		newSuffixStart++
	}
	return diff.String()
}

func multiEditDiffRange(start, count int) string {
	if count == 0 {
		return fmt.Sprintf("%d,0", start)
	}
	return fmt.Sprintf("%d,%d", start+1, count)
}

func multiEditDiffLines(text string) ([]string, bool) {
	if text == "" {
		return nil, false
	}
	trailing := strings.HasSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	if trailing {
		lines = lines[:len(lines)-1]
	}
	return lines, trailing
}

type stagedMultiEditFile struct {
	snapshot   *multiEditSnapshot
	tempPath   string
	backupPath string
}

func commitMultiEditFiles(ctx context.Context, files []*multiEditSnapshot) error {
	return commitMultiEditFilesWithRename(ctx, files, os.Rename)
}

func commitMultiEditFilesWithRename(ctx context.Context, files []*multiEditSnapshot, rename func(string, string) error) error {
	staged := make([]stagedMultiEditFile, 0, len(files))
	// cleanup removes all temp and backup files for staged entries.
	// Safe to call after a successful commit — os.Remove no-ops on already-renamed temps
	// and backup files that were consumed during rollback.
	cleanup := func() {
		for _, file := range staged {
			_ = os.Remove(file.tempPath)
			_ = os.Remove(file.backupPath)
		}
	}
	for _, snapshot := range files {
		if !snapshot.changed() {
			continue
		}
		if err := ctx.Err(); err != nil {
			cleanup()
			return err
		}
		tempPath, err := stageMultiEditFile(snapshot, snapshot.output(), ".multi_edit_new_*")
		if err != nil {
			cleanup()
			return fmt.Errorf("write failed on %s: %w", snapshot.path, err)
		}
		backupPath, err := stageMultiEditFile(snapshot, snapshot.original, ".multi_edit_backup_*")
		if err != nil {
			_ = os.Remove(tempPath)
			cleanup()
			return fmt.Errorf("write failed on %s: %w", snapshot.path, err)
		}
		staged = append(staged, stagedMultiEditFile{snapshot: snapshot, tempPath: tempPath, backupPath: backupPath})
	}

	if err := ctx.Err(); err != nil {
		cleanup()
		return err
	}
	for _, file := range staged {
		if err := revalidateMultiEditSnapshot(file.snapshot); err != nil {
			cleanup()
			return err
		}
	}

	written := make([]stagedMultiEditFile, 0, len(staged))
	for _, file := range staged {
		if err := rename(file.tempPath, file.snapshot.path); err != nil {
			rollbackErr := rollbackMultiEditFiles(written, rename)
			if rollbackErr != nil {
				for _, stagedFile := range staged {
					_ = os.Remove(stagedFile.tempPath)
				}
				for i := len(written); i < len(staged); i++ {
					_ = os.Remove(staged[i].backupPath)
				}
				return fmt.Errorf("rename failed on %s: %w (rollback failed: %v)", file.snapshot.path, err, rollbackErr)
			}
			cleanup()
			return fmt.Errorf("rename failed on %s: %w", file.snapshot.path, err)
		}
		written = append(written, file)
	}
	cleanup()
	return nil
}

func stageMultiEditFile(snapshot *multiEditSnapshot, content []byte, pattern string) (path string, err error) {
	temp, err := os.CreateTemp(filepath.Dir(snapshot.path), pattern)
	if err != nil {
		return "", err
	}
	path = temp.Name()
	defer func() {
		if closeErr := temp.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	if err = temp.Chmod(snapshot.mode); err != nil {
		return "", err
	}
	if _, err = temp.Write(content); err != nil {
		return "", err
	}
	if err = temp.Sync(); err != nil {
		return "", err
	}
	return path, nil
}

func revalidateMultiEditSnapshot(snapshot *multiEditSnapshot) error {
	info, err := os.Lstat(snapshot.path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !os.SameFile(snapshot.info, info) {
		return fmt.Errorf("conflict on %s: file identity changed after it was read", snapshot.path)
	}
	content, err := os.ReadFile(snapshot.path)
	if err != nil {
		return fmt.Errorf("conflict on %s: failed to re-read file: %w", snapshot.path, err)
	}
	if !bytes.Equal(content, snapshot.original) {
		return fmt.Errorf("conflict on %s: file content changed after it was read", snapshot.path)
	}
	return nil
}

func rollbackMultiEditFiles(written []stagedMultiEditFile, rename func(string, string) error) error {
	var rollbackErr error
	for i := len(written) - 1; i >= 0; i-- {
		file := written[i]
		if err := rename(file.backupPath, file.snapshot.path); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("restore %s: %w", file.snapshot.path, err)
		}
	}
	return rollbackErr
}

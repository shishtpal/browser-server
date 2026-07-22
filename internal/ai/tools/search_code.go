package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const searchCodeSchema = `{"type":"object","properties":{"pattern":{"type":"string","maxLength":500},"path":{"type":"string"},"include":{"type":"array","items":{"type":"string"}},"exclude":{"type":"array","items":{"type":"string"}},"case_sensitive":{"type":"boolean"},"whole_word":{"type":"boolean"},"max_results":{"type":"integer","minimum":1,"maximum":100},"context_lines":{"type":"integer","minimum":0,"maximum":10},"type":{"type":"string","enum":["regex","literal","fixed"]}},"required":["pattern"],"additionalProperties":false}`

func registerSearchCode(r *Registry) {
	r.add(Tool{
		Name:        "search_code",
		Category:    "Code Intelligence",
		Description: "Search source files using regex, literal, or fixed-string matching",
		Schema:      json.RawMessage(searchCodeSchema),
		Execute:     searchCode,
	})
}

func searchCode(ctx context.Context, raw json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, codeToolTimeout)
	defer cancel()
	var a struct {
		Pattern       string   `json:"pattern"`
		Path          string   `json:"path"`
		Type          string   `json:"type"`
		Include       []string `json:"include"`
		Exclude       []string `json:"exclude"`
		CaseSensitive bool     `json:"case_sensitive"`
		WholeWord     bool     `json:"whole_word"`
		MaxResults    int      `json:"max_results"`
		ContextLines  *int     `json:"context_lines"`
	}
	if err := strict(raw, &a, map[string]bool{"pattern": true, "path": true, "include": true, "exclude": true, "case_sensitive": true, "whole_word": true, "max_results": true, "context_lines": true, "type": true}); err != nil {
		return nil, err
	}
	if a.Pattern == "" || len(a.Pattern) > 500 {
		return nil, fmt.Errorf("pattern is required and must not exceed 500 characters")
	}
	if a.Path == "" {
		a.Path = "."
	}
	if a.Type == "" {
		a.Type = "regex"
	}
	if a.Type != "regex" && a.Type != "literal" && a.Type != "fixed" {
		return nil, fmt.Errorf("type must be regex, literal, or fixed")
	}
	if a.MaxResults == 0 {
		a.MaxResults = 50
	}
	if a.MaxResults < 1 || a.MaxResults > 100 {
		return nil, fmt.Errorf("max_results must be 1 to 100")
	}
	contextLines := 2
	if a.ContextLines != nil {
		contextLines = *a.ContextLines
	}
	if contextLines < 0 || contextLines > 10 {
		return nil, fmt.Errorf("context_lines must be 0 to 10")
	}
	if err := validateGlobs(a.Include, a.Exclude); err != nil {
		return nil, err
	}
	pattern := a.Pattern
	if a.Type != "regex" {
		pattern = regexp.QuoteMeta(pattern)
	}
	if a.WholeWord {
		pattern = `\b(?:` + pattern + `)\b`
	}
	if !a.CaseSensitive {
		pattern = `(?i)` + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	type match struct {
		File          string   `json:"file"`
		Line          int      `json:"line"`
		Column        int      `json:"column"`
		Match         string   `json:"match"`
		ContextBefore []string `json:"context_before"`
		ContextAfter  []string `json:"context_after"`
	}
	start := time.Now()
	matches := []match{}
	total := 0
	truncated := false
	err = filepath.WalkDir(a.Path, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(a.Path, path)
			if path != a.Path && (d.Name() == ".git" || d.Name() == "node_modules" || globMatch(a.Exclude, rel) || globMatch(a.Exclude, rel+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(a.Path, path)
		if len(a.Include) > 0 && !globMatch(a.Include, rel) || globMatch(a.Exclude, rel) {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if info.Size() > maxSourceSize {
			truncated = true
			return nil
		}
		file, e := os.Open(path)
		if e != nil {
			return e
		}
		data, e := io.ReadAll(io.LimitReader(file, maxSourceSize+1))
		_ = file.Close()
		if e != nil {
			return e
		}
		if len(data) > maxSourceSize {
			truncated = true
			return nil
		}
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			for _, loc := range re.FindAllStringIndex(line, -1) {
				total++
				if len(matches) >= a.MaxResults {
					truncated = true
					continue
				}
				lo := i - contextLines
				if lo < 0 {
					lo = 0
				}
				hi := i + contextLines + 1
				if hi > len(lines) {
					hi = len(lines)
				}
				m := match{
					File:          path,
					Line:          i + 1,
					Column:        loc[0] + 1,
					Match:         line[loc[0]:loc[1]],
					ContextBefore: append([]string{}, lines[lo:i]...),
					ContextAfter:  append([]string{}, lines[i+1:hi]...),
				}
				probe, _ := json.Marshal(map[string]any{"matches": append(matches, m), "total_matches": total, "truncated": true, "search_time_ms": 0})
				if len(probe) > maxOutput-resultHeadroom {
					truncated = true
					continue
				}
				matches = append(matches, m)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"matches": matches, "total_matches": total, "truncated": truncated, "search_time_ms": time.Since(start).Milliseconds()}, nil
}

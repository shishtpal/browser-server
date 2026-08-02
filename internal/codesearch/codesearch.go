package codesearch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"browser-server/internal/searchengine"
)

var errCandidateCap = errors.New("code search candidate cap reached")

// Match is one code-search match location.
type Match struct {
	File          string   `json:"file"`
	Line          int      `json:"line"`
	Column        int      `json:"column"`
	Match         string   `json:"match"`
	ContextBefore []string `json:"context_before"`
	ContextAfter  []string `json:"context_after"`
}

// Options configures the code search walk.
type Options struct {
	Root          string
	Pattern       string
	Type          string // regex, literal, fixed
	Include       []string
	Exclude       []string
	CaseSensitive bool
	WholeWord     bool
	ContextLines  int
	MaxSourceSize int64
	MaxCandidates int
}

// CandidateSet runs the filesystem traversal and returns pre-matched candidates
// for the search engine's exact strategy. It is intentionally separated from
// the AI tool so it can be unit-tested independently.
func CandidateSet(ctx context.Context, opts Options) (searchengine.CandidateSet[Match], error) {
	re, err := compilePattern(opts)
	if err != nil {
		return searchengine.CandidateSet[Match]{}, err
	}
	if opts.ContextLines < 0 {
		opts.ContextLines = 2
	}
	if opts.ContextLines > 10 {
		opts.ContextLines = 10
	}
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.MaxSourceSize == 0 {
		opts.MaxSourceSize = 8 << 20
	}

	var candidates []searchengine.Candidate[Match]
	truncated := false
	err = filepath.WalkDir(opts.Root, func(pathName string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(opts.Root, pathName)
			if pathName != opts.Root && (d.Name() == ".git" || d.Name() == "node_modules" || globMatch(opts.Exclude, rel) || globMatch(opts.Exclude, rel+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(opts.Root, pathName)
		if len(opts.Include) > 0 && !globMatch(opts.Include, rel) || globMatch(opts.Exclude, rel) {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if info.Size() > opts.MaxSourceSize {
			truncated = true
			return nil
		}
		file, e := os.Open(pathName)
		if e != nil {
			return e
		}
		data, e := io.ReadAll(io.LimitReader(file, opts.MaxSourceSize+1))
		_ = file.Close()
		if e != nil {
			return e
		}
		if len(data) > int(opts.MaxSourceSize) || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			for _, loc := range re.FindAllStringIndex(line, -1) {
				if opts.MaxCandidates > 0 && len(candidates) >= opts.MaxCandidates {
					truncated = true
					return errCandidateCap
				}
				match := makeMatch(pathName, lines, i, loc, opts.ContextLines)
				candidates = append(candidates, searchengine.Candidate[Match]{
					Key: pathName + ":" + fmt.Sprint(match.Line) + ":" + fmt.Sprint(match.Column),
					Fields: []searchengine.Field{
						{Name: "file", Text: pathName, Weight: 1},
					},
					Value: match,
					// BaseScore reflects overall match quality: exact whole-word/literal
					// matches score highest; generic regex matches score lower.
					BaseScore:  scoreMatch(opts, line, loc),
					SourceRank: len(candidates),
				})
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, errCandidateCap) {
		return searchengine.CandidateSet[Match]{}, err
	}
	return searchengine.CandidateSet[Match]{Candidates: candidates, Truncated: truncated}, nil
}

func compilePattern(opts Options) (*regexp.Regexp, error) {
	pattern := opts.Pattern
	if opts.Type != "regex" {
		pattern = regexp.QuoteMeta(pattern)
	}
	if opts.WholeWord {
		pattern = `\b(?:` + pattern + `)\b`
	}
	if !opts.CaseSensitive {
		pattern = `(?i)` + pattern
	}
	return regexp.Compile(pattern)
}

func makeMatch(pathName string, lines []string, lineIndex int, loc []int, contextLines int) Match {
	lo := lineIndex - contextLines
	if lo < 0 {
		lo = 0
	}
	hi := lineIndex + contextLines + 1
	if hi > len(lines) {
		hi = len(lines)
	}
	return Match{
		File:          pathName,
		Line:          lineIndex + 1,
		Column:        loc[0] + 1,
		Match:         lines[lineIndex][loc[0]:loc[1]],
		ContextBefore: append([]string{}, lines[lo:lineIndex]...),
		ContextAfter:  append([]string{}, lines[lineIndex+1:hi]...),
	}
}

func scoreMatch(opts Options, line string, loc []int) float64 {
	if opts.Type != "regex" {
		if loc[0] == 0 && loc[1] == len(line) {
			return 1
		}
		leftBoundary := loc[0] == 0 || !isWordByte(line[loc[0]-1])
		rightBoundary := loc[1] == len(line) || !isWordByte(line[loc[1]])
		if leftBoundary && rightBoundary {
			return 0.9
		}
		if leftBoundary {
			return 0.8
		}
		return 0.7
	}
	if opts.WholeWord {
		return 0.8
	}
	return 0.6
}

func isWordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

func globMatch(patterns []string, rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		p = filepath.ToSlash(p)
		target := path.Base(rel)
		if strings.Contains(p, "/") {
			target = rel
		}
		if ok, _ := path.Match(p, target); ok {
			return true
		}
	}
	return false
}

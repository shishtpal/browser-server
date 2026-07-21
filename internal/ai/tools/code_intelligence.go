package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	codeToolTimeout  = 30 * time.Second
	maxSourceSize    = 8 << 20
	maxCommandOutput = 256 << 10
	resultHeadroom   = 2048
)

const searchCodeSchema = `{"type":"object","properties":{"pattern":{"type":"string","maxLength":500},"path":{"type":"string"},"include":{"type":"array","items":{"type":"string"}},"exclude":{"type":"array","items":{"type":"string"}},"case_sensitive":{"type":"boolean"},"whole_word":{"type":"boolean"},"max_results":{"type":"integer","minimum":1,"maximum":100},"context_lines":{"type":"integer","minimum":0,"maximum":10},"type":{"type":"string","enum":["regex","literal","fixed"]}},"required":["pattern"],"additionalProperties":false}`
const analyzeCodeSchema = `{"type":"object","properties":{"path":{"type":"string"},"include_patterns":{"type":"array","items":{"type":"string"}},"exclude_patterns":{"type":"array","items":{"type":"string"}},"analysis_type":{"type":"string","enum":["symbols","imports","exports","functions","types","all"]},"include_private":{"type":"boolean"},"max_depth":{"type":"integer","minimum":1,"maximum":10},"max_results":{"type":"integer","minimum":1,"maximum":500}},"required":["path"],"additionalProperties":false}`
const getDiagnosticsSchema = `{"type":"object","properties":{"path":{"type":"string"},"language":{"type":"string","enum":["go","typescript","javascript","python","rust","java","auto"]},"severity":{"type":"string","enum":["error","warning","info","hint","all"]},"include_related":{"type":"boolean"},"max_results":{"type":"integer","minimum":1,"maximum":200}},"required":["path"],"additionalProperties":false}`

func validateGlobs(patterns ...[]string) error {
	for _, set := range patterns {
		for _, pattern := range set {
			if _, err := path.Match(filepath.ToSlash(pattern), ""); err != nil {
				return fmt.Errorf("invalid glob %q: %w", pattern, err)
			}
		}
	}
	return nil
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
				m := match{File: path, Line: i + 1, Column: loc[0] + 1, Match: line[loc[0]:loc[1]], ContextBefore: append([]string{}, lines[lo:i]...), ContextAfter: append([]string{}, lines[i+1:hi]...)}
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

func nodeText(fset *token.FileSet, n any) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, n)
	return b.String()
}
func docText(d *ast.CommentGroup) string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(d.Text())
}

func analyzeCode(ctx context.Context, raw json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, codeToolTimeout)
	defer cancel()
	var a struct {
		Path     string   `json:"path"`
		Include  []string `json:"include_patterns"`
		Exclude  []string `json:"exclude_patterns"`
		Analysis string   `json:"analysis_type"`
		Private  bool     `json:"include_private"`
		Depth    int      `json:"max_depth"`
		Max      int      `json:"max_results"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "include_patterns": true, "exclude_patterns": true, "analysis_type": true, "include_private": true, "max_depth": true, "max_results": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if a.Analysis == "" {
		a.Analysis = "all"
	}
	valid := map[string]bool{"symbols": true, "imports": true, "exports": true, "functions": true, "types": true, "all": true}
	if !valid[a.Analysis] {
		return nil, fmt.Errorf("invalid analysis_type")
	}
	if a.Depth == 0 {
		a.Depth = 3
	}
	if a.Depth < 1 || a.Depth > 10 {
		return nil, fmt.Errorf("max_depth must be 1 to 10")
	}
	if a.Max == 0 {
		a.Max = 200
	}
	if a.Max < 1 || a.Max > 500 {
		return nil, fmt.Errorf("max_results must be 1 to 500")
	}
	if len(a.Include) == 0 {
		a.Include = []string{"*.go"}
	}
	if err := validateGlobs(a.Include, a.Exclude); err != nil {
		return nil, err
	}
	info, err := os.Stat(a.Path)
	if err != nil {
		return nil, err
	}
	root := a.Path
	if !info.IsDir() {
		root = filepath.Dir(a.Path)
	}
	paths := []string{}
	err = filepath.WalkDir(a.Path, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !info.IsDir() && path == a.Path {
			rel := filepath.Base(path)
			if filepath.Ext(path) != ".go" {
				return fmt.Errorf("explicit file must be a .go file")
			}
			if globMatch(a.Include, rel) && !globMatch(a.Exclude, rel) {
				paths = append(paths, path)
			}
			return nil
		}
		if d.IsDir() {
			if path != a.Path {
				rel, _ := filepath.Rel(root, path)
				if strings.Count(filepath.ToSlash(rel), "/")+1 > a.Depth {
					return filepath.SkipDir
				}
				if d.Name() == ".git" || d.Name() == "node_modules" || globMatch(a.Exclude, rel) || globMatch(a.Exclude, rel+"/") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if filepath.Ext(path) == ".go" && globMatch(a.Include, rel) && !globMatch(a.Exclude, rel) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	files := []map[string]any{}
	remaining := a.Max
	funcs, typesN, importsN := 0, 0, 0
	filesAnalyzed := 0
	truncated := false
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, e := os.Stat(path)
		if e != nil {
			return nil, e
		}
		if info.Size() > maxSourceSize {
			truncated = true
			continue
		}
		fset := token.NewFileSet()
		f, e := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if e != nil {
			return nil, fmt.Errorf("parse %s: %w", path, e)
		}
		filesAnalyzed++
		imps := []map[string]any{}
		if a.Analysis == "all" || a.Analysis == "imports" {
			for _, i := range f.Imports {
				alias := ""
				if i.Name != nil {
					alias = i.Name.Name
				}
				p, _ := strconv.Unquote(i.Path.Value)
				imps = append(imps, map[string]any{"path": p, "alias": alias})
				importsN++
			}
		}
		syms := []map[string]any{}
		add := func(s map[string]any) {
			if remaining > 0 {
				syms = append(syms, s)
				remaining--
			} else {
				truncated = true
			}
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				exp := ast.IsExported(d.Name.Name)
				if !a.Private && !exp {
					continue
				}
				if a.Analysis != "all" && a.Analysis != "symbols" && a.Analysis != "exports" && a.Analysis != "functions" {
					continue
				}
				if a.Analysis == "exports" && !exp {
					continue
				}
				kind := "function"
				if d.Recv != nil {
					kind = "method"
				}
				s := map[string]any{"name": d.Name.Name, "type": kind, "signature": nodeText(fset, d.Type), "exports": exp, "line": fset.Position(d.Pos()).Line, "end_line": fset.Position(d.End()).Line, "doc": docText(d.Doc)}
				if d.Recv != nil {
					s["receiver"] = nodeText(fset, d.Recv)
				}
				add(s)
				funcs++
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch x := sp.(type) {
					case *ast.TypeSpec:
						exp := ast.IsExported(x.Name.Name)
						if !a.Private && !exp {
							continue
						}
						if a.Analysis != "all" && a.Analysis != "symbols" && a.Analysis != "exports" && a.Analysis != "types" {
							continue
						}
						if a.Analysis == "exports" && !exp {
							continue
						}
						kind := "type"
						s := map[string]any{"name": x.Name.Name, "type": kind, "signature": nodeText(fset, x), "exports": exp, "line": fset.Position(x.Pos()).Line, "end_line": fset.Position(x.End()).Line, "doc": docText(d.Doc)}
						if st, ok := x.Type.(*ast.StructType); ok {
							kind = "struct"
							s["type"] = kind
							fields := []map[string]any{}
							for _, fl := range st.Fields.List {
								names := []string{""}
								if len(fl.Names) > 0 {
									names = nil
									for _, n := range fl.Names {
										names = append(names, n.Name)
									}
								}
								for _, n := range names {
									tag := ""
									if fl.Tag != nil {
										tag = fl.Tag.Value
									}
									fields = append(fields, map[string]any{"name": n, "type": nodeText(fset, fl.Type), "tag": tag})
								}
							}
							s["fields"] = fields
						}
						add(s)
						typesN++
					case *ast.ValueSpec:
						if a.Analysis != "all" && a.Analysis != "symbols" && a.Analysis != "exports" {
							continue
						}
						kind := strings.ToLower(d.Tok.String())
						for _, n := range x.Names {
							exp := ast.IsExported(n.Name)
							if !a.Private && !exp || a.Analysis == "exports" && !exp {
								continue
							}
							add(map[string]any{"name": n.Name, "type": kind, "signature": nodeText(fset, x), "exports": exp, "line": fset.Position(n.Pos()).Line, "end_line": fset.Position(x.End()).Line, "doc": docText(d.Doc)})
						}
					}
				}
			}
		}
		entry := map[string]any{"file": path, "package": f.Name.Name, "imports": imps, "symbols": syms}
		probe, _ := json.Marshal(map[string]any{"files": append(files, entry), "summary": map[string]any{"files_analyzed": filesAnalyzed, "total_functions": funcs, "total_types": typesN, "total_imports": importsN}, "truncated": true})
		if len(probe) > maxOutput-resultHeadroom {
			truncated = true
			continue
		}
		files = append(files, entry)
	}
	return map[string]any{"files": files, "summary": map[string]any{"files_analyzed": filesAnalyzed, "total_functions": funcs, "total_types": typesN, "total_imports": importsN}, "truncated": truncated}, nil
}

var diagnosticLine = regexp.MustCompile(`^(.*):(\d+):(\d+):\s*(.*)$`)

type diagnosticPoint struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type diagnosticRange struct {
	Start diagnosticPoint `json:"start"`
	End   diagnosticPoint `json:"end"`
}
type codeDiagnostic struct {
	File     string          `json:"file"`
	Range    diagnosticRange `json:"range"`
	Severity string          `json:"severity"`
	Source   string          `json:"source"`
	Message  string          `json:"message"`
}

type cappedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.Len()
	if remaining := b.limit - b.Len(); remaining > 0 {
		if remaining < len(p) {
			_, _ = b.Buffer.Write(p[:remaining])
		} else {
			_, _ = b.Buffer.Write(p)
		}
	}
	if n > remaining {
		b.truncated = true
	}
	return n, nil
}

func severityAllowed(threshold, severity string) bool {
	rank := map[string]int{"error": 0, "warning": 1, "info": 2, "hint": 3}
	if threshold == "all" {
		threshold = "hint"
	}
	return rank[severity] <= rank[threshold]
}

func parseDiagnostic(line, severity, source string) (codeDiagnostic, bool) {
	m := diagnosticLine.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return codeDiagnostic{}, false
	}
	ln, _ := strconv.Atoi(m[2])
	col, _ := strconv.Atoi(m[3])
	if ln > 0 {
		ln--
	}
	if col > 0 {
		col--
	}
	p := diagnosticPoint{Line: ln, Character: col}
	return codeDiagnostic{File: m[1], Range: diagnosticRange{Start: p, End: p}, Severity: severity, Source: source, Message: m[4]}, true
}

func canonicalDiagnosticFile(dir, name string) string {
	name = filepath.FromSlash(name)
	if !filepath.IsAbs(name) {
		name = filepath.Join(dir, name)
	}
	value, err := filepath.Abs(name)
	if err != nil {
		return filepath.Clean(name)
	}
	return filepath.Clean(value)
}

func getDiagnostics(ctx context.Context, raw json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, codeToolTimeout)
	defer cancel()
	var a struct {
		Path           string `json:"path"`
		Language       string `json:"language"`
		Severity       string `json:"severity"`
		IncludeRelated bool   `json:"include_related"`
		Max            int    `json:"max_results"`
	}
	if err := strict(raw, &a, map[string]bool{"path": true, "language": true, "severity": true, "include_related": true, "max_results": true}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if a.Language == "" {
		a.Language = "auto"
	}
	if a.Language != "go" && a.Language != "auto" {
		return nil, fmt.Errorf("language %q is not supported in Phase 1; only go and auto are supported", a.Language)
	}
	if a.Severity == "" {
		a.Severity = "warning"
	}
	if !map[string]bool{"error": true, "warning": true, "info": true, "hint": true, "all": true}[a.Severity] {
		return nil, fmt.Errorf("invalid severity")
	}
	if a.Max == 0 {
		a.Max = 100
	}
	if a.Max < 1 || a.Max > 200 {
		return nil, fmt.Errorf("max_results must be 1 to 200")
	}
	st, err := os.Stat(a.Path)
	if err != nil {
		return nil, err
	}
	dir := a.Path
	target := "./..."
	if !st.IsDir() {
		if filepath.Ext(a.Path) != ".go" && a.Language == "auto" {
			return nil, fmt.Errorf("could not auto-detect Go from path")
		}
		dir = filepath.Dir(a.Path)
		target = "."
	}
	diags := []codeDiagnostic{}
	counts := map[string]int{"errors": 0, "warnings": 0, "info": 0, "hints": 0}
	truncated := false
	var wantedFile string
	if !st.IsDir() {
		wantedFile, _ = filepath.Abs(a.Path)
		wantedFile = filepath.Clean(wantedFile)
	}
	buildDiagnostics := 0
	for _, run := range []struct{ name, severity string }{{"build", "error"}, {"vet", "warning"}} {
		if run.name == "vet" && buildDiagnostics > 0 {
			break
		}
		cmd := exec.CommandContext(ctx, "go", run.name, target)
		cmd.Dir = dir
		out := &cappedBuffer{limit: maxCommandOutput}
		cmd.Stdout, cmd.Stderr = out, out
		e := cmd.Run()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if e != nil {
			if _, ok := e.(*exec.ExitError); !ok {
				return nil, fmt.Errorf("start go %s: %w", run.name, e)
			}
		}
		truncated = truncated || out.truncated
		positioned := 0
		for _, line := range strings.Split(out.String(), "\n") {
			d, ok := parseDiagnostic(line, run.severity, "go"+run.name)
			if !ok {
				continue
			}
			if wantedFile != "" && !strings.EqualFold(canonicalDiagnosticFile(dir, d.File), wantedFile) {
				continue
			}
			positioned++
			counts[run.severity+"s"]++
			if !severityAllowed(a.Severity, run.severity) {
				continue
			}
			probe, _ := json.Marshal(append(diags, d))
			if len(diags) >= a.Max || len(probe) > maxOutput-resultHeadroom {
				truncated = true
				continue
			}
			diags = append(diags, d)
		}
		if run.name == "build" {
			buildDiagnostics = positioned
		}
		if e != nil && positioned == 0 {
			message := strings.TrimSpace(out.String())
			if message == "" {
				message = e.Error()
			}
			if len(message) > maxOutput/2 {
				message = string([]byte(message)[:maxOutput/2])
				truncated = true
			}
			d := codeDiagnostic{File: a.Path, Range: diagnosticRange{}, Severity: run.severity, Source: "go" + run.name, Message: message}
			counts[run.severity+"s"]++
			probe, _ := json.Marshal(append(diags, d))
			if severityAllowed(a.Severity, run.severity) && len(diags) < a.Max && len(probe) <= maxOutput-resultHeadroom {
				diags = append(diags, d)
			} else if severityAllowed(a.Severity, run.severity) {
				truncated = true
			}
			if run.name == "build" {
				buildDiagnostics = 1
			}
		}
	}
	checked := 0
	if !st.IsDir() {
		checked = 1
	} else {
		_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, e error) error {
			if e != nil {
				return nil
			}
			if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			if !d.IsDir() && filepath.Ext(d.Name()) == ".go" {
				checked++
			}
			return nil
		})
	}
	return map[string]any{"diagnostics": diags, "summary": map[string]any{"errors": counts["errors"], "warnings": counts["warnings"], "info": counts["info"], "hints": counts["hints"], "files_checked": checked}, "truncated": truncated}, nil
}

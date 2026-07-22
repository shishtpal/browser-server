package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const getDiagnosticsSchema = `{"type":"object","properties":{"path":{"type":"string"},"language":{"type":"string","enum":["go","typescript","javascript","python","rust","java","auto"]},"severity":{"type":"string","enum":["error","warning","info","hint","all"]},"include_related":{"type":"boolean"},"max_results":{"type":"integer","minimum":1,"maximum":200}},"required":["path"],"additionalProperties":false}`

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
	if remaining > 0 {
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

const maxCommandOutput = 256 << 10

func registerGetDiagnostics(r *Registry) {
	r.add(Tool{
		Name:        "get_diagnostics",
		Category:    "Code Intelligence",
		Description: "Get Go build and vet diagnostics",
		Schema:      json.RawMessage(getDiagnosticsSchema),
		Execute:     getDiagnostics,
	})
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

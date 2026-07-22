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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const analyzeCodeSchema = `{"type":"object","properties":{"path":{"type":"string"},"include_patterns":{"type":"array","items":{"type":"string"}},"exclude_patterns":{"type":"array","items":{"type":"string"}},"analysis_type":{"type":"string","enum":["symbols","imports","exports","functions","types","all"]},"include_private":{"type":"boolean"},"max_depth":{"type":"integer","minimum":1,"maximum":10},"max_results":{"type":"integer","minimum":1,"maximum":500}},"required":["path"],"additionalProperties":false}`

func registerAnalyzeCode(r *Registry) {
	r.add(Tool{
		Name:        "analyze_code",
		Category:    "Code Intelligence",
		Description: "Analyze Go code structure, imports, functions, types, and symbols",
		Schema:      json.RawMessage(analyzeCodeSchema),
		Execute:     analyzeCode,
	})
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
		fileInfo, e := os.Stat(path)
		if e != nil {
			return nil, e
		}
		if fileInfo.Size() > maxSourceSize {
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

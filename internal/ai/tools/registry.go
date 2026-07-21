package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
	"browser-server/internal/db"
)

const maxOutput = 32 * 1024

type Tool struct {
	Name        string
	Description string
	Category    string
	Schema      json.RawMessage
	Execute     func(context.Context, json.RawMessage) (any, error)
}

type Registry struct {
	tools map[string]Tool
	shell ShellInfo
}

type Options struct{ Memory config.MemoryConfig }

func New(options ...Options) *Registry {
	shell := DetectShell()
	r := &Registry{tools: map[string]Tool{}, shell: shell}
	var memory config.MemoryConfig
	if len(options) > 0 {
		memory = options[0].Memory
	}
	registerMemoryTools(r, newMemoryStore(memory))
	r.add(Tool{Name: "get_current_time", Category: "General", Description: "Get the current server time", Schema: json.RawMessage(`{"type":"object","properties":{"timezone":{"type":"string"}},"additionalProperties":false}`), Execute: currentTime})
	r.add(Tool{Name: "search_bookmarks", Category: "General", Description: "Search the local bookmark database", Schema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"integer","minimum":1},"query":{"type":"string","maxLength":200},"limit":{"type":"integer","minimum":1,"maximum":20}},"required":["user_id","query"],"additionalProperties":false}`), Execute: searchBookmarks})
	r.add(Tool{
		Name:        "execute_command",
		Category:    "Process Management",
		Description: fmt.Sprintf("Execute a shell command on the server. The server is running on %s with %s. Generate commands using %s syntax. Use this to run system commands, check file contents, list directories, manage processes, etc. Commands time out after 30 seconds max.", shell.Platform, shell.Name, shell.Name),
		Schema:      json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"The shell command to execute. Use ` + shell.Name + ` syntax.","maxLength":4096},"working_dir":{"type":"string","description":"Optional working directory for the command. Defaults to the server binary directory."},"timeout_seconds":{"type":"integer","description":"Timeout in seconds (1-30). Defaults to 10.","minimum":1,"maximum":30}},"required":["command"],"additionalProperties":false}`),
		Execute:     executeCommand(shell),
	})
	r.add(Tool{Name: "read_file", Category: "File Operations", Description: "Read a UTF-8 text file from the server filesystem (maximum 32 KiB)", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path to the file on the server"}},"required":["path"],"additionalProperties":false}`), Execute: readFile})
	r.add(Tool{Name: "write_file", Category: "File Operations", Description: "Create or overwrite a UTF-8 text file on the server filesystem, creating parent directories as needed", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Destination path on the server"},"content":{"type":"string","description":"Complete file content"}},"required":["path","content"],"additionalProperties":false}`), Execute: writeFile})
	r.add(Tool{Name: "list_directory", Category: "File Operations", Description: "List the immediate contents of a directory on the server filesystem", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path on the server; defaults to the server working directory"}},"additionalProperties":false}`), Execute: listDirectory})
	r.add(Tool{Name: "delete_file", Category: "File Operations", Description: "Delete a file from the server filesystem", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path to the file on the server"}},"required":["path"],"additionalProperties":false}`), Execute: deleteFile})
	r.add(Tool{Name: "move_file", Category: "File Operations", Description: "Move or rename a file on the server filesystem without overwriting an existing destination, creating parent directories as needed", Schema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"string","description":"Existing source file path"},"destination":{"type":"string","description":"New destination file path"}},"required":["source","destination"],"additionalProperties":false}`), Execute: moveFile})
	r.add(Tool{Name: "copy_file", Category: "File Operations", Description: "Copy a file on the server filesystem without overwriting an existing destination, creating parent directories as needed", Schema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"string","description":"Existing source file path"},"destination":{"type":"string","description":"New destination file path"}},"required":["source","destination"],"additionalProperties":false}`), Execute: copyFile})
	r.add(Tool{Name: "directory_tree", Category: "File Operations", Description: "Generate a tree-style directory listing showing the hierarchical structure of files and folders. Ignores .git and node_modules by default.", Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path to generate tree for; defaults to the server working directory"},"max_depth":{"type":"integer","description":"Maximum depth to recurse (1-20, default 5)","minimum":1,"maximum":20},"ignore_patterns":{"type":"array","items":{"type":"string"},"description":"Additional file/directory patterns to ignore (supports glob patterns like *.log). .git and node_modules are always ignored."}},"additionalProperties":false}`), Execute: directoryTree})

	// Code intelligence tools
	r.add(Tool{Name: "search_code", Category: "Code Intelligence", Description: "Search source files using regex, literal, or fixed-string matching", Schema: json.RawMessage(searchCodeSchema), Execute: searchCode})
	r.add(Tool{Name: "analyze_code", Category: "Code Intelligence", Description: "Analyze Go code structure, imports, functions, types, and symbols", Schema: json.RawMessage(analyzeCodeSchema), Execute: analyzeCode})
	r.add(Tool{Name: "get_diagnostics", Category: "Code Intelligence", Description: "Get Go build and vet diagnostics", Schema: json.RawMessage(getDiagnosticsSchema), Execute: getDiagnostics})

	// Git tools
	r.add(Tool{Name: "git_status", Category: "Git Operations", Description: "Check the git repository status: current branch, staged/unstaged changes, untracked files, ahead/behind remote", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string","description":"Repository path. Defaults to the server binary directory."}},"additionalProperties":false}`), Execute: gitStatus})
	r.add(Tool{Name: "git_diff", Category: "Git Operations", Description: "View git diff output (working tree, staged, or between commits)", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"cached":{"type":"boolean","description":"Show staged changes (--cached)"},"commit1":{"type":"string","description":"Base ref"},"commit2":{"type":"string","description":"Target ref"},"path":{"type":"string","description":"Limit diff to a specific path"}},"additionalProperties":false}`), Execute: gitDiff})
	r.add(Tool{Name: "git_log", Category: "Git Operations", Description: "View git commit history with optional filtering by branch, path, date range, or author", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":50,"description":"Max commits (default 20)"},"branch":{"type":"string"},"path":{"type":"string"},"since":{"type":"string","description":"ISO date"},"until":{"type":"string","description":"ISO date"},"author":{"type":"string"}},"additionalProperties":false}`), Execute: gitLog})
	r.add(Tool{Name: "git_branch", Category: "Git Operations", Description: "Manage git branches: list, create, delete, or rename", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"operation":{"type":"string","enum":["list","create","delete","rename"],"description":"Branch operation (default: list)"},"name":{"type":"string","description":"Branch name (required for create/delete/rename)"},"new_name":{"type":"string","description":"New name (required for rename)"},"start_point":{"type":"string","description":"Start point for create"},"force":{"type":"boolean","description":"Force delete (-D)"},"all":{"type":"boolean","description":"Include remote branches in list"}},"required":["operation"],"additionalProperties":false}`), Execute: gitBranch})
	r.add(Tool{Name: "git_checkout", Category: "Git Operations", Description: "Switch to a branch or create and switch to a new branch", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"branch":{"type":"string","description":"Branch to switch to or create"},"create":{"type":"boolean","description":"Create new branch (-b)"},"force":{"type":"boolean","description":"Force checkout"}},"required":["branch"],"additionalProperties":false}`), Execute: gitCheckout})
	r.add(Tool{Name: "git_commit", Category: "Git Operations", Description: "Create a git commit, optionally staging files first", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"message":{"type":"string","description":"Commit message (required unless amend)"},"add":{"type":"array","items":{"type":"string"},"description":"Files to stage before committing"},"all":{"type":"boolean","description":"Stage all tracked changes"},"amend":{"type":"boolean","description":"Amend the previous commit"},"allow_empty":{"type":"boolean","description":"Allow empty commit"}},"additionalProperties":false}`), Execute: gitCommit})
	r.add(Tool{Name: "git_push", Category: "Git Operations", Description: "Push commits to a remote repository", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"remote":{"type":"string","description":"Remote name (default: origin)"},"branch":{"type":"string","description":"Branch to push"},"set_upstream":{"type":"boolean","description":"Set upstream tracking (-u)"},"force":{"type":"boolean","description":"Force push (uses --force-with-lease)"},"tags":{"type":"boolean","description":"Push tags"}},"additionalProperties":false}`), Execute: gitPush})
	r.add(Tool{Name: "git_pull", Category: "Git Operations", Description: "Pull changes from a remote repository", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"remote":{"type":"string","description":"Remote name (default: origin)"},"branch":{"type":"string","description":"Branch to pull"},"rebase":{"type":"boolean","description":"Rebase instead of merge"},"ff_only":{"type":"boolean","description":"Fast-forward only"}},"additionalProperties":false}`), Execute: gitPull})
	r.add(Tool{Name: "git_merge", Category: "Git Operations", Description: "Merge a branch into the current branch", Schema: json.RawMessage(`{"type":"object","properties":{"working_dir":{"type":"string"},"branch":{"type":"string","description":"Branch to merge"},"no_ff":{"type":"boolean","description":"Create merge commit even if fast-forward possible"},"squash":{"type":"boolean","description":"Squash commits"},"no_commit":{"type":"boolean","description":"Merge without auto-commit"},"message":{"type":"string","description":"Custom merge commit message"}},"required":["branch"],"additionalProperties":false}`), Execute: gitMerge})

	return r
}
func (r *Registry) add(t Tool) { r.tools[t.Name] = t }

// Categories returns a map of tool name → category for all allowed tools.
func (r *Registry) Categories(allowed []string) map[string]string {
	out := make(map[string]string, len(allowed))
	for _, n := range allowed {
		if t, ok := r.tools[n]; ok {
			out[n] = t.Category
		}
	}
	return out
}

func (r *Registry) Specs(allowed []string) []provider.ToolSpec {
	var out []provider.ToolSpec
	for _, n := range allowed {
		if t, ok := r.tools[n]; ok {
			out = append(out, provider.ToolSpec{Name: t.Name, Description: t.Description, Parameters: t.Schema})
		}
	}
	return out
}
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) ([]byte, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool")
	}
	v, err := t.Execute(ctx, args)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if len(b) > maxOutput {
		return nil, fmt.Errorf("tool output exceeds limit")
	}
	return b, err
}

func currentTime(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Timezone string `json:"timezone"`
	}
	if err := strict(raw, &a, map[string]bool{"timezone": true}); err != nil {
		return nil, err
	}
	loc := time.UTC
	if a.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(a.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone")
		}
	}
	return map[string]string{"time": time.Now().In(loc).Format(time.RFC3339), "timezone": loc.String()}, nil
}
func searchBookmarks(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "limit": true}); err != nil {
		return nil, err
	}
	a.Query = strings.TrimSpace(a.Query)
	if a.UserID < 1 || a.Query == "" || len(a.Query) > 200 {
		return nil, fmt.Errorf("invalid bookmark search arguments")
	}
	if a.Limit == 0 {
		a.Limit = 10
	}
	if a.Limit < 1 || a.Limit > 20 {
		return nil, fmt.Errorf("limit must be 1 to 20")
	}
	rows, err := db.BookmarkDB.QueryContext(ctx, `SELECT id,title,url FROM bookmarks WHERE user_id=? AND (title LIKE ? OR url LIKE ? OR description LIKE ?) ORDER BY updated_at DESC LIMIT ?`, a.UserID, "%"+a.Query+"%", "%"+a.Query+"%", "%"+a.Query+"%", a.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int
		var title, url string
		if err := rows.Scan(&id, &title, &url); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "url": url})
	}
	return out, rows.Err()
}
func strict(raw json.RawMessage, dst any, allowed map[string]bool) error {
	var fields map[string]json.RawMessage
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	if fields == nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("unknown argument %q", k)
		}
	}
	return json.Unmarshal(raw, dst)
}

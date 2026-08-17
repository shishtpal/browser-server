# How to Add an AI Tool

AI tools are server-side functions the LLM can call during chat conversations. They live in `internal/ai/tools/` and are registered in the tool registry. Adding a new tool involves **3 files**:

### 1. Implement the tool function (`internal/ai/tools/<domain>.go`)

Group related tools in a single file (e.g. `git.go` for all git tools, `filesystem.go` for file ops). Each tool function has the same signature:

```go
func myTool(ctx context.Context, raw json.RawMessage) (any, error) {
    var a struct {
        Param1 string `json:"param1"`
        Param2 int    `json:"param2"`
    }
    if err := strict(raw, &a, map[string]bool{"param1": true, "param2": true}); err != nil {
        return nil, err
    }
    // Validate inputs
    if a.Param1 == "" {
        return nil, fmt.Errorf("param1 is required")
    }
    // Do work...
    return map[string]any{"result": "value"}, nil
}
```

Conventions:
- Use `strict(raw, &a, allowedKeys)` to validate and parse JSON arguments (rejects unknown fields)
- Return `(any, error)` — the `any` value is JSON-marshaled and sent back to the model
- Use `context.Context` for timeouts and cancellation
- Keep output under `maxOutput` (32 KiB) to avoid blowing up model context
- For shell/exec tools, use `exec.Command` with discrete args (never string interpolation) to prevent injection
- Validate that user-supplied ref names, paths, or identifiers don't start with `-` to prevent flag confusion

### 2. Register the tool (`internal/ai/tools/registry.go`)

Add an `r.add(Tool{...})` call inside the `New()` function:

```go
r.add(Tool{
    Name:        "my_tool",
    Description: "Short description of what the tool does",
    Schema:      json.RawMessage(`{"type":"object","properties":{"param1":{"type":"string","description":"..."},"param2":{"type":"integer"}},"required":["param1"],"additionalProperties":false}`),
    Execute:     myTool,
})
```

- `Name` — snake_case identifier used by the model to invoke the tool
- `Description` — helps the model decide when to use the tool; be concise but specific
- `Schema` — JSON Schema for the tool's parameters (the model sees this)
- `Execute` — the function from step 1

### 3. Whitelist in config validation AND `bs-ai-config.json`

**`internal/ai/config/config.go`** — add the tool name to the `known` map in the validation function:

```go
known := map[string]bool{
    // ... existing tools ...
    "my_tool": true,
}
```

**`bs-ai-config.json`** — add the tool name to `tools.allowed[]`:

```json
"tools": {
    "allowed": ["get_current_time", "search_bookmarks", "...", "my_tool"]
}
```

For tools registered with a JSON schema file beside the code (e.g.
`schemas/<tool>.json`), add that schema as well. Finally, update the `tools`
description in `internal/ai/tools/` and any relevant `AGENTS.md` / `README.md`
sections.

### Checklist

- [ ] Tool function in `internal/ai/tools/<domain>.go` (uses `strict()`, returns `(any, error)`)
- [ ] `r.add(Tool{...})` in `internal/ai/tools/registry.go` `New()` function
- [ ] JSON schema beside the code (e.g. `schemas/<tool>.json`) where applicable
- [ ] Tool name added to `known` map in `internal/ai/config/config.go`
- [ ] Tool name added to `tools.allowed[]` in `bs-ai-config.json`
- [ ] `go build ./cmd/server` passes
- [ ] `go vet ./...` passes
- [ ] Server starts without "unknown tool" errors

## Existing tools

| Tool | File | Description |
|------|------|-------------|
| `get_current_time` | `get_current_time.go` | Get server time in a timezone |
| `search_todos` | `search_todos.go` | Search todo database (filter by status, priority, text) |
| `search_calendar` | `search_calendar.go` | Search calendar events (todos with scheduled dates, date range filtering) |
| `manage_calendar` | `manage_calendar.go` | Manage calendar events: add, edit, remove, get (todos with start_date/end_date/rrule) |
| `search_questions` | `search_questions.go` | Search the question bank (filter by type, difficulty, tags (any-of array), subject/topic/sub_topic, text); `random: true` draws a random sample of `page_size` matches instead of ranking |
| `manage_question` | `manage_question.go` | Add, edit, remove, get, or list questions in the question bank (single_choice, multiple_choice, input, chronology); accepts a `tags` array on create/edit; `list_tags` returns the user's distinct tags/subjects/topics/sub_topics vocabulary from `quiz.TagVocabulary` |
| `search_prompts` | `search_prompts.go` | Search the prompt database (filter by user, text query) |
| `manage_prompt` | `manage_prompt.go` | Add, edit, or remove a prompt |
| `search_bookmarks` | `search_bookmarks.go` | Search bookmark database |
| `search_history` | `search_history.go` | Search browsing history |
| `execute_command` | `execute_command.go` | Run a shell command (30s timeout) |
| `web_search` | `web.go` | Search the web (requires web_search config) |
| `web_fetch` | `web.go` | Fetch content from a URL |
| `read_file` | `read_file.go` | Read a UTF-8 file (32 KiB max) |
| `write_file` | `write_file.go` | Create/overwrite a file |
| `edit_file` | `edit_file.go` | Find-and-replace edit in a file |
| `multi_edit` | `multi_edit.go` | Atomic multi-file find-and-replace edits |
| `list_directory` | `list_directory.go` | List directory contents |
| `delete_file` | `delete_file.go` | Delete a file |
| `move_file` | `move_file.go` | Move/rename a file |
| `copy_file` | `copy_file.go` | Copy a file |
| `directory_tree` | `directory_tree.go` | Recursive directory tree listing |
| `search_code` | `search_code.go` | Regex search across files |
| `analyze_code` | `analyze_code.go` | AST-based code analysis |
| `get_diagnostics` | `get_diagnostics.go` | Get compile/lint diagnostics for a file |
| `git_status` | `git_status.go` | Repository status (branch, staged, untracked) |
| `git_diff` | `git_diff.go` | View diffs (working tree, staged, between refs) |
| `git_log` | `git_log.go` | Commit history with filtering |
| `git_branch` | `git_branch.go` | List/create/delete/rename branches |
| `git_checkout` | `git_checkout.go` | Switch or create branches |
| `git_commit` | `git_commit.go` | Stage files and commit |
| `git_push` | `git_push.go` | Push to remote (uses --force-with-lease) |
| `git_pull` | `git_pull.go` | Pull from remote |
| `git_merge` | `git_merge.go` | Merge a branch |
| `recall_memory` | `internal/ai/memory` | Read/search/traverse the memory graph (optionally synthesize) |
| `write_memory` | `internal/ai/memory` | Batched, atomic memory mutations (upsert/append/link/move/archive/delete) |
| `list_skills` | `skills.go` | List available AI skills |
| `activate_skill` | `skills.go` | Activate an AI skill |
| `deactivate_skill` | `skills.go` | Deactivate an AI skill |
| `get_active_skills` | `skills.go` | Get currently active skills |
| `generate_image` | `internal/ai/bootstrap` | Generate or edit an image from a prompt |
| `text_to_speech` | `internal/ai/bootstrap` | Convert text to speech and save an MP3 under `.data/ai-voices/` |
| `speech_to_text` | `internal/ai/bootstrap` | Transcribe an audio file to text using an OpenRouter STT model |

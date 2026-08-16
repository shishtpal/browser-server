# Domain Packages (shared business logic)

Domains that are reachable from **both** the REST API and the AI tools keep their
logic in a dedicated package under `internal/<domain>/` rather than duplicating it
in `internal/handlers/` and `internal/ai/tools/`:

| Package | Used by |
|---------|---------|
| `internal/todo` | `handlers/todos.go`, `todo_subtasks.go`, `todo_reorder.go`, and the `add_todo_record`, `update_todo_record`, `manage_calendar`, `search_todos`, `search_calendar` tools |
| `internal/prompt` | `handlers/prompts.go` and the `manage_prompt`, `search_prompts` tools |
| `internal/bookmark` | `handlers/bookmarks.go`, `bookmark_import.go`, and the `search_bookmarks` tool |
| `internal/history` | `handlers/history.go`, `history_import.go`, `search.go`, and the `search_history` tool |
| `internal/quiz` | `handlers/questions.go` and the `search_questions`, `manage_question` tools |

Each package is layered the same way:

- **`<domain>.go`** — pure validation and constants (field limits, valid enum
  values, date parsing). No database access, so it is trivial to unit test.
- **`store.go`** — the single source of truth for the SQL: the `Columns`
  constant, the row `Scan`, `Create`, an `UpdateBuilder` for partial updates,
  and ownership checks. Every query that scans a row **must** select `Columns`.
- **`view.go`** — the two renderings of a record: `Response(...)` for the REST
  API (typed `models.*Response`) and `Map(...)` for AI tools (a flat
  `map[string]any` where blank optional strings become `null`).

**Rules when working in these domains:**

- Never re-declare a column list, row scanner, or validation table in a handler
  or a tool — import the domain package instead. Adding a column then means
  editing one `Columns` constant.
- Keep HTTP concerns (status codes, `helpers.WriteError`) in handlers and
  tool-argument concerns (`strict(...)`) in tools. Domain packages return plain
  values and sentinel errors (`ErrNotFound`, `ErrFolderNotOwned`, …) that the
  caller maps to its own response format.
- The REST API and the AI tools intentionally accept **different status sets**
  for todos: use `IsValidCoreStatus` for the API and `IsValidStatus` for tools,
  which additionally tolerates the legacy `done`/`cancelled` aliases.
- Prompt rows store `created_at` with sub-second precision from Go rather than
  SQL's whole-second `CURRENT_TIMESTAMP`, because prompt listings order by
  `created_at` and would otherwise be non-deterministic within the same second.

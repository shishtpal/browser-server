# Quiz Configuration

The Quiz / Question Bank feature is its own self-contained feature gated by a sibling file next to the server binary: `bs-quiz-config.json`. When the file is missing or `"enabled": false` the feature is a no-op — no `quiz.db` is created, no routes are registered, and the `search_questions` / `manage_question` AI tools report `"quiz feature disabled"`. See `internal/quiz/config/config.go` for the loader and `internal/handlers/questions.go` for the per-handler gate.

Config path resolution (first match wins):

1. `BS_QUIZ_CONFIG_PATH` environment variable
2. `<executable dir>/bs-quiz-config.json` (same `ExecutableDir()` anchor used by the AI config)

Key sections:

```jsonc
{
  "enabled": true,
  "db_path": ".data/quiz.db",
  "image_dir": ".data/quiz-images",
  "limits": {
    "max_question_length": 2000,
    "max_explanation_length": 20000,
    "max_option_length": 500,
    "max_chronology_items": 20,
    "max_options_per_question": 10,
    "max_image_bytes": 5242880,
    "max_paper_size": 200,
    "max_papers_per_user": 100
  },
  "allowed_question_types": ["single_choice", "multiple_choice", "input", "chronology"],
  "allowed_difficulties": ["easy", "medium", "hard"],
  "tag_categories": ["subject", "topic", "sub_topic"],
  "ai_tools": {
    "enabled": true,
    "search_questions": true,
    "manage_question": true
  },
  "paper_generation": {
    "default_sample_strategy": "random",
    "allow_duplicate_questions_within_paper": false
  },
  "retention_days": 365,
  "cors_enabled": false
}
```

Question types supported by `allowed_question_types`:

- `single_choice` — exactly one option marked `correct: true` (2..`max_options_per_question` options).
- `multiple_choice` — one or more options marked `correct`.
- `input` — free-text answer stored as `{"text": "..."}` in `answer_json`.
- `chronology` — arrange-items-in-correct-order; items stored with `correct_order` values forming a 1..N permutation.

Tags `tags`, `subject`, `topic`, `sub_topic` drive both filtering (`/api/quiz/questions?tag=...&tag=...&subject=...`) and sectioned paper generation (`/api/quiz/papers` body accepts `[{tags, subject, topic, sub_topic, type, difficulty, count}]`). The `tags` field is a JSON array on each question (e.g. `["SSC","RRB"]`) — a single question can carry any number of tags, so a "polity" question can be marked valid for both SSC and UPSC at once. Filtering uses `EXISTS (SELECT 1 FROM json_each(tags) WHERE value IN (...))` so multiple `?tag=` query params match any-of.

To enable the AI tools, two steps are required: the quiz feature must be enabled here (the file must exist with `"enabled": true`) **and** `search_questions` / `manage_question` must appear in `bs-ai-config.json` → `tools.allowed[]`.

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/gorilla/mux"

	browserconfig "browser-server/internal/ai/browser/config"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/images"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/tts"
	"browser-server/internal/ai/videos"
	"browser-server/internal/ai/voice"
	quizconfig "browser-server/internal/quiz/config"
)

// maxSchemaDepth bounds the recursion used while post-processing generated
// schemas so a pathological schema cannot overflow the stack.
const maxSchemaDepth = 64

// configSchema returns a JSON Schema (draft-07 shape) describing the editable
// shape of the given whitelisted config file, built from the Go structs that
// actually parse it via jsonschema.For. Files without a curated struct mapping
// return the JSON value null, which the frontend treats as "no form editor,
// use code mode".
//
// The generated schema is normalized before it is returned: the struct-based
// generator marks every non-omitempty field as required, but these config
// files are intentionally partial (the parsers default whatever is missing).
// We therefore strip every "required" array and apply small per-field overlays
// (enums, bounds, descriptions) that the reflection-based generator cannot
// derive. Unknown/extra fields are still validated server-side on save, so the
// form never silently drops a server-side rule.
func configSchema(name string) (json.RawMessage, error) {
	schema, buildErr := schemaForFile(name)
	if schema == nil && buildErr == nil {
		// No struct-backed form for this file (or it is intentionally
		// code-only). Return JSON null so callers can distinguish "no form"
		// from a build failure.
		return json.RawMessage(`null`), nil
	}
	if buildErr != nil {
		return nil, buildErr
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	root := make(map[string]any)
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	stripRequired(root, 0)
	if err := applySchemaOverlays(name, root, 0); err != nil {
		return nil, err
	}
	return json.Marshal(root)
}

// schemaForFile maps each whitelisted config file to the Go type that parses
// it and derives its schema. Returning (nil, nil) means "no form available".
func schemaForFile(name string) (*jsonschema.Schema, error) {
	switch name {
	case "bs-quiz-config.json":
		return jsonschema.For[quizconfig.Config](nil)
	case "bs-browser-config.json":
		return jsonschema.For[browserconfig.Config](nil)
	case "bs-ai-config.json":
		return jsonschema.For[aiconfig.Config](nil)
	case "bs-ai-voice.json":
		return jsonschema.For[voice.Config](nil)
	case "bs-ai-image-models.json":
		return jsonschema.For[images.Config](nil)
	case "bs-ai-video-models.json":
		return jsonschema.For[videos.Config](nil)
	case "bs-ai-mcp.json":
		return jsonschema.For[aimcp.Config](nil)
	case "bs-ai-models.json":
		return jsonschema.For[aiconfig.ModelsFile](nil)
	case "bs-ai-tts.json":
		return jsonschema.For[tts.Config](nil)
	default:
		return nil, nil
	}
}

// schemaOverlays holds per-file, per JSON-path schema enrichments that the
// reflection generator cannot infer from the Go struct tags alone (enums,
// numeric bounds, and human descriptions). Keys use dot-separated property
// paths, e.g. "timeouts.default_command_timeout_ms".
var schemaOverlays = map[string]map[string]map[string]any{
	"bs-quiz-config.json": {
		"scheduler": {
			"description": "Spaced-repetition engine: sm2 (default) or fsrs.",
			"enum":        []any{"sm2", "fsrs"},
		},
		"retention_days": {"description": "How long quiz records are kept before cleanup."},
	},
	"bs-browser-config.json": {
		"eval": {"description": "Controls how browser_eval / browser_execute JavaScript runs."},
		"eval.default_mode": {
			"description": "Eval mode used when a call omits mode: inject (default, no infobar) or cdp (CDP via debugger, bypasses page CSP).",
			"enum":        []any{"inject", "cdp"},
		},
		"timeouts":                            {"description": "Per-command execution and selector-polling bounds for the browser automation tools."},
		"timeouts.default_command_timeout_ms": {"description": "Default per-command timeout when timeout_ms is omitted (ms). 0 uses the built-in default (60000).", "minimum": 0},
		"timeouts.max_command_timeout_ms":     {"description": "Hard ceiling on any command's timeout_ms. 0 uses the built-in default (600000). Must be >= the default.", "minimum": 0},
		"timeouts.selector_timeout_ms":        {"description": "Default selector-polling budget for browser_wait (ms). 0 uses the built-in default (10000).", "minimum": 0},
	},
	"bs-ai-models.json": {
		"providers": {"description": "Provider/model catalog. Provider names are referenced by bs-ai-config.json (default_provider, synthesizer, tasks, ocr)."},
		"providers.*.type": {
			"description": "Provider client type: openai_compatible (OpenRouter/OpenAI-compatible) or gemini_interactions (Google's stateless Interactions API).",
			"enum":        []any{"openai_compatible", "gemini_interactions"},
		},
	},
	"bs-ai-tts.json": {
		"default_provider": {"description": "Name of the speech provider used when a call omits provider. Must match a key under providers."},
		"providers.*.type": {
			"description": "TTS provider type: openrouter_speech (OpenAI-compatible /audio/speech) or fish_audio.",
			"enum":        []any{"openrouter_speech", "fish_audio"},
		},
	},
}

// propertyAt walks dot-separated property paths through a schema's
// "properties" nested objects. A "*" segment steps into the
// "additionalProperties" value schema (for map-shaped nodes), so an overlay
// can reach fields of a map value, e.g. "providers.*.type".
func propertyAt(root map[string]any, path string) (map[string]any, bool) {
	cur := root
	for _, segment := range strings.Split(path, ".") {
		if segment == "*" {
			additional, ok := cur["additionalProperties"].(map[string]any)
			if !ok {
				return nil, false
			}
			cur = additional
			continue
		}
		props, ok := cur["properties"].(map[string]any)
		if !ok {
			return nil, false
		}
		child, ok := props[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = child
	}
	return cur, true
}

// stripRequired recursively removes every "required" array from a generated
// schema. The struct-based generator marks partial config fields as required,
// but these files are intentionally partial (parsers default missing fields),
// so a strict required set would block otherwise-valid saves.
func stripRequired(node any, depth int) {
	if depth > maxSchemaDepth {
		return
	}
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	delete(obj, "required")
	if props, ok := obj["properties"].(map[string]any); ok {
		for _, child := range props {
			stripRequired(child, depth+1)
		}
	}
	if items, ok := obj["items"].(map[string]any); ok {
		stripRequired(items, depth+1)
	}
	if additional, ok := obj["additionalProperties"].(map[string]any); ok {
		stripRequired(additional, depth+1)
	}
	if anyOf, ok := obj["anyOf"].([]any); ok {
		for _, child := range anyOf {
			stripRequired(child, depth+1)
		}
	}
	if oneOf, ok := obj["oneOf"].([]any); ok {
		for _, child := range oneOf {
			stripRequired(child, depth+1)
		}
	}
}

func applySchemaOverlays(name string, root map[string]any, depth int) error {
	if depth > maxSchemaDepth {
		return errors.New("schema overlay nesting too deep")
	}
	overlays, ok := schemaOverlays[name]
	if !ok {
		return nil
	}
	for path, props := range overlays {
		node, ok := propertyAt(root, path)
		if !ok {
			return errors.New("configured schema overlay path not found: " + path)
		}
		for key, value := range props {
			node[key] = value
		}
	}
	return nil
}

// configSchemaFile is the GET /config/schema/{name} admin handler.
func (m *Module) configSchemaFile(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	meta, status, ok := configMeta(name)
	if !ok {
		writeError(w, status, "invalid_config_name", "Config file is not in the administrator whitelist.")
		return
	}
	schema, err := configSchema(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "schema_build_failed",
			errors.New("build config schema: "+err.Error()).Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":   meta.Name,
		"schema": json.RawMessage(schema),
	})
}

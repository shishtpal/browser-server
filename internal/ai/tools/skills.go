package tools

import (
	"context"
	"encoding/json"

	"browser-server/internal/ai/skills"
)

// RegisterSkillTools adds skill management tools to the registry.
// activate_skill and deactivate_skill have nil Execute because they are
// intercepted by the chat service's tool-call loop (they modify session state).
func RegisterSkillTools(r *Registry, reg *skills.Registry) {
	r.add(Tool{
		Name:        "list_skills",
		Category:    "Skills",
		Description: "List all available AI skills with their descriptions and tool requirements",
		Schema:      json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		Execute:     listSkillsTool(reg),
	})
	r.add(Tool{
		Name:        "activate_skill",
		Category:    "Skills",
		Description: "Activate a skill to gain its focused instructions and tools. Multiple skills can be active simultaneously (max 5).",
		Schema:      json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Skill name (kebab-case identifier)"}},"required":["name"],"additionalProperties":false}`),
		Execute:     nil, // intercepted by chat service
	})
	r.add(Tool{
		Name:        "deactivate_skill",
		Category:    "Skills",
		Description: "Deactivate a currently active skill. Removes its instructions and tools from the session.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Skill name to deactivate"}},"required":["name"],"additionalProperties":false}`),
		Execute:     nil, // intercepted by chat service
	})
	r.add(Tool{
		Name:        "get_active_skills",
		Category:    "Skills",
		Description: "Get the list of currently active skills in this conversation",
		Schema:      json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		Execute:     nil, // intercepted by chat service
	})
}

// SkillToolNames returns the names of all skill meta-tools that should always
// be available regardless of active skill tool restrictions.
func SkillToolNames() []string {
	return []string{"list_skills", "activate_skill", "deactivate_skill", "get_active_skills"}
}

func listSkillsTool(reg *skills.Registry) func(context.Context, json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		list := reg.List()
		type skillSummary struct {
			Name        string   `json:"name"`
			Label       string   `json:"label"`
			Description string   `json:"description"`
			Category    string   `json:"category"`
			Tags        []string `json:"tags"`
			Tools       []string `json:"tools"`
		}
		out := make([]skillSummary, 0, len(list))
		for _, s := range list {
			out = append(out, skillSummary{
				Name:        s.Name,
				Label:       s.Label,
				Description: s.Description,
				Category:    s.Category,
				Tags:        s.Tags,
				Tools:       s.Tools,
			})
		}
		return out, nil
	}
}

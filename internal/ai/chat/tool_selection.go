package chat

import (
	"browser-server/internal/ai/provider"
	"browser-server/internal/ai/skills"
	"browser-server/internal/ai/tools"
)

// resolveActiveTools computes the effective tool list for a request.
// If skills specify tool whitelists, only their union (intersected with server allowed) is used.
// If the client sends an explicit ActiveTools list, it's intersected further.
// Skill meta-tools are always included.
func (s *Service) resolveActiveTools(clientActive []string, activeSkills []*skills.Skill) []string {
	allowed := s.cfg.Tools.Allowed

	// If any active skill specifies a tools list, union them to form the base
	hasToolRestriction := false
	skillToolSet := make(map[string]bool)
	for _, skill := range activeSkills {
		if len(skill.Tools) > 0 {
			hasToolRestriction = true
			for _, name := range skill.Tools {
				skillToolSet[name] = true
			}
		}
	}

	if hasToolRestriction {
		// Always include skill meta-tools
		for _, name := range tools.SkillToolNames() {
			skillToolSet[name] = true
		}
		skillToolSet[tools.SearchToolName] = true
		allowedSet := make(map[string]bool, len(allowed))
		for _, name := range allowed {
			allowedSet[name] = true
		}
		var skillAllowed []string
		for name := range skillToolSet {
			if allowedSet[name] {
				skillAllowed = append(skillAllowed, name)
			}
		}
		allowed = skillAllowed
	}

	if clientActive == nil {
		return allowed
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = true
	}
	// Always include configured skill meta-tools even if the client didn't list them.
	hasSkills := s.skills != nil && len(s.skills.List()) > 0
	if hasSkills {
		serverAllowedSet := make(map[string]bool, len(s.cfg.Tools.Allowed))
		for _, name := range s.cfg.Tools.Allowed {
			serverAllowedSet[name] = true
		}
		for _, name := range tools.SkillToolNames() {
			if serverAllowedSet[name] {
				allowedSet[name] = true
			}
		}
	}
	var result []string
	for _, name := range clientActive {
		if allowedSet[name] {
			result = append(result, name)
		}
	}
	// Ensure skill tools are always present when skills exist
	if hasSkills {
		for _, name := range tools.SkillToolNames() {
			found := false
			for _, r := range result {
				if r == name {
					found = true
					break
				}
			}
			if !found {
				result = append(result, name)
			}
		}
	}
	return result
}

func (s *Service) configureToolDefinitions(chatReq *provider.ChatRequest, activeTools []string, loaded map[string]bool, includeAll bool) map[string]bool {
	activeSet := make(map[string]bool, len(activeTools))
	for _, name := range activeTools {
		activeSet[name] = true
	}
	for name := range loaded {
		if !activeSet[name] {
			delete(loaded, name)
		}
	}

	visible := activeTools
	if !includeAll && activeSet[tools.SearchToolName] {
		visible = make([]string, 0, len(loaded)+1)
		for _, name := range activeTools {
			if name == tools.SearchToolName || loaded[name] {
				visible = append(visible, name)
			}
		}
	}
	chatReq.Tools = s.tools.Specs(visible)
	return activeSet
}

func (s *Service) activeToolNames(activeSet map[string]bool) []string {
	names := make([]string, 0, len(activeSet))
	for _, name := range s.cfg.Tools.Allowed {
		if activeSet[name] {
			names = append(names, name)
		}
	}
	return names
}

func isToolCallable(name string, activeSet, loaded map[string]bool, includeAll bool) bool {
	if !activeSet[name] {
		return false
	}
	if includeAll || !activeSet[tools.SearchToolName] {
		return true
	}
	return name == tools.SearchToolName || loaded[name]
}

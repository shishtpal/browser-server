package config

import (
	"fmt"
	"log"
)

// validateExploreProject checks the explore_project section for internal
// consistency and references into the providers/models file. The explorer agent
// drives read-only tools via function calls, so the resolved default model must
// advertise supports_tools: true.
func validateExploreProject(cfg *Config) error {
	if !cfg.ExploreProject.Enabled {
		return nil
	}
	if cfg.ExploreProject.MaxIterations < 1 || cfg.ExploreProject.MaxIterations > 50 {
		return fmt.Errorf("explore_project.max_iterations must be between 1 and 50")
	}
	if cfg.ExploreProject.TimeoutMS < 1000 || cfg.ExploreProject.TimeoutMS > 600000 {
		return fmt.Errorf("explore_project.timeout_ms must be between 1000 and 600000")
	}
	if cfg.ExploreProject.Temperature < 0 || cfg.ExploreProject.Temperature > 2 {
		return fmt.Errorf("explore_project.temperature must be between 0 and 2")
	}
	_, model, ok := cfg.FindModel(cfg.ExploreProject.DefaultProvider, cfg.ExploreProject.DefaultModel)
	if !ok {
		return fmt.Errorf("explore_project.default_provider/explore_project.default_model (%s/%s) must reference a model configured in the models file", cfg.ExploreProject.DefaultProvider, cfg.ExploreProject.DefaultModel)
	}
	if !model.SupportsTools {
		return fmt.Errorf("explore_project.default_model %q must have supports_tools: true in the models file (the explorer agent uses function calling)", cfg.ExploreProject.DefaultModel)
	}
	allowed := false
	for _, name := range cfg.Tools.Allowed {
		if name == "explore_project" {
			allowed = true
			break
		}
	}
	if !allowed {
		// tools.allowed is explicit operator policy — never auto-append, only warn.
		log.Printf("WARN: explore_project.enabled is true but %q is absent from tools.allowed; the explore_project tool will not be exposed", "explore_project")
	}
	return nil
}

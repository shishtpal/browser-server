package skills

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	skillDir     = ".skills"
	maxSkillSize = 100 * 1024 // 100 KB
	maxActive    = 5          // max simultaneously active skills
)

// Skill represents a single skill loaded from a .md file with YAML frontmatter.
type Skill struct {
	Name        string   `json:"name" yaml:"name"`
	Label       string   `json:"label" yaml:"label"`
	Description string   `json:"description,omitempty" yaml:"description"`
	Category    string   `json:"category,omitempty" yaml:"category"`
	Tags        []string `json:"tags,omitempty" yaml:"tags"`
	Tools       []string `json:"tools,omitempty" yaml:"tools"`
	Context     []string `json:"context,omitempty" yaml:"context"`
	Content     string   `json:"-" yaml:"-"` // prompt body (not exposed in list API)
	SourceFile  string   `json:"-" yaml:"-"` // file path
}

// Registry holds loaded skills and provides lookup.
type Registry struct {
	skills []Skill
	byName map[string]*Skill
}

// Load scans the .skills/ directory relative to baseDir for .md files
// and returns a Registry. If the directory does not exist or is empty,
// an empty registry is returned without error.
func Load(baseDir string) (*Registry, error) {
	dir := filepath.Join(baseDir, skillDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{byName: map[string]*Skill{}}, nil
		}
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	reg := &Registry{byName: map[string]*Skill{}}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Printf("WARN: skill %q stat error: %v", name, err)
			continue
		}
		if info.Size() > maxSkillSize {
			log.Printf("WARN: skill %q exceeds max size (%d > %d), skipping", name, info.Size(), maxSkillSize)
			continue
		}

		filePath := filepath.Join(dir, name)
		skill, err := parseSkillFile(filePath)
		if err != nil {
			log.Printf("WARN: skill %q parse error: %v", name, err)
			continue
		}

		if skill.Name == "" {
			log.Printf("WARN: skill %q has no name in frontmatter, skipping", name)
			continue
		}
		if skill.Label == "" {
			skill.Label = skill.Name
		}

		// Deduplicate by name (first wins)
		if _, exists := reg.byName[skill.Name]; exists {
			log.Printf("WARN: duplicate skill name %q in %q, skipping", skill.Name, name)
			continue
		}

		reg.skills = append(reg.skills, *skill)
		reg.byName[skill.Name] = &reg.skills[len(reg.skills)-1]
	}

	return reg, nil
}

// List returns all loaded skills (without content).
func (r *Registry) List() []Skill {
	if r == nil {
		return nil
	}
	return r.skills
}

// Get returns a skill by name. Returns nil, false if not found.
func (r *Registry) Get(name string) (*Skill, bool) {
	if r == nil {
		return nil, false
	}
	skill, ok := r.byName[name]
	return skill, ok
}

// MaxActive returns the maximum number of simultaneously active skills.
func (r *Registry) MaxActive() int {
	return maxActive
}

// parseSkillFile reads a .md file and extracts YAML frontmatter + body content.
func parseSkillFile(filePath string) (*Skill, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	skill, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill.Content = strings.TrimSpace(body)
	skill.SourceFile = filePath
	return skill, nil
}

// parseFrontmatter splits content into YAML frontmatter and body.
// Frontmatter is delimited by --- on its own line at the start.
func parseFrontmatter(content string) (*Skill, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		// No frontmatter — treat entire content as body with empty metadata
		return &Skill{}, content, nil
	}

	// Find closing ---
	rest := content[3:]
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		return &Skill{}, "", nil
	}

	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return nil, "", fmt.Errorf("unterminated YAML frontmatter")
	}

	frontmatter := rest[:endIdx]
	body := rest[endIdx+4:] // skip \n---
	// Skip optional newline after closing ---
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
	}

	return &skill, body, nil
}

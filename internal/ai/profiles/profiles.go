package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	profileDir     = ".profiles"
	maxProfileSize = 100 * 1024 // 100 KB
)

// Profile represents a single system prompt profile loaded from a .md file.
type Profile struct {
	Name    string `json:"name"`    // lowercase identifier derived from filename
	Label   string `json:"label"`   // display name (filename without extension, preserving case)
	Content string `json:"-"`       // markdown content (not exposed in JSON)
}

// Registry holds loaded profiles and provides lookup methods.
type Registry struct {
	profiles []Profile
	byName   map[string]string // name -> content
}

// Load scans the .profiles/ directory relative to baseDir for .md files
// and returns a Registry. If the directory does not exist or is empty,
// an empty registry is returned without error.
func Load(baseDir string) (*Registry, error) {
	dir := filepath.Join(baseDir, profileDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{byName: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	reg := &Registry{byName: map[string]string{}}

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
			continue
		}
		if info.Size() > maxProfileSize {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		label := strings.TrimSuffix(name, filepath.Ext(name))
		key := strings.ToLower(label)

		profile := Profile{
			Name:    key,
			Label:   label,
			Content: string(content),
		}
		reg.profiles = append(reg.profiles, profile)
		reg.byName[key] = profile.Content
	}

	return reg, nil
}

// List returns all loaded profiles (without content).
func (r *Registry) List() []Profile {
	if r == nil {
		return nil
	}
	return r.profiles
}

// Get returns the content of a profile by name. Returns empty string and
// false if the profile does not exist.
func (r *Registry) Get(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	content, ok := r.byName[strings.ToLower(name)]
	return content, ok
}

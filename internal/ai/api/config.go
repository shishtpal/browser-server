package api

import (
	"net/http"

	aiconfig "browser-server/internal/ai/config"

	"github.com/gorilla/mux"
)

type configResponse struct {
	aiconfig.SanitizedConfig
	Profiles []profileInfo `json:"profiles"`
	Skills   []skillInfo   `json:"skills"`
}

type profileInfo struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type skillInfo struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Tools       []string `json:"tools,omitempty"`
}

func (m *Module) Config(w http.ResponseWriter, r *http.Request) {
	var categories map[string]string
	if m.service != nil {
		categories = m.service.ToolCategories()
	}
	resp := configResponse{
		SanitizedConfig: m.cfg.Sanitized(categories),
		Profiles:        make([]profileInfo, 0),
		Skills:          make([]skillInfo, 0),
	}
	for _, p := range m.profiles.List() {
		resp.Profiles = append(resp.Profiles, profileInfo{Name: p.Name, Label: p.Label})
	}
	if m.skills != nil {
		for _, sk := range m.skills.List() {
			resp.Skills = append(resp.Skills, skillInfo{
				Name:        sk.Name,
				Label:       sk.Label,
				Description: sk.Description,
				Category:    sk.Category,
				Tags:        sk.Tags,
				Tools:       sk.Tools,
			})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (m *Module) ListSkills(w http.ResponseWriter, r *http.Request) {
	if m.skills == nil {
		writeJSON(w, http.StatusOK, []skillInfo{})
		return
	}
	out := make([]skillInfo, 0)
	for _, sk := range m.skills.List() {
		out = append(out, skillInfo{
			Name:        sk.Name,
			Label:       sk.Label,
			Description: sk.Description,
			Category:    sk.Category,
			Tags:        sk.Tags,
			Tools:       sk.Tools,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) GetSkill(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if m.skills == nil {
		writeError(w, http.StatusNotFound, "not_found", "Skill not found")
		return
	}
	sk, ok := m.skills.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Skill not found")
		return
	}
	// Return full skill including content for detail view
	type skillDetail struct {
		skillInfo
		Content string `json:"content"`
	}
	writeJSON(w, http.StatusOK, skillDetail{
		skillInfo: skillInfo{
			Name:        sk.Name,
			Label:       sk.Label,
			Description: sk.Description,
			Category:    sk.Category,
			Tags:        sk.Tags,
			Tools:       sk.Tools,
		},
		Content: sk.Content,
	})
}

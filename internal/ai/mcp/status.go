package mcp

// ServerStatus is deliberately sanitized: it never contains endpoints,
// commands, arguments, environment values, headers, or credentials.
type ServerStatus struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Tools      []string `json:"tools,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Error      string   `json:"error,omitempty"`
	discovered []pendingTool
}

// Statuses returns statuses in deterministic server-name order.
func (m *Manager) Statuses() []ServerStatus {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ServerStatus, 0, len(m.servers))
	for _, server := range m.servers {
		s := server.status
		s.Tools = append([]string(nil), s.Tools...)
		s.Warnings = append([]string(nil), s.Warnings...)
		s.discovered = nil
		out = append(out, s)
	}
	return out
}

package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// frontmatter mirrors Fragment for the JSON frontmatter. Body is deliberately
// excluded; it lives after the closing "---" separator as markdown.
type frontmatter struct {
	ID          string    `json:"id"`
	Kind        Kind      `json:"kind"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Tags        []string  `json:"tags,omitempty"`
	Status      Status    `json:"status,omitempty"`
	Pinned      bool      `json:"pinned,omitempty"`
	Confidence  float32   `json:"confidence,omitempty"`
	Salience    float32   `json:"salience,omitempty"`
	Links       []Link    `json:"links,omitempty"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Accessed    time.Time `json:"accessed,omitempty"`
	AccessCount int       `json:"access_count,omitempty"`
	Source      string    `json:"source"`
}

// encodeFragment renders a git-diff friendly markdown file: JSON frontmatter,
// the markdown body, and a human-navigable "## Links" section derived from
// frontmatter (frontmatter stays the source of truth on read).
func encodeFragment(f *Fragment) ([]byte, error) {
	fm := frontmatter{
		ID: f.ID, Kind: f.Kind, Title: f.Title, Summary: f.Summary, Tags: f.Tags,
		Status: f.Status, Pinned: f.Pinned, Confidence: f.Confidence, Salience: f.Salience,
		Links: f.Links, Created: f.Created, Updated: f.Updated, Accessed: f.Accessed,
		AccessCount: f.AccessCount, Source: f.Source,
	}
	h, err := json.MarshalIndent(fm, "", "  ")
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(h)
	sb.WriteString("\n---\n")
	if strings.TrimSpace(f.Body) != "" {
		sb.WriteString(strings.TrimSpace(f.Body))
		sb.WriteString("\n")
	}
	if len(fm.Links) > 0 {
		sb.WriteString("\n## Links\n")
		for _, l := range fm.Links {
			sb.WriteString(fmt.Sprintf("- %s → [[%s]]", l.Rel, l.To))
			if l.Note != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", l.Note))
			}
			sb.WriteString("\n")
		}
	}
	return []byte(sb.String()), nil
}

// decodeFragment parses a fragment file back into a *Fragment. A trailing
// rendered "## Links" section is stripped because frontmatter is the source of
// truth; this keeps the codec a true round-trip.
func decodeFragment(b []byte) (*Fragment, error) {
	raw := string(b)
	parts := strings.SplitN(raw, "\n---\n", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "---\n") {
		return nil, fmt.Errorf("invalid fragment frontmatter")
	}
	var fm frontmatter
	if err := json.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &fm); err != nil {
		return nil, err
	}
	body := parts[1]
	// Only strip the trailing rendered "## Links" section when there are
	// frontmatter links it could have been generated from; a body that ends
	// with its own ## Links heading and has no frontmatter links is kept as-is.
	if len(fm.Links) > 0 {
		if idx := strings.LastIndex(body, "\n## Links\n"); idx >= 0 {
			body = body[:idx]
		}
	}
	f := &Fragment{
		ID: fm.ID, Kind: fm.Kind, Title: fm.Title, Summary: fm.Summary, Tags: fm.Tags,
		Status: fm.Status, Pinned: fm.Pinned, Confidence: fm.Confidence, Salience: fm.Salience,
		Links: fm.Links, Created: fm.Created, Updated: fm.Updated, Accessed: fm.Accessed,
		AccessCount: fm.AccessCount, Source: fm.Source, Body: strings.TrimSpace(body),
	}
	if f.Kind == "" {
		f.Kind = KindNote
	}
	if f.Status == "" {
		f.Status = StatusActive
	}
	if f.Source == "" {
		f.Source = "ai"
	}
	return f, nil
}

// atomicWrite writes b to path atomically via a temp file + rename in the
// same directory, so a crash never leaves a truncated fragment.
func atomicWrite(path string, b []byte) error {
	return writeTempThenRename(path, b)
}

func writeTempThenRename(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".mem-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

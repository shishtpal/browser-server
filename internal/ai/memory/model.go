package memory

import (
	"regexp"
	"strings"
	"time"
)

// Kind is the closed vocabulary for fragment kinds. The model cannot invent
// new kinds; write_memory rejects any that fall outside this set.
type Kind string

const (
	KindPersona    Kind = "persona"
	KindProject    Kind = "project"
	KindComponent  Kind = "component"
	KindDecision   Kind = "decision"
	KindTask       Kind = "task"
	KindPreference Kind = "preference"
	KindFact       Kind = "fact"
	KindPerson     Kind = "person"
	KindEvent      Kind = "event"
	KindGlossary   Kind = "glossary"
	KindIndex      Kind = "index"
	KindNote       Kind = "note"
	KindStub       Kind = "stub"
)

var validKinds = map[Kind]bool{
	KindPersona: true, KindProject: true, KindComponent: true, KindDecision: true,
	KindTask: true, KindPreference: true, KindFact: true, KindPerson: true,
	KindEvent: true, KindGlossary: true, KindIndex: true, KindNote: true, KindStub: true,
}

func validateKind(k Kind) bool { return validKinds[k] }

// Rel is the closed vocabulary for edge relationships.
type Rel string

const (
	RelChildOf     Rel = "child_of"
	RelRelates     Rel = "relates"
	RelDependsOn   Rel = "depends_on"
	RelSupersedes  Rel = "supersedes"
	RelAbout       Rel = "about"
	RelContradicts Rel = "contradicts"
	RelSource      Rel = "source"
)

var validRels = map[Rel]bool{
	RelChildOf: true, RelRelates: true, RelDependsOn: true, RelSupersedes: true,
	RelAbout: true, RelContradicts: true, RelSource: true,
}

func validateRel(r Rel) bool { return validRels[r] }

// symmetricRels are mirrored automatically onto both endpoints.
var symmetricRels = map[Rel]bool{RelRelates: true, RelContradicts: true}

// inverseRels maps a directional rel to the derived rel reported on the
// reverse edge during traversal. These derived rels are not part of the closed
// vocab; they exist only on reads.
var inverseRels = map[Rel]Rel{
	RelDependsOn:  "required_by",
	RelSupersedes: "superseded_by",
}

// relDecay scales spreading activation by edge type.
var relDecay = map[Rel]float64{
	RelChildOf: 1.0, RelRelates: 0.9, RelDependsOn: 0.8, RelAbout: 0.8,
	RelSupersedes: 0.5, RelSource: 0.3, RelContradicts: 0.6,
}

// Status is the closed status vocabulary for a fragment.
type Status string

const (
	StatusActive     Status = "active"
	StatusArchived   Status = "archived"
	StatusSuperseded Status = "superseded"
)

// Fragment is the only unit of storage in the graph. The summary is always
// loaded; the body is loaded on demand from disk (detail:"full").
type Fragment struct {
	ID          string    `json:"id"`
	Kind        Kind      `json:"kind"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Body        string    `json:"-"` // markdown after frontmatter, lazy
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

// Link is one typed edge on a fragment.
type Link struct {
	Rel    Rel     `json:"rel"`
	To     string  `json:"to"`
	Note   string  `json:"note,omitempty"`
	Weight float32 `json:"weight,omitempty"`
}

// Edge is a directed relationship used in traversal output and the reverse
// index.
type Edge struct {
	From   string  `json:"from"`
	Rel    Rel     `json:"rel"`
	To     string  `json:"to"`
	Note   string  `json:"note,omitempty"`
	Weight float32 `json:"weight,omitempty"`
}

var idRe = regexp.MustCompile(`^mem_[a-z0-9](?:[a-z0-9_]{0,62}[a-z0-9])?$`)

// validID reports whether id is a legal mem_ slug (path-traversal safe: no
// slash, dot, or uppercase).
func validID(id string) bool { return idRe.MatchString(id) }

// reservedIDs are auto-created at bootstrap and are never garbage collected.
var reservedIDs = map[string]bool{
	"mem_root": true, "mem_self": true, "mem_user": true,
	"mem_projects": true, "mem_glossary": true, "mem_inbox": true,
}

// undeletableIDs can never be hard-deleted.
var undeletableIDs = map[string]bool{
	"mem_root": true, "mem_self": true, "mem_user": true, "mem_projects": true,
}

// slugify derives a stable human-readable id from a title, e.g.
// "Memory system v2" -> "mem_memory_system_v2".
func slugify(title string) string {
	var b strings.Builder
	b.WriteString("mem_")
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '/' || r == '_' || r == '.' || r == ':':
			b.WriteByte('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "mem" {
		return "mem_note"
	}
	return s
}

package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// v1 memory file frontmatter (memoryMeta). Only the fields we map are present.
type v1Meta struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Title        string    `json:"title"`
	Type         string    `json:"type"`
	TargetID     string    `json:"target_id"`
	Relationship string    `json:"relationship"`
	References   []string  `json:"references"`
	Tags         []string  `json:"tags"`
	Category     string    `json:"category"`
	Importance   string    `json:"importance"`
	Source       string    `json:"source"`
}

type v1File struct {
	Meta    v1Meta
	Content string
}

func decodeV1(b []byte) (*v1File, error) {
	raw := string(b)
	parts := strings.SplitN(raw, "\n---\n", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "---\n") {
		return nil, fmt.Errorf("invalid v1 frontmatter")
	}
	var m v1Meta
	if err := json.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &m); err != nil {
		return nil, err
	}
	return &v1File{Meta: m, Content: strings.TrimSpace(parts[1])}, nil
}

// migrateIfNeeded runs v1 -> v2 once: when a v1 memories/ or refs/ dir exists
// and no fragments/ dir does. Old dirs are renamed to legacy-<date>.
func (s *Store) migrateIfNeeded() error {
	v1Mem := filepath.Join(s.root, "memories")
	v1Refs := filepath.Join(s.root, "refs")
	_, errMem := os.Stat(v1Mem)
	_, errRefs := os.Stat(v1Refs)
	_, errFrags := os.Stat(s.frags)
	hasMem := errMem == nil
	hasRefs := errRefs == nil
	hasFrags := errFrags == nil
	if !hasMem && !hasRefs {
		return nil
	}
	if hasFrags {
		return nil
	}
	if err := s.Migrate(s.root, false); err != nil {
		return err
	}
	return nil
}

// Migrate converts a v1 flat-file store under root into v2 fragments. When
// dryRun is true it only prints a plan (the CLI --memory-migrate-dry-run flag).
func (s *Store) Migrate(root string, dryRun bool) error {
	legacy := filepath.Join(root, "legacy-"+time.Now().UTC().Format("20060102"))
	collected := []*v1File{}
	legacyAliases := map[string]string{}

	scan := func(dir string, isRef bool) error {
		es, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		for _, en := range es {
			if en.IsDir() || filepath.Ext(en.Name()) != ".md" {
				continue
			}
			b, e := os.ReadFile(filepath.Join(dir, en.Name()))
			if e != nil {
				continue
			}
			v1, e := decodeV1(b)
			if e != nil {
				continue
			}
			collected = append(collected, v1)
			_ = isRef
		}
		return nil
	}
	if err := scan(filepath.Join(root, "memories"), false); err != nil {
		return err
	}
	if err := scan(filepath.Join(root, "refs"), true); err != nil {
		return err
	}

	// Build fragments.
	var writes []*Fragment
	for _, v1 := range collected {
		title := v1.Meta.Title
		if title == "" {
			title = "Migrated memory"
		}
		id := slugify(title)
		if !validID(id) {
			id = "mem_note_" + shortHash(title)
		}
		// de-dup ids
		for _, existing := range writes {
			if existing.ID == id {
				id = id + "_" + shortHash(v1.Meta.ID)
				break
			}
		}
		f := &Fragment{
			ID: id, Kind: kindFromCategory(v1.Meta.Category), Title: title,
			Summary: firstN(v1.Content, 280), Body: v1.Content, Tags: v1.Meta.Tags,
			Status: StatusActive, Created: v1.Meta.Timestamp, Updated: v1.Meta.Timestamp,
			Accessed: time.Now().UTC(), Source: orDef(v1.Meta.Source, "ai"),
		}
		if f.Created.IsZero() {
			f.Created = time.Now().UTC()
			f.Updated = f.Created
		}
		applyImportance(f, v1.Meta.Importance)
		legacyAliases[v1.Meta.ID] = f.ID

		// structural edge
		if v1.Meta.Type == "reference" && v1.Meta.TargetID != "" {
			targetSlug := slugifyTitleOnly(v1.Meta.TargetID)
			parent := "mem_" + targetSlug
			if v1.Meta.Relationship != "" {
				f.Links = append(f.Links, Link{Rel: RelChildOf, To: parent, Note: v1.Meta.Relationship, Weight: 1.0})
			} else {
				f.Links = append(f.Links, Link{Rel: RelChildOf, To: parent, Weight: 1.0})
			}
		} else {
			f.Links = append(f.Links, Link{Rel: RelChildOf, To: "mem_inbox", Weight: 1.0})
		}
		for _, ref := range v1.Meta.References {
			f.Links = append(f.Links, Link{Rel: RelRelates, To: "mem_" + slugify(ref), Weight: 1.0})
		}
		writes = append(writes, f)
	}

	if dryRun {
		fmt.Printf("memory migrate dry-run: would create %d fragment(s), %d alias(es)\n", len(writes), len(legacyAliases))
		return nil
	}

	// ensure fragments dir then write all
	if err := os.MkdirAll(s.frags, 0o755); err != nil {
		return err
	}
	for _, f := range writes {
		b, err := encodeFragment(f)
		if err != nil {
			return err
		}
		if err := writeTempThenRename(s.pathFor(f.ID), b); err != nil {
			return err
		}
	}
	// record aliases for one release
	if len(legacyAliases) > 0 {
		al, _ := json.Marshal(legacyAliases)
		_ = os.WriteFile(filepath.Join(s.root, "legacy-aliases.json"), al, 0o644)
	}
	// move old dirs to legacy
	for _, d := range []string{"memories", "refs", "cache"} {
		src := filepath.Join(root, d)
		if _, e := os.Stat(src); e == nil {
			_ = os.Rename(src, filepath.Join(legacy, d))
		}
	}
	return nil
}

func kindFromCategory(cat string) Kind {
	switch strings.ToLower(cat) {
	case "technical":
		return KindComponent
	case "project":
		return KindProject
	case "plans":
		return KindTask
	default:
		return KindNote
	}
}

func applyImportance(f *Fragment, imp string) {
	switch strings.ToLower(imp) {
	case "high":
		f.Pinned = true
		f.Salience = 1.0
	case "medium":
		f.Salience = 0.6
	case "low":
		f.Salience = 0.3
	default:
		f.Salience = 1.0
	}
}

func firstN(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}

func orDef(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// slugifyTitleOnly converts an arbitrary id/target into a mem_ slug using the
// same rules as slugify but tolerating leading "mem_" prefixes.
func slugifyTitleOnly(s string) string {
	s = strings.TrimPrefix(s, "memory_")
	s = strings.TrimPrefix(s, "mem_")
	return strings.TrimPrefix(slugify(s), "mem_")
}

func shortHash(s string) string {
	h := 0
	for _, r := range s {
		h = h*31 + int(r)
		if h < 0 {
			h = -h
		}
	}
	return fmt.Sprintf("%x", h)
}

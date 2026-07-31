package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (s *memoryStore) ensure() error {
	for _, d := range []string{s.primary, s.refs, s.cache} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (s *memoryStore) pathFor(id string) (string, error) {
	if err := validID(id); err != nil {
		return "", err
	}
	for _, d := range []string{s.primary, s.refs} {
		p := filepath.Join(d, id+".md")
		if _, e := os.Stat(p); e == nil {
			return p, nil
		}
	}
	return "", fs.ErrNotExist
}

func (s *memoryStore) read(id string, content bool) (memoryData, error) {
	p, e := s.pathFor(id)
	if e != nil {
		return memoryData{}, e
	}
	st, e := os.Stat(p)
	if e != nil {
		return memoryData{}, e
	}
	if st.Size() > s.maxFile {
		return memoryData{}, fmt.Errorf("memory exceeds file size limit")
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return memoryData{}, e
	}
	m, e := decodeMemory(b)
	if e == nil && content {
		_ = atomicWrite(filepath.Join(s.cache, id+".md"), b)
	}
	if e == nil && !content {
		m.Content = ""
	}
	return m, e
}

func (s *memoryStore) write(m memoryData) error {
	b, e := encodeMemory(m)
	if e != nil {
		return e
	}
	if int64(len(b)) > s.maxFile {
		return fmt.Errorf("memory exceeds file size limit")
	}
	d := s.primary
	if m.Metadata.Type == "reference" {
		d = s.refs
	}
	return atomicWrite(filepath.Join(d, m.Metadata.ID+".md"), b)
}

func (s *memoryStore) walk(ctx context.Context, fn func(string, memoryData) error) error {
	if e := s.ensure(); e != nil {
		return e
	}
	for _, d := range []string{s.primary, s.refs} {
		es, e := os.ReadDir(d)
		if e != nil {
			return e
		}
		for _, x := range es {
			if e := ctx.Err(); e != nil {
				return e
			}
			if x.IsDir() || filepath.Ext(x.Name()) != ".md" {
				continue
			}
			b, e := os.ReadFile(filepath.Join(d, x.Name()))
			if e != nil {
				continue
			}
			if int64(len(b)) > s.maxFile {
				continue
			}
			m, e := decodeMemory(b)
			if e != nil {
				continue
			}
			if e = fn(filepath.Join(d, x.Name()), m); e != nil {
				return e
			}
		}
	}
	return nil
}

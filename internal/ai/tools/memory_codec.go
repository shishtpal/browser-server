package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func encodeMemory(m memoryData) ([]byte, error) {
	h, e := json.MarshalIndent(m.Metadata, "", "  ")
	if e != nil {
		return nil, e
	}
	return []byte("---\n" + string(h) + "\n---\n" + m.Content), nil
}

func decodeMemory(b []byte) (memoryData, error) {
	var m memoryData
	p := strings.SplitN(string(b), "\n---\n", 2)
	if len(p) != 2 || !strings.HasPrefix(p[0], "---\n") {
		return m, fmt.Errorf("invalid memory frontmatter")
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(p[0], "---\n")), &m.Metadata); err != nil {
		return m, err
	}
	m.Content = p[1]
	return m, nil
}

func atomicWrite(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".memory-*")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(n, path)
}

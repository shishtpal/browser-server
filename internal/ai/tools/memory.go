package tools

import (
	"os"
	"path/filepath"

	"browser-server/internal/ai/config"
)

// memoryStore is the on-disk backing store for AI memory tools. The struct is
// defined here so it stays co-located with its constructor; the tool handlers
// and helpers that operate on it live in sibling files within this package.
type memoryStore struct {
	root, primary, refs, cache string
	maxFile                    int64
	maxDepth                   int
	cacheLimit                 int64
}

func newMemoryStore(c config.MemoryConfig) *memoryStore {
	if c.Directory == "" {
		c.Directory = ".memory"
	}
	if c.PrimaryDir == "" {
		c.PrimaryDir = "memories"
	}
	if c.RefsDir == "" {
		c.RefsDir = "refs"
	}
	if c.CacheDir == "" {
		c.CacheDir = "cache"
	}
	if c.MaxFileSizeKB == 0 {
		c.MaxFileSizeKB = 1024
	}
	if c.MaxReferenceDepth == 0 {
		c.MaxReferenceDepth = 5
	}
	if c.CacheSizeLimitMB == 0 {
		c.CacheSizeLimitMB = 100
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	root := filepath.Join(filepath.Dir(exe), c.Directory)
	return &memoryStore{
		root:       root,
		primary:    filepath.Join(root, c.PrimaryDir),
		refs:       filepath.Join(root, c.RefsDir),
		cache:      filepath.Join(root, c.CacheDir),
		maxFile:    int64(c.MaxFileSizeKB) * 1024,
		maxDepth:   c.MaxReferenceDepth,
		cacheLimit: int64(c.CacheSizeLimitMB) * 1024 * 1024,
	}
}

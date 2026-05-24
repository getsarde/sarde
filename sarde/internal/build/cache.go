package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/internal/engine"
)

// PageCache caches markdown render results in .cache/pages/ to speed up
// dev-mode rebuilds. Keyed by sha256 of raw markdown content.
type PageCache struct {
	Dir string // e.g., ".cache/pages"
}

// CacheEntry holds the cached result of rendering a page's markdown.
type CacheEntry struct {
	ContentHash   string                 `json:"content_hash"`
	HTML          string                 `json:"html"`
	Headings      []engine.Heading       `json:"headings"`
	HasCodeBlocks bool                   `json:"has_code_blocks,omitempty"`
	HasImages     bool                   `json:"has_images,omitempty"`
	Links         []engine.CollectedLink `json:"links,omitempty"`
}

// NewPageCache creates a PageCache in the given project directory.
func NewPageCache(projectDir string) *PageCache {
	return &PageCache{Dir: filepath.Join(projectDir, ".cache", "pages")}
}

// Get retrieves a cached entry by content hash. Returns nil if not found.
func (c *PageCache) Get(hash string) *CacheEntry {
	path := filepath.Join(c.Dir, hash+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	return &entry
}

// Put stores a cache entry keyed by content hash.
func (c *PageCache) Put(hash string, entry *CacheEntry) {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(c.Dir, hash+".json"), data, 0o644)
}

// ContentHash returns the sha256 hex digest of raw content.
func ContentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

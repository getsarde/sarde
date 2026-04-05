package asset

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Cache stores processed asset results keyed by source hash + processing params.
// Entries are stored as JSON files in the .cache/ directory.
type Cache struct {
	Dir string // e.g., {ProjectDir}/.cache/images
}

// CacheEntry holds cached image processing results.
type CacheEntry struct {
	SourceHash string         `json:"source_hash"`
	Params     string         `json:"params"`
	Variants   []ImageVariant `json:"variants"`
	LQIP       string         `json:"lqip"` // base64 data URI
}

// NewCache creates a cache rooted at {projectDir}/.cache/images.
func NewCache(projectDir string) *Cache {
	return &Cache{
		Dir: filepath.Join(projectDir, ".cache", "images"),
	}
}

// Key generates a cache key from a source hash and processing parameters.
func (c *Cache) Key(sourceHash, params string) string {
	combined := sourceHash + ":" + params
	h := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", h[:8])
}

// Get retrieves a cached entry. Returns nil if the entry does not exist.
func (c *Cache) Get(key string) (*CacheEntry, error) {
	path := filepath.Join(c.Dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Put stores a processing result in the cache.
func (c *Cache) Put(key string, entry *CacheEntry) error {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.Dir, key+".json"), data, 0o644)
}

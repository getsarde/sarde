package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/frostybee/sarde/internal/engine"
)

const DefaultPageCacheCapacity = 512

type lruNode struct {
	key   string
	entry *CacheEntry
	prev  *lruNode
	next  *lruNode
}

// PageCache is a two-layer (in-memory LRU + filesystem) content-addressed
// cache for rendered markdown. Safe for concurrent use.
type PageCache struct {
	Dir      string
	capacity int

	mu    sync.Mutex
	items map[string]*lruNode
	head  *lruNode // sentinel (MRU end)
	tail  *lruNode // sentinel (LRU end)
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

// NewPageCache creates a PageCache with the default capacity.
func NewPageCache(projectDir string) *PageCache {
	c := &PageCache{
		Dir:      filepath.Join(projectDir, ".cache", "pages"),
		capacity: DefaultPageCacheCapacity,
		items:    make(map[string]*lruNode, DefaultPageCacheCapacity),
	}
	c.head = &lruNode{}
	c.tail = &lruNode{}
	c.head.next = c.tail
	c.tail.prev = c.head
	return c
}

// Get retrieves a cached entry by content hash.
// Checks in-memory LRU first, then falls through to filesystem.
//
// The returned *CacheEntry (including its Headings/Links slices) is shared with
// the in-memory LRU and, on a hit, with every other caller of the same hash.
// Callers MUST treat it as read-only: mutating it (e.g. appending to Headings)
// would corrupt the cache and other pages. Copy before mutating if needed.
func (c *PageCache) Get(hash string) *CacheEntry {
	c.mu.Lock()
	if node, ok := c.items[hash]; ok {
		c.moveToFront(node)
		c.mu.Unlock()
		return node.entry
	}
	c.mu.Unlock()

	path := filepath.Join(c.Dir, hash+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	c.mu.Lock()
	if node, ok := c.items[hash]; ok {
		c.moveToFront(node)
		c.mu.Unlock()
		return node.entry
	}
	c.insertFront(hash, &entry)
	c.mu.Unlock()

	return &entry
}

// Put stores a cache entry in both in-memory LRU and filesystem.
func (c *PageCache) Put(hash string, entry *CacheEntry) {
	c.mu.Lock()
	if node, ok := c.items[hash]; ok {
		node.entry = entry
		c.moveToFront(node)
	} else {
		c.insertFront(hash, entry)
	}
	c.mu.Unlock()

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

// insertFront adds a new node at the MRU position. Evicts LRU if at capacity.
// Must be called with c.mu held.
func (c *PageCache) insertFront(hash string, entry *CacheEntry) {
	if len(c.items) >= c.capacity {
		c.evictTail()
	}
	node := &lruNode{key: hash, entry: entry}
	node.next = c.head.next
	node.prev = c.head
	c.head.next.prev = node
	c.head.next = node
	c.items[hash] = node
}

// moveToFront promotes a node to the MRU position.
// Must be called with c.mu held.
func (c *PageCache) moveToFront(node *lruNode) {
	if node.prev == c.head {
		return
	}
	node.prev.next = node.next
	node.next.prev = node.prev
	node.next = c.head.next
	node.prev = c.head
	c.head.next.prev = node
	c.head.next = node
}

// evictTail removes the LRU node (disk file is left intact).
// Must be called with c.mu held.
func (c *PageCache) evictTail() {
	lru := c.tail.prev
	if lru == c.head {
		return
	}
	lru.prev.next = c.tail
	c.tail.prev = lru.prev
	delete(c.items, lru.key)
}

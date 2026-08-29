package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
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
	ContentHash    string                 `json:"content_hash"`
	HTML           string                 `json:"html"`
	Headings       []engine.Heading       `json:"headings"`
	HasCodeBlocks  bool                   `json:"has_code_blocks,omitempty"`
	HasImages      bool                   `json:"has_images,omitempty"`
	Links          []engine.CollectedLink `json:"links,omitempty"`
	PendingAnchors []CachedAnchorCheck    `json:"pending_anchors,omitempty"`
	Refs           []CachedLinkRef        `json:"refs,omitempty"`
}

// CachedAnchorCheck holds the content-derived fields of a pending anchor
// check (links.PendingAnchorCheck) that are safe to persist across builds.
// Page-derived fields (source file, from/target page pointers, dimension)
// are reconstructed from the live page at cache-hit time; see
// replayPendingAnchors in helpers.go.
type CachedAnchorCheck struct {
	TargetPermalink    string         `json:"target_permalink"`
	TargetRelPermalink string         `json:"target_rel_permalink,omitempty"`
	Fragment           string         `json:"fragment"`
	RawHref            string         `json:"raw_href"`
	Kind               links.LinkKind `json:"kind"`
	Resolved           string         `json:"resolved"`
}

// CachedLinkRef holds the content-derived fields of a recorded link ref
// (links.LinkRef) that are safe to persist across builds. On a cache hit the
// snapshot is first verified against the live page index (see entryStale in
// helpers.go); any divergence demotes the hit to a miss so the page re-renders
// with fresh URLs instead of replaying stale results.
type CachedLinkRef struct {
	RawDest            string           `json:"raw_dest"`
	Kind               links.LinkKind   `json:"kind"`
	Resolved           string           `json:"resolved"`
	TargetPermalink    string           `json:"target_permalink,omitempty"`
	TargetRelPermalink string           `json:"target_rel_permalink,omitempty"`
	Fragment           string           `json:"fragment,omitempty"`
	Status             links.LinkStatus `json:"status"`
	Line               int              `json:"line,omitempty"`
	Col                int              `json:"col,omitempty"`
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

// pageCacheSchemaVersion is folded into every page-cache key. Bump it whenever
// CacheEntry gains validation-bearing fields or the cached values are computed
// differently: entries written under an older layout unmarshal cleanly but
// replay incomplete or stale state (an entry without pending_anchors would
// keep skipping anchor re-validation; entries from before custom heading IDs
// were honored carry clobbered ids in Headings; entries without refs would
// replay zero link refs forever, reproducing the coverage undercount; entries
// from before markdown.hard_wraps became configurable carry <br> for every
// soft line break; entries from before markdown.asides.style became
// configurable carry the classic aside icons regardless of style).
const pageCacheSchemaVersion = "8"

// pageCacheKey builds the content-addressed key for a rendered page. Both the
// parallel and serial render paths must use this single helper so the key
// composition cannot drift between them.
func pageCacheKey(processed, shortcodesHash, resolutionKey, iconRenderKey, rendererKey, lang string) string {
	return ContentHash(processed + shortcodesHash + resolutionKey + iconRenderKey + rendererKey +
		"\x00lang=" + lang + "\x00schema=" + pageCacheSchemaVersion)
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

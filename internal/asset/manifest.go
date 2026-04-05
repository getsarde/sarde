package asset

import "sync"

// ImageVariant represents a single processed image variant (width + format).
type ImageVariant struct {
	Width    int    `json:"width"`
	Format   string `json:"format"`   // "jpeg", "png", "webp"
	URL      string `json:"url"`      // output URL path
	FileSize int64  `json:"fileSize"` // bytes
}

// ManifestEntry maps an original asset path to its processed output.
type ManifestEntry struct {
	OriginalPath string         // original asset path as referenced
	OutputPath   string         // relative to output dir (e.g., "assets/main.a1b2c3d4.css")
	OutputURL    string         // URL path (e.g., "/assets/main.a1b2c3d4.css")
	Hash         string         // content hash
	Variants     []ImageVariant // image variants (Phase 10b)
	LQIP         string         // base64 LQIP data URI (Phase 10b)
}

// Manifest maps original asset paths to their output (potentially fingerprinted) paths.
type Manifest struct {
	mu      sync.RWMutex
	entries map[string]ManifestEntry
}

// NewManifest creates an empty asset manifest.
func NewManifest() *Manifest {
	return &Manifest{
		entries: make(map[string]ManifestEntry),
	}
}

// Add registers an asset mapping in the manifest.
func (m *Manifest) Add(original string, entry ManifestEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[original] = entry
}

// Lookup retrieves the manifest entry for an original asset path.
func (m *Manifest) Lookup(original string) (ManifestEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[original]
	return e, ok
}

// Len returns the number of entries in the manifest.
func (m *Manifest) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

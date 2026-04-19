package plugin

// SharedStore is a simple key-value store for cross-hook data passing.
// Not thread-safe — only used in serial hook execution (ConfigSetup, ContentLoaded, BeforeRender).
type SharedStore struct {
	data map[string]any
}

// NewStore creates an empty shared store.
func NewStore() *SharedStore {
	return &SharedStore{data: make(map[string]any)}
}

// Set stores a value by key.
func (s *SharedStore) Set(key string, value any) {
	s.data[key] = value
}

// Get retrieves a value by key. Returns nil if not found.
func (s *SharedStore) Get(key string) any {
	return s.data[key]
}

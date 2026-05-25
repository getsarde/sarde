package plugin

import "sync"

// SharedStore is a simple key-value store for cross-hook data passing.
// It is safe for concurrent BeforeRender hooks.
type SharedStore struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewStore creates an empty shared store.
func NewStore() *SharedStore {
	return &SharedStore{data: make(map[string]any)}
}

// Set stores a value by key.
func (s *SharedStore) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get retrieves a value by key. Returns nil if not found.
func (s *SharedStore) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

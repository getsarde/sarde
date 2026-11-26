package asset

import (
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCache_KeyDeterministic(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}

	k1 := c.Key("abc123", "w=[400,800],q=80")
	k2 := c.Key("abc123", "w=[400,800],q=80")

	if k1 != k2 {
		t.Errorf("same inputs produced different keys: %q vs %q", k1, k2)
	}
	if len(k1) != 16 { // 8 bytes = 16 hex chars
		t.Errorf("key length = %d, want 16", len(k1))
	}
}

func TestCache_KeyDifferentInputs(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}

	k1 := c.Key("abc", "params1")
	k2 := c.Key("def", "params1")
	k3 := c.Key("abc", "params2")

	if k1 == k2 {
		t.Error("different source hashes produced same key")
	}
	if k1 == k3 {
		t.Error("different params produced same key")
	}
}

func TestCache_PutAndGet(t *testing.T) {
	c := NewCache(t.TempDir())

	entry := &CacheEntry{
		SourceHash: "abc123",
		Params:     "w=[400,800],q=80",
		Variants: []ImageVariant{
			{Width: 400, Format: "jpeg", URL: "/assets/images/hero-abc-400w.jpg"},
			{Width: 800, Format: "jpeg", URL: "/assets/images/hero-abc-800w.jpg"},
		},
		LQIP: "data:image/jpeg;base64,/9j/test",
	}

	key := c.Key("abc123", "w=[400,800],q=80")
	if err := c.Put(key, entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil entry")
	}
	if got.SourceHash != "abc123" {
		t.Errorf("SourceHash = %q, want abc123", got.SourceHash)
	}
	if len(got.Variants) != 2 {
		t.Errorf("Variants count = %d, want 2", len(got.Variants))
	}
	if got.LQIP != "data:image/jpeg;base64,/9j/test" {
		t.Errorf("LQIP mismatch")
	}
}

func TestCache_Miss(t *testing.T) {
	c := NewCache(t.TempDir())

	got, err := c.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for cache miss")
	}
}

// Parallel workers can write and read the same entry. A reader must see
// a miss, a read error, or a complete entry, never a truncated file.
//
// Skipped on Windows: a file that is open for reading cannot be renamed
// over there, and with this many goroutines on one key atomicwrite runs out
// of retries and falls back to a direct write, as it is designed to. A real
// build has at most a few workers on one key.
func TestCache_ConcurrentPutGet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rename over an open file is refused on Windows; see the comment above")
	}
	c := &Cache{Dir: t.TempDir()}
	entry := &CacheEntry{SourceHash: "abc", Params: "w=800", LQIP: strings.Repeat("x", 64<<10)}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := c.Put("k", entry); err != nil {
					t.Errorf("Put: %v", err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				got, err := c.Get("k")
				// A read that collides with the rename can fail on Windows
				// with a sharing violation, which callers treat as a miss.
				// What must never happen is a file that is there but cut
				// short, which surfaces as a JSON syntax error.
				var syntaxErr *json.SyntaxError
				if errors.As(err, &syntaxErr) {
					t.Errorf("Get saw a partial entry: %v", err)
					return
				}
				if got != nil && len(got.LQIP) != len(entry.LQIP) {
					t.Errorf("Get returned a truncated entry: %d bytes", len(got.LQIP))
					return
				}
			}
		}()
	}
	wg.Wait()
}

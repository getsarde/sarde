package asset

import (
	"os"
	"path/filepath"
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

// An entry whose file is cut short is a miss, and the next Put repairs it.
func TestCache_TruncatedEntryIsMiss(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(c.Dir, "k.json"), []byte(`{"source_hash":"abc","par`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("k")
	if err != nil || got != nil {
		t.Fatalf("Get on a truncated entry = %v, %v; want nil, nil", got, err)
	}

	if err := c.Put("k", &CacheEntry{SourceHash: "abc"}); err != nil {
		t.Fatal(err)
	}
	got, err = c.Get("k")
	if err != nil || got == nil || got.SourceHash != "abc" {
		t.Fatalf("Get after Put = %v, %v; want the entry", got, err)
	}
}

// Parallel workers can write and read the same entry. A reader must see a
// miss or a complete entry, never part of one. A read that collides with a
// write can also fail outright on Windows (a sharing violation), which the
// caller treats as a miss, so read errors are ignored here.
func TestCache_ConcurrentPutGet(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}
	entry := &CacheEntry{SourceHash: "abc", Params: "w=800", LQIP: strings.Repeat("x", 64<<10)}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				// A write can be refused on Windows while readers hold
				// the file open. The image pipeline only warns about a
				// failed cache write, so it is not a failure here either.
				_ = c.Put("k", entry)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				got, _ := c.Get("k")
				if got != nil && len(got.LQIP) != len(entry.LQIP) {
					t.Errorf("Get returned a truncated entry: %d bytes", len(got.LQIP))
					return
				}
			}
		}()
	}
	wg.Wait()

	// With no contention left, a write succeeds and reads back whole.
	if err := c.Put("k", entry); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := c.Get("k")
	if err != nil || got == nil || len(got.LQIP) != len(entry.LQIP) {
		t.Fatalf("Get after the last Put = %v, %v; want the complete entry", got != nil, err)
	}
}

package asset

import (
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

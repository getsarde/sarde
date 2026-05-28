package config

import "testing"

func TestMergeStringMap_DeepMerge(t *testing.T) {
	// Simulates a theme layer providing redirects and the user adding more:
	// both layers' keys must survive, with the later layer winning per key.
	base := map[string]string{"/old-a": "/a", "/shared": "/theme"}
	mergeStringMap(&base, map[string]string{"/old-b": "/b", "/shared": "/user"})

	if len(base) != 3 {
		t.Fatalf("expected 3 keys after merge, got %d: %v", len(base), base)
	}
	if base["/old-a"] != "/a" {
		t.Errorf("/old-a = %q, want /a (base-layer key preserved)", base["/old-a"])
	}
	if base["/old-b"] != "/b" {
		t.Errorf("/old-b = %q, want /b (over-layer key added)", base["/old-b"])
	}
	if base["/shared"] != "/user" {
		t.Errorf("/shared = %q, want /user (later layer wins per key)", base["/shared"])
	}
}

func TestMergeStringMap_NilBaseAllocated(t *testing.T) {
	var base map[string]string
	mergeStringMap(&base, map[string]string{"/a": "/x"})
	if base == nil || base["/a"] != "/x" {
		t.Fatalf("nil base should be allocated and merged, got %v", base)
	}
}

func TestMergeStringMap_EmptyOverNoOp(t *testing.T) {
	base := map[string]string{"/a": "/x"}
	mergeStringMap(&base, nil)
	if len(base) != 1 || base["/a"] != "/x" {
		t.Errorf("empty over must not alter base, got %v", base)
	}
}

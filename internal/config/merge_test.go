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

func boolPtr(b bool) *bool { return &b }

// Regression: mergeBuild used to omit Expired and Cache, silently dropping
// build.expired / build.cache set in sarde.yaml or theme.yaml.
func TestMergeBuild_ExpiredAndCache(t *testing.T) {
	base := &BuildSettings{}
	over := &BuildSettings{Expired: boolPtr(true), Cache: boolPtr(false)}
	mergeBuild(base, over)

	if base.Expired == nil || *base.Expired != true {
		t.Errorf("Expired not merged: %v", base.Expired)
	}
	if base.Cache == nil || *base.Cache != false {
		t.Errorf("Cache not merged: %v", base.Cache)
	}
}

// Regression: mergeI18n used to omit Strict, silently dropping i18n.strict.
func TestMergeI18n_Strict(t *testing.T) {
	base := &I18nSettings{}
	mergeI18n(base, &I18nSettings{Strict: true})
	if !base.Strict {
		t.Error("Strict not merged from override layer")
	}
	// A lower-layer true must not be cleared by a higher layer that omits it.
	base2 := &I18nSettings{Strict: true}
	mergeI18n(base2, &I18nSettings{})
	if !base2.Strict {
		t.Error("Strict cleared by a layer that did not set it")
	}
}

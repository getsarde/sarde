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

func intPtr(i int) *int { return &i }

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

func TestMergePlugins_Disabled(t *testing.T) {
	base := &PluginSettings{Enabled: []string{"search"}, Disabled: []string{"seo"}}
	mergePlugins(base, &PluginSettings{Disabled: []string{"slideviewer"}})
	if len(base.Disabled) != 1 || base.Disabled[0] != "slideviewer" {
		t.Errorf("Disabled not replaced by overriding layer: %v", base.Disabled)
	}
	if len(base.Enabled) != 1 || base.Enabled[0] != "search" {
		t.Errorf("Enabled should be untouched by empty override: %v", base.Enabled)
	}

	mergePlugins(base, &PluginSettings{})
	if len(base.Disabled) != 1 || base.Disabled[0] != "slideviewer" {
		t.Errorf("empty override should keep base Disabled: %v", base.Disabled)
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

// Regression: mergeInt treated 0 as "unset", so an explicit
// prefetch.delay: 0 in user config silently fell back to the embedded
// default (300) instead of being honored. Prefetch.Delay is now a *int
// merged via mergeIntP so an explicit zero is distinguishable from "unset".
func TestMergePrefetch_ExplicitZeroDelayWins(t *testing.T) {
	base := &PrefetchSettings{Delay: intPtr(300)}
	over := &PrefetchSettings{Delay: intPtr(0)}
	mergePrefetch(base, over)

	if base.Delay == nil || *base.Delay != 0 {
		t.Errorf("Delay = %v, want 0 (explicit zero override must not fall back to base)", base.Delay)
	}
}

func TestMergePrefetch_NilDelayKeepsBase(t *testing.T) {
	base := &PrefetchSettings{Delay: intPtr(300)}
	over := &PrefetchSettings{Delay: nil}
	mergePrefetch(base, over)

	if base.Delay == nil || *base.Delay != 300 {
		t.Errorf("Delay = %v, want 300 (omitted override must keep base value)", base.Delay)
	}
}

func TestMergeLogo_ReplacesTitleCascades(t *testing.T) {
	// Embedded defaults ship replaces_title: false; a user setting true must win,
	// and a user layer that omits the key must leave the base value intact.
	f := false
	tr := true

	base := Logo{Light: "/theme-logo.svg", ReplacesTitle: &f}
	mergeLogo(&base, &Logo{Light: "/user-logo.svg", ReplacesTitle: &tr})
	if base.Light != "/user-logo.svg" {
		t.Errorf("Light = %q, want /user-logo.svg", base.Light)
	}
	if !BoolVal(base.ReplacesTitle, false) {
		t.Error("ReplacesTitle = false, want true (user layer wins)")
	}

	base = Logo{Light: "/theme-logo.svg", Alt: "Theme", ReplacesTitle: &tr}
	mergeLogo(&base, &Logo{})
	if base.Alt != "Theme" {
		t.Errorf("Alt = %q, want Theme (empty over must not clobber)", base.Alt)
	}
	if !BoolVal(base.ReplacesTitle, false) {
		t.Error("ReplacesTitle = false, want true (nil over must not clobber)")
	}
}

func TestMergeMarkdown_AsideStyle(t *testing.T) {
	base := &MarkdownSettings{Asides: AsidesSettings{Style: "classic"}}
	over := &MarkdownSettings{Asides: AsidesSettings{Style: "galaxy"}}
	mergeMarkdown(base, over)
	if base.Asides.Style != "galaxy" {
		t.Errorf("Asides.Style = %q, want %q (override must win)", base.Asides.Style, "galaxy")
	}

	base = &MarkdownSettings{Asides: AsidesSettings{Style: "classic"}}
	over = &MarkdownSettings{}
	mergeMarkdown(base, over)
	if base.Asides.Style != "classic" {
		t.Errorf("Asides.Style = %q, want %q (empty override must keep base)", base.Asides.Style, "classic")
	}
}

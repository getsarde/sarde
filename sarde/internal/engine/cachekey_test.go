package engine

import "testing"

func TestURLResolverCacheKey_DiffersOnBasePath(t *testing.T) {
	a := &URLResolver{BasePath: "/", BaseURL: "https://example.com"}
	b := &URLResolver{BasePath: "/web-course/", BaseURL: "https://example.com"}

	if a.CacheKey() == b.CacheKey() {
		t.Error("CacheKey must differ when BasePath differs")
	}
}

func TestURLResolverCacheKey_StableAndOrderIndependent(t *testing.T) {
	a := &URLResolver{
		BasePath:         "/",
		I18nEnabled:      true,
		DefaultLang:      "en",
		Strategy:         "prefix-except-default",
		Languages:        map[string]bool{"en": true, "fr": true, "ar": true},
		CollectionMounts: []string{"/docs", "/blog"},
		VersionIDs:       map[string]bool{"v1": true, "v2": true},
	}
	// Same logical config, different map/slice insertion order.
	b := &URLResolver{
		BasePath:         "/",
		I18nEnabled:      true,
		DefaultLang:      "en",
		Strategy:         "prefix-except-default",
		Languages:        map[string]bool{"ar": true, "fr": true, "en": true},
		CollectionMounts: []string{"/blog", "/docs"},
		VersionIDs:       map[string]bool{"v2": true, "v1": true},
	}

	if a.CacheKey() != b.CacheKey() {
		t.Error("CacheKey must be stable regardless of map/slice ordering")
	}
	if a.CacheKey() != a.CacheKey() {
		t.Error("CacheKey must be deterministic across calls")
	}
}

func TestURLResolverCacheKey_DiffersOnLanguageSet(t *testing.T) {
	a := &URLResolver{Languages: map[string]bool{"en": true}}
	b := &URLResolver{Languages: map[string]bool{"en": true, "fr": true}}

	if a.CacheKey() == b.CacheKey() {
		t.Error("CacheKey must differ when the language set differs")
	}
}

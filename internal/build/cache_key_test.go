package build

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// Two bundle pages with the same body render different image URLs, since
// each page's files sit beside it, so their cache keys differ. Pages without
// resources keep sharing a key.
func TestPageCacheKey_BundlePagesKeyedByLocation(t *testing.T) {
	docs := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{
		Versioning: &engine.VersionConfig{Enabled: true, LastVersion: "v2"},
	}}
	page := func(version string, resources bool) *engine.Page {
		p := &engine.Page{
			PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/"},
			PageVersioning:    engine.PageVersioning{Version: version},
			PageRelationships: engine.PageRelationships{Collection: docs},
		}
		if resources {
			p.Resources = []engine.Resource{{Name: "flow.svg"}}
		}
		return p
	}
	key := func(p *engine.Page) string { return pageCacheKey("body", "sc", "res", "icons", "renderer", p) }

	if key(page("v1", true)) == key(page("v2", true)) {
		t.Error("v1 and v2 bundle pages with the same body share a cache key")
	}
	if key(page("v1", false)) != key(page("v2", false)) {
		t.Error("pages without resources no longer share a cache key")
	}
}

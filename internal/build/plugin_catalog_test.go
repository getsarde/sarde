package build

import (
	"testing"

	"github.com/getsarde/sarde/internal/plugin/catalog"
)

// The catalog published by `sarde plugins` must describe every name the
// config validator accepts, so Studio never sees a plugin it cannot render.
func TestPluginCatalog_CoversKnownPluginNames(t *testing.T) {
	ids := map[string]bool{}
	for _, id := range catalog.Build("").IDs() {
		ids[id] = true
	}
	for _, name := range KnownPluginNames("") {
		if !ids[name] {
			t.Errorf("plugin %q is known to the build but missing from the catalog", name)
		}
	}
}

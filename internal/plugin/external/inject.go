package external

import (
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
)

// injectMatches reports whether the manifest's inject condition holds for the
// given page. It extends the shared inject-rule vocabulary with the
// parameterized layout and collection rules.
func injectMatches(inj *InjectConfig, page *engine.Page, rd *engine.RouteData) bool {
	switch inj.When {
	case "", "always":
		return true
	case "layout":
		return string(rd.Layout) == inj.Layout
	case "collection":
		return page.Collection != nil && page.Collection.Name == inj.Collection
	default:
		return plugin.MatchesInjectRule(inj.When, page, rd)
	}
}

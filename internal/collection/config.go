package collection

import (
	"strings"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// MergeCollectionConfig overlays site.yaml collection values onto inferred defaults.
// Non-zero/non-nil site config values win over inferred values.
// Returns a new CollectionConfig (does not mutate the input).
func MergeCollectionConfig(inferred *engine.CollectionConfig, siteCfg *config.CollectionSiteConfig) *engine.CollectionConfig {
	if siteCfg == nil {
		return inferred
	}

	// Clone the inferred config
	merged := *inferred

	if siteCfg.Sort != "" {
		sortBy, sortOrder := parseSortString(siteCfg.Sort)
		if sortBy != "" {
			merged.SortBy = sortBy
		}
		if sortOrder != "" {
			merged.SortOrder = sortOrder
		}
	}

	if siteCfg.Layout != "" {
		merged.Layout = engine.ResolveLayout(siteCfg.Layout)
	}
	if siteCfg.Permalink != "" {
		merged.Permalink = siteCfg.Permalink
	}
	if siteCfg.Paginate != 0 {
		merged.Paginate = siteCfg.Paginate
	}
	if siteCfg.Feed != nil {
		merged.Feed = *siteCfg.Feed
	}

	// Sidebar merge
	if siteCfg.Sidebar != nil {
		if merged.Sidebar == nil {
			merged.Sidebar = &engine.SidebarConfig{}
		}
		s := *merged.Sidebar // clone
		if siteCfg.Sidebar.Collapsible != nil {
			s.Collapsible = *siteCfg.Sidebar.Collapsible
		}
		if siteCfg.Sidebar.CollapsedByDefault != nil {
			s.CollapsedByDefault = *siteCfg.Sidebar.CollapsedByDefault
		}
		if siteCfg.Sidebar.MaxDepth != 0 {
			s.MaxDepth = siteCfg.Sidebar.MaxDepth
		}
		if siteCfg.Sidebar.Search != nil {
			s.Search = *siteCfg.Sidebar.Search
		}
		merged.Sidebar = &s
	}

	// TOC merge
	if siteCfg.TOC != nil {
		if merged.TOC == nil {
			merged.TOC = &engine.TOCConfig{}
		}
		tc := *merged.TOC // clone
		if siteCfg.TOC.Enabled != nil {
			tc.Enabled = *siteCfg.TOC.Enabled
		}
		if siteCfg.TOC.Depth != 0 {
			tc.MaxLevel = siteCfg.TOC.Depth
		}
		if siteCfg.TOC.ScrollHighlight != nil {
			tc.ScrollHighlight = *siteCfg.TOC.ScrollHighlight
		}
		merged.TOC = &tc
	}

	// PrevNext merge
	if siteCfg.PrevNext != nil {
		if merged.PrevNext == nil {
			merged.PrevNext = &engine.PrevNextConfig{}
		}
		pn := *merged.PrevNext // clone
		if siteCfg.PrevNext.Enabled != nil {
			pn.Enabled = *siteCfg.PrevNext.Enabled
		}
		if len(siteCfg.PrevNext.Labels) == 2 {
			pn.Labels = [2]string{siteCfg.PrevNext.Labels[0], siteCfg.PrevNext.Labels[1]}
		}
		merged.PrevNext = &pn
	}

	return &merged
}

// parseSortString splits "date desc" into ("date", "desc").
// If only one word, returns (word, ""). If empty, returns ("", "").
func parseSortString(s string) (sortBy, sortOrder string) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", ""
	}
	sortBy = parts[0]
	if len(parts) >= 2 {
		sortOrder = parts[1]
	}
	return
}

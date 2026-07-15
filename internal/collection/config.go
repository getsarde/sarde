package collection

import (
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// MergeCollectionConfig overlays sarde.yaml collection values onto inferred defaults.
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
	if siteCfg.Tabs != nil {
		merged.Tabs = siteCfg.Tabs
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
		if siteCfg.Sidebar.CollapseLevel != nil {
			s.CollapseLevel = *siteCfg.Sidebar.CollapseLevel
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

	// Versioning merge
	if siteCfg.Versioning != nil && config.BoolVal(siteCfg.Versioning.Enabled, false) {
		vc := &engine.VersionConfig{
			Enabled:                   true,
			LastVersion:               siteCfg.Versioning.LastVersion,
			PublishLatestAtVersionURL: siteCfg.Versioning.PublishLatestAtVersionURL,
		}
		for _, v := range siteCfg.Versioning.Versions {
			path := v.Path
			if path == "" {
				path = v.ID
			}
			banner := string(v.Banner)
			if banner == "" {
				banner = "none"
			}
			redirect := string(v.Redirect)
			if redirect == "" {
				redirect = "same-page"
			}
			vc.Versions = append(vc.Versions, engine.VersionDef{
				ID:       v.ID,
				Label:    v.Label,
				Path:     path,
				Banner:   banner,
				Redirect: redirect,
			})
		}
		merged.Versioning = vc
	}

	return &merged
}

// ApplySidebarFile overlays one collection's sidebar.yaml entry onto its
// resolved config. sidebar.yaml wins over sarde.yaml and frontmatter per the
// documented precedence chain. Returns a new CollectionConfig (never mutates
// the input) so it composes after MergeCollectionConfig. entry may be nil.
func ApplySidebarFile(cfg *engine.CollectionConfig, entry *config.SidebarCollectionEntry) *engine.CollectionConfig {
	if entry == nil {
		return cfg
	}
	if len(entry.Overrides) == 0 && len(entry.Tabs) == 0 && entry.CollapseLevel == nil {
		return cfg
	}

	merged := *cfg
	sb := engine.SidebarConfig{}
	if merged.Sidebar != nil {
		sb = *merged.Sidebar
	}

	if entry.CollapseLevel != nil {
		sb.CollapseLevel = *entry.CollapseLevel
	}

	if len(entry.Overrides) > 0 {
		sb.Overrides = make(map[string]*engine.SidebarOverride, len(entry.Overrides))
		for key, ov := range entry.Overrides {
			if ov == nil {
				continue
			}
			sb.Overrides[normalizeOverrideKey(key)] = &engine.SidebarOverride{
				Label:       ov.Label,
				Description: ov.Description,
				Order:       ov.Order,
				Collapsed:   ov.Collapsed,
				Icon:        ov.Icon,
				Badge:       ov.Badge,
				Hidden:      ov.Hidden,
				Attrs:       ov.Attrs,
			}
		}
	}

	if len(entry.Tabs) > 0 {
		sb.TabOverrides = make(map[string]*engine.TabOverride, len(entry.Tabs))
		for slug, ov := range entry.Tabs {
			if ov == nil {
				continue
			}
			sb.TabOverrides[normalizeOverrideKey(slug)] = &engine.TabOverride{
				Label:       ov.Label,
				Description: ov.Description,
				Icon:        ov.Icon,
				Order:       ov.Order,
			}
		}
	}

	merged.Sidebar = &sb
	return &merged
}

// normalizeOverrideKey canonicalizes a sidebar.yaml path key: slashes are
// forward, no leading or trailing slash.
func normalizeOverrideKey(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")
	return strings.Trim(key, "/")
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

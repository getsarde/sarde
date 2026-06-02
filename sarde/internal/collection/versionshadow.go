package collection

import "github.com/frostybee/sarde/internal/engine"

// ShadowDrop records a loose page dropped because it shadowed the latest-version
// page at the same canonical URL within a versioned collection.
type ShadowDrop struct {
	Dropped *engine.Page // loose page (no version directory) that was removed
	Owner   *engine.Page // latest-version page that owns the bare URL
}

// ResolveVersionShadowing removes loose (Version=="") pages whose (Lang, RelPermalink)
// is already owned by the lastVersion page of the same versioned collection.
//
// In a versioned collection the latest version drops its /vN/ segment and serves at
// the bare collection URL (e.g. docs/v2/guide/x.md → /docs/guide/x/). A "loose" page
// placed directly under the collection root resolves to that same bare URL, silently
// shadowing it: two source files, one URL. The latest-version page is the documented
// owner of the bare URL, so it wins deterministically and the loose page is dropped.
//
// Surgical: a loose page with no lastVersion twin (a genuine version-independent page)
// is kept; older versions keep their /vN/ segment and never collide, so are never
// touched. kept preserves input order; the result is independent of map iteration
// order, hence deterministic.
func ResolveVersionShadowing(pages []*engine.Page, lastVersion string) (kept []*engine.Page, drops []ShadowDrop) {
	if lastVersion == "" || len(pages) == 0 {
		return pages, nil
	}

	type laneURL struct {
		lang string
		rel  string
	}
	owners := make(map[laneURL]*engine.Page)
	for _, p := range pages {
		if p.Version == lastVersion {
			owners[laneURL{p.Lang, p.RelPermalink}] = p
		}
	}
	if len(owners) == 0 {
		return pages, nil
	}

	kept = make([]*engine.Page, 0, len(pages))
	for _, p := range pages {
		if p.Version == "" {
			if owner, ok := owners[laneURL{p.Lang, p.RelPermalink}]; ok && owner != p {
				if p.Kind == engine.KindSection {
					// Section pages are structural (section tree, IndexPage, nav) so
					// they stay in col.Pages, but they must not be rendered to disk —
					// the latest-version's section page owns the output path. Setting
					// render=false uses the existing renderablePages filter in builder.go.
					if p.Params == nil {
						p.Params = make(map[string]any)
					}
					p.Params["render"] = false
				} else {
					drops = append(drops, ShadowDrop{Dropped: p, Owner: owner})
					continue
				}
			}
		}
		kept = append(kept, p)
	}
	return kept, drops
}

package i18n

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
)

// FallbackOptions controls fallback generation behaviour.
type FallbackOptions struct {
	// SiteFallback is the site-level policy: "default" or "omit".
	SiteFallback string

	// CollectionFallback maps collection name → override policy.
	CollectionFallback map[string]string
}

// GenerateFallbacks creates fallback pages for non-default languages where
// a page exists in the default language but has no translation.
// Per-collection i18n_fallback:"omit" suppresses fallbacks for that collection.
// Returns the newly created fallback pages (caller should append to allPages).
//
// For versioned pages, fallback operates within a version (no cross-version
// fallback). The grouping key includes collection+version for versioned pages,
// making the intent explicit rather than relying on version being embedded in
// LangRelPath.
func GenerateFallbacks(pages []*engine.Page, languageCodes []string, defaultLang string, opts FallbackOptions) []*engine.Page {
	if len(languageCodes) <= 1 {
		return nil
	}

	if opts.SiteFallback == "omit" && len(opts.CollectionFallback) == 0 {
		return nil
	}

	exists := make(map[string]bool)
	defaultPages := make(map[string]*engine.Page)
	for _, p := range pages {
		key := fallbackKey(p)
		exists[p.Lang+":"+key] = true
		if p.Lang == defaultLang {
			defaultPages[key] = p
		}
	}

	// Iterate defaultPages in sorted key order. Go randomizes map iteration,
	// which would otherwise make the fallback slice order (and therefore
	// last-write-wins page-index population downstream) nondeterministic.
	fbKeys := make([]string, 0, len(defaultPages))
	for k := range defaultPages {
		fbKeys = append(fbKeys, k)
	}
	sort.Strings(fbKeys)

	var fallbacks []*engine.Page
	for _, lang := range languageCodes {
		if lang == defaultLang {
			continue
		}
		for _, fbKey := range fbKeys {
			defPage := defaultPages[fbKey]
			lookupKey := lang + ":" + fbKey
			if exists[lookupKey] {
				continue
			}

			effectivePolicy := opts.SiteFallback
			if defPage.Collection != nil {
				if override, ok := opts.CollectionFallback[defPage.Collection.Name]; ok && override != "" {
					effectivePolicy = override
				}
			}
			if effectivePolicy == "omit" {
				continue
			}

			fb := clonePage(defPage)
			fb.Lang = lang
			fb.LangRelPath = defPage.LangRelPath
			fb.IsFallback = true
			fb.RelPermalink = defPage.RelPermalink
			fb.Permalink = ""

			fallbacks = append(fallbacks, fb)
		}
	}

	return fallbacks
}

// fallbackKey returns the grouping key for fallback generation.
// For versioned pages: "collection:version:versionRelPath" — scopes
// fallback within a version (no cross-version fallback).
// For unversioned pages: LangRelPath (unchanged from before).
func fallbackKey(p *engine.Page) string {
	if p.Version != "" && p.Collection != nil {
		return p.Collection.Name + ":" + p.Version + ":" + p.VersionRelPath
	}
	return p.LangRelPath
}

// clonePage creates a shallow copy of a Page. Slice/pointer fields like
// Content and Headings share the underlying data (intentional — fallback
// pages display the same content as the original).
func clonePage(p *engine.Page) *engine.Page {
	cp := *p
	// Clear per-language relationship fields (rebuilt by LinkAllTranslations / nav wiring).
	// Collection and Section are intentionally kept — they are structural references
	// needed by BuildRouteData for layout, sidebar, template, and breadcrumbs.
	cp.Translations = nil
	cp.AllTranslations = nil
	cp.PrevPage = nil
	cp.NextPage = nil
	cp.NavNode = nil
	// Deep-copy Params: BeforeRender plugins (seo, socialcards) write into
	// page.Params (including nested maps like Params["seo"]) during the
	// parallel render phase. A shared map between the source page and its
	// fallback clones is a concurrent-map-write crash.
	cp.Params = deepCopyParams(p.Params)
	return &cp
}

// deepCopyParams clones a frontmatter params tree (maps, slices, scalars).
// Values come from YAML/TOML/JSON decoding, so map[string]any and []any are
// the only container types that need recursive copying.
func deepCopyParams(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyParams(val)
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = deepCopyValue(e)
		}
		return out
	default:
		return v
	}
}

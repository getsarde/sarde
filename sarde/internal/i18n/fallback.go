package i18n

import (
	"github.com/frostybee/sarde/internal/engine"
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
func GenerateFallbacks(pages []*engine.Page, languageCodes []string, defaultLang string, opts FallbackOptions) []*engine.Page {
	if len(languageCodes) <= 1 {
		return nil
	}

	if opts.SiteFallback == "omit" && len(opts.CollectionFallback) == 0 {
		return nil
	}

	// Index existing pages by (lang, langRelPath)
	exists := make(map[string]bool) // "fr:docs/getting-started.md" → true
	defaultPages := make(map[string]*engine.Page)
	for _, p := range pages {
		key := p.Lang + ":" + p.LangRelPath
		exists[key] = true
		if p.Lang == defaultLang {
			defaultPages[p.LangRelPath] = p
		}
	}

	var fallbacks []*engine.Page
	for _, lang := range languageCodes {
		if lang == defaultLang {
			continue
		}
		for relPath, defPage := range defaultPages {
			key := lang + ":" + relPath
			if exists[key] {
				continue // real translation exists
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

			// Clone the default-language page as a fallback
			fb := clonePage(defPage)
			fb.Lang = lang
			fb.LangRelPath = relPath
			fb.IsFallback = true
			fb.RelPermalink = defPage.RelPermalink
			fb.Permalink = ""

			fallbacks = append(fallbacks, fb)
		}
	}

	return fallbacks
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
	return &cp
}

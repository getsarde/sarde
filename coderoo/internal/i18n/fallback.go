package i18n

import (
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// GenerateFallbacks creates fallback pages for non-default languages where
// a page exists in the default language but has no translation.
// Returns the newly created fallback pages (caller should append to allPages).
func GenerateFallbacks(pages []*engine.Page, languageCodes []string, defaultLang string) []*engine.Page {
	if len(languageCodes) <= 1 {
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

			// Clone the default-language page as a fallback
			fb := clonePage(defPage)
			fb.Lang = lang
			fb.LangRelPath = relPath
			fb.IsFallback = true
			fb.RelPermalink = content.PrefixPermalink(defPage.RelPermalink, lang, defaultLang)
			fb.Permalink = fb.RelPermalink

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
	// Clear relationship fields that will be re-established
	cp.Translations = nil
	cp.PrevPage = nil
	cp.NextPage = nil
	cp.Collection = nil
	cp.Section = nil
	cp.NavNode = nil
	return &cp
}

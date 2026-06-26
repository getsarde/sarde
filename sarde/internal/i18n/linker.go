package i18n

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
)

// LinkTranslations groups pages by LangRelPath and populates each page's
// Translations slice with pointers to the same page in other languages.
// Pages are sorted by language weight (if available via the weights map).
func LinkTranslations(pages []*engine.Page, weights map[string]int) {
	// Group pages by their language-relative path
	groups := make(map[string][]*engine.Page)
	for _, p := range pages {
		if p.LangRelPath == "" {
			continue
		}
		groups[p.LangRelPath] = append(groups[p.LangRelPath], p)
	}

	// For each group with multiple languages, link them
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		// Sort by language weight
		sort.Slice(group, func(i, j int) bool {
			wi := weights[group[i].Lang]
			wj := weights[group[j].Lang]
			if wi != wj {
				return wi < wj
			}
			return group[i].Lang < group[j].Lang
		})

		// Set Translations on each page (all others in the group)
		for _, p := range group {
			var translations []*engine.Page
			for _, other := range group {
				if other != p {
					translations = append(translations, other)
				}
			}
			p.Translations = translations
		}
	}
}

// LinkAllTranslations groups all pages (real + fallback) by LangRelPath and
// populates each page's AllTranslations slice. Called after GenerateFallbacks
// so that fallback pages appear in AllTranslations but not in Translations.
func LinkAllTranslations(pages []*engine.Page, weights map[string]int) {
	groups := make(map[string][]*engine.Page)
	for _, p := range pages {
		if p.LangRelPath == "" {
			continue
		}
		groups[p.LangRelPath] = append(groups[p.LangRelPath], p)
	}

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		sort.Slice(group, func(i, j int) bool {
			wi := weights[group[i].Lang]
			wj := weights[group[j].Lang]
			if wi != wj {
				return wi < wj
			}
			return group[i].Lang < group[j].Lang
		})

		for _, p := range group {
			var all []*engine.Page
			for _, other := range group {
				if other != p {
					all = append(all, other)
				}
			}
			p.AllTranslations = all
		}
	}
}

package i18n

import (
	"sort"

	"github.com/coderoo-dev/coderoo/internal/engine"
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

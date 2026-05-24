package i18n

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestLinkTranslations_BasicPairing(t *testing.T) {
	enPage := &engine.Page{Lang: "en", LangRelPath: "docs/getting-started.md", Title: "Getting Started"}
	frPage := &engine.Page{Lang: "fr", LangRelPath: "docs/getting-started.md", Title: "Premiers pas"}

	pages := []*engine.Page{enPage, frPage}
	weights := map[string]int{"en": 1, "fr": 2}

	LinkTranslations(pages, weights)

	if len(enPage.Translations) != 1 || enPage.Translations[0] != frPage {
		t.Errorf("en page should have fr as translation, got %d translations", len(enPage.Translations))
	}
	if len(frPage.Translations) != 1 || frPage.Translations[0] != enPage {
		t.Errorf("fr page should have en as translation, got %d translations", len(frPage.Translations))
	}
}

func TestLinkTranslations_ThreeLanguages(t *testing.T) {
	en := &engine.Page{Lang: "en", LangRelPath: "docs/api.md"}
	fr := &engine.Page{Lang: "fr", LangRelPath: "docs/api.md"}
	ar := &engine.Page{Lang: "ar", LangRelPath: "docs/api.md"}

	pages := []*engine.Page{ar, en, fr}
	weights := map[string]int{"en": 1, "fr": 2, "ar": 3}

	LinkTranslations(pages, weights)

	// Each page should have 2 translations
	for _, p := range pages {
		if len(p.Translations) != 2 {
			t.Errorf("page %s should have 2 translations, got %d", p.Lang, len(p.Translations))
		}
	}

	// en's translations should be sorted by weight: fr, ar
	if en.Translations[0].Lang != "fr" || en.Translations[1].Lang != "ar" {
		t.Errorf("en translations order: got [%s, %s], want [fr, ar]",
			en.Translations[0].Lang, en.Translations[1].Lang)
	}
}

func TestLinkTranslations_NoMatch(t *testing.T) {
	en := &engine.Page{Lang: "en", LangRelPath: "docs/getting-started.md"}
	fr := &engine.Page{Lang: "fr", LangRelPath: "docs/introduction.md"} // different path

	pages := []*engine.Page{en, fr}
	LinkTranslations(pages, nil)

	if len(en.Translations) != 0 {
		t.Errorf("en should have 0 translations, got %d", len(en.Translations))
	}
	if len(fr.Translations) != 0 {
		t.Errorf("fr should have 0 translations, got %d", len(fr.Translations))
	}
}

func TestLinkTranslations_SinglePage(t *testing.T) {
	en := &engine.Page{Lang: "en", LangRelPath: "about.md"}

	pages := []*engine.Page{en}
	LinkTranslations(pages, nil)

	if len(en.Translations) != 0 {
		t.Errorf("single page should have 0 translations, got %d", len(en.Translations))
	}
}

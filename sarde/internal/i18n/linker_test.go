package i18n

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestLinkTranslations_BasicPairing(t *testing.T) {
	enPage := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Getting Started"}, PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/getting-started.md"}}
	frPage := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Premiers pas"}, PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "docs/getting-started.md"}}

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
	en := &engine.Page{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"}}
	fr := &engine.Page{PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "docs/api.md"}}
	ar := &engine.Page{PageI18n: engine.PageI18n{Lang: "ar", LangRelPath: "docs/api.md"}}

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
	en := &engine.Page{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/getting-started.md"}}
	fr := &engine.Page{PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "docs/introduction.md"}} // different path

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
	en := &engine.Page{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "about.md"}}

	pages := []*engine.Page{en}
	LinkTranslations(pages, nil)

	if len(en.Translations) != 0 {
		t.Errorf("single page should have 0 translations, got %d", len(en.Translations))
	}
}

func TestLinkAllTranslations_IncludesFallbacks(t *testing.T) {
	enReal := &engine.Page{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"}}
	frFallback := &engine.Page{PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "docs/api.md", IsFallback: true}}

	pages := []*engine.Page{enReal, frFallback}
	weights := map[string]int{"en": 1, "fr": 2}

	// LinkTranslations on real pages only (simulating pipeline order)
	LinkTranslations([]*engine.Page{enReal}, weights)
	if len(enReal.Translations) != 0 {
		t.Errorf("Translations should be empty for single real page; got %d", len(enReal.Translations))
	}

	// LinkAllTranslations on all pages (real + fallback)
	LinkAllTranslations(pages, weights)
	if len(enReal.AllTranslations) != 1 {
		t.Fatalf("AllTranslations should include fallback; got %d", len(enReal.AllTranslations))
	}
	if enReal.AllTranslations[0] != frFallback {
		t.Error("AllTranslations[0] should be the fr fallback page")
	}
	if len(frFallback.AllTranslations) != 1 || frFallback.AllTranslations[0] != enReal {
		t.Error("fr fallback AllTranslations should point to en real page")
	}
}

func TestLinkAllTranslations_ThreeLanguages(t *testing.T) {
	en := &engine.Page{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"}}
	fr := &engine.Page{PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "docs/api.md", IsFallback: true}}
	ar := &engine.Page{PageI18n: engine.PageI18n{Lang: "ar", LangRelPath: "docs/api.md", IsFallback: true}}

	pages := []*engine.Page{en, fr, ar}
	weights := map[string]int{"en": 1, "fr": 2, "ar": 3}

	LinkAllTranslations(pages, weights)

	for _, p := range pages {
		if len(p.AllTranslations) != 2 {
			t.Errorf("page %s: AllTranslations = %d, want 2", p.Lang, len(p.AllTranslations))
		}
	}

	// en's AllTranslations should be sorted by weight: fr, ar
	if en.AllTranslations[0].Lang != "fr" || en.AllTranslations[1].Lang != "ar" {
		t.Errorf("en AllTranslations order: got [%s, %s], want [fr, ar]",
			en.AllTranslations[0].Lang, en.AllTranslations[1].Lang)
	}
}

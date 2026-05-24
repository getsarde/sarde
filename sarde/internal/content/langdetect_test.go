package content

import "testing"

func TestClassifyLang_DefaultLanguage(t *testing.T) {
	languages := map[string]bool{"en": true, "fr": true}
	cf := &ContentFile{
		RelPath:        "docs/getting-started.md",
		CollectionName: "docs",
	}

	ClassifyLang(cf, languages, "en")

	if cf.Lang != "en" {
		t.Errorf("Lang = %q, want %q", cf.Lang, "en")
	}
	if cf.LangRelPath != "docs/getting-started.md" {
		t.Errorf("LangRelPath = %q, want %q", cf.LangRelPath, "docs/getting-started.md")
	}
	if cf.CollectionName != "docs" {
		t.Errorf("CollectionName = %q, want %q", cf.CollectionName, "docs")
	}
}

func TestClassifyLang_NonDefaultLanguage(t *testing.T) {
	languages := map[string]bool{"en": true, "fr": true}
	cf := &ContentFile{
		RelPath:        "fr/docs/getting-started.md",
		CollectionName: "fr",
	}

	ClassifyLang(cf, languages, "en")

	if cf.Lang != "fr" {
		t.Errorf("Lang = %q, want %q", cf.Lang, "fr")
	}
	if cf.LangRelPath != "docs/getting-started.md" {
		t.Errorf("LangRelPath = %q, want %q", cf.LangRelPath, "docs/getting-started.md")
	}
	if cf.CollectionName != "docs" {
		t.Errorf("CollectionName = %q, want %q", cf.CollectionName, "docs")
	}
}

func TestClassifyLang_NonDefaultLanguage_RootFile(t *testing.T) {
	languages := map[string]bool{"en": true, "fr": true}
	cf := &ContentFile{
		RelPath:        "fr/about.md",
		CollectionName: "fr",
	}

	ClassifyLang(cf, languages, "en")

	if cf.Lang != "fr" {
		t.Errorf("Lang = %q, want %q", cf.Lang, "fr")
	}
	if cf.LangRelPath != "about.md" {
		t.Errorf("LangRelPath = %q, want %q", cf.LangRelPath, "about.md")
	}
	if cf.CollectionName != "" {
		t.Errorf("CollectionName = %q, want empty", cf.CollectionName)
	}
}

func TestClassifyLang_NoLanguages(t *testing.T) {
	cf := &ContentFile{
		RelPath:        "docs/getting-started.md",
		CollectionName: "docs",
	}

	ClassifyLang(cf, nil, "en")

	if cf.Lang != "" {
		t.Errorf("Lang = %q, want empty", cf.Lang)
	}
	if cf.LangRelPath != "" {
		t.Errorf("LangRelPath = %q, want empty", cf.LangRelPath)
	}
}

func TestClassifyLang_NonLanguageDirectory(t *testing.T) {
	languages := map[string]bool{"en": true, "fr": true}
	cf := &ContentFile{
		RelPath:        "blog/welcome.md",
		CollectionName: "blog",
	}

	ClassifyLang(cf, languages, "en")

	if cf.Lang != "en" {
		t.Errorf("Lang = %q, want %q", cf.Lang, "en")
	}
	if cf.LangRelPath != "blog/welcome.md" {
		t.Errorf("LangRelPath = %q, want %q", cf.LangRelPath, "blog/welcome.md")
	}
	if cf.CollectionName != "blog" {
		t.Errorf("CollectionName = %q, want %q", cf.CollectionName, "blog")
	}
}

func TestClassifyLang_DeepNesting(t *testing.T) {
	languages := map[string]bool{"en": true, "fr": true}
	cf := &ContentFile{
		RelPath:        "fr/docs/reference/api.md",
		CollectionName: "fr",
	}

	ClassifyLang(cf, languages, "en")

	if cf.Lang != "fr" {
		t.Errorf("Lang = %q, want %q", cf.Lang, "fr")
	}
	if cf.LangRelPath != "docs/reference/api.md" {
		t.Errorf("LangRelPath = %q, want %q", cf.LangRelPath, "docs/reference/api.md")
	}
	if cf.CollectionName != "docs" {
		t.Errorf("CollectionName = %q, want %q", cf.CollectionName, "docs")
	}
}

func TestSplitFirstSegment(t *testing.T) {
	tests := []struct {
		input     string
		wantFirst string
		wantRest  string
	}{
		{"fr/docs/getting-started.md", "fr", "docs/getting-started.md"},
		{"docs/getting-started.md", "docs", "getting-started.md"},
		{"about.md", "about.md", ""},
		{"fr/about.md", "fr", "about.md"},
	}

	for _, tt := range tests {
		first, rest := splitFirstSegment(tt.input)
		if first != tt.wantFirst || rest != tt.wantRest {
			t.Errorf("splitFirstSegment(%q) = (%q, %q), want (%q, %q)",
				tt.input, first, rest, tt.wantFirst, tt.wantRest)
		}
	}
}

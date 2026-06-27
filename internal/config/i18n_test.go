package config

import "testing"

func TestI18nSettings_IsLanguageCode(t *testing.T) {
	s := &I18nSettings{
		Languages: map[string]LanguageConfig{
			"en": {Name: "English"},
			"fr": {Name: "Français"},
		},
	}

	if !s.IsLanguageCode("en") {
		t.Error("IsLanguageCode(\"en\") = false, want true")
	}
	if !s.IsLanguageCode("fr") {
		t.Error("IsLanguageCode(\"fr\") = false, want true")
	}
	if s.IsLanguageCode("docs") {
		t.Error("IsLanguageCode(\"docs\") = true, want false")
	}
	if s.IsLanguageCode("") {
		t.Error("IsLanguageCode(\"\") = true, want false")
	}
}

func TestI18nSettings_ResolveLang(t *testing.T) {
	s := &I18nSettings{DefaultLanguage: "en"}

	if got := s.ResolveLang(""); got != "en" {
		t.Errorf("ResolveLang(\"\") = %q, want \"en\"", got)
	}
	if got := s.ResolveLang("fr"); got != "fr" {
		t.Errorf("ResolveLang(\"fr\") = %q, want \"fr\"", got)
	}
}

func TestI18nSettings_Language(t *testing.T) {
	s := &I18nSettings{
		Languages: map[string]LanguageConfig{
			"en": {Name: "English", Dir: "ltr"},
		},
	}

	lc, ok := s.Language("en")
	if !ok {
		t.Fatal("Language(\"en\") returned false")
	}
	if lc.Name != "English" {
		t.Errorf("Language(\"en\").Name = %q, want \"English\"", lc.Name)
	}

	_, ok = s.Language("de")
	if ok {
		t.Error("Language(\"de\") returned true, want false")
	}
}

func TestNormalizeI18n_Defaults(t *testing.T) {
	s := &I18nSettings{
		DefaultLanguage: "en",
		Languages: map[string]LanguageConfig{
			"en": {Name: "English"},
			"fr": {Name: "Français"},
		},
	}

	if err := normalizeI18n(s); err != nil {
		t.Fatalf("normalizeI18n() error: %v", err)
	}

	if s.Strategy != "prefix-except-default" {
		t.Errorf("Strategy = %q, want \"prefix-except-default\"", s.Strategy)
	}
	if s.Fallback != "default" {
		t.Errorf("Fallback = %q, want \"default\"", s.Fallback)
	}
	if s.Languages["en"].Dir != "ltr" {
		t.Errorf("Languages[\"en\"].Dir = %q, want \"ltr\"", s.Languages["en"].Dir)
	}
	if s.Languages["fr"].Dir != "ltr" {
		t.Errorf("Languages[\"fr\"].Dir = %q, want \"ltr\"", s.Languages["fr"].Dir)
	}
}

func TestNormalizeI18n_PreservesExplicitDir(t *testing.T) {
	s := &I18nSettings{
		DefaultLanguage: "en",
		Languages: map[string]LanguageConfig{
			"en": {Name: "English"},
			"ar": {Name: "العربية", Dir: "rtl"},
		},
	}

	if err := normalizeI18n(s); err != nil {
		t.Fatalf("normalizeI18n() error: %v", err)
	}

	if s.Languages["ar"].Dir != "rtl" {
		t.Errorf("Languages[\"ar\"].Dir = %q, want \"rtl\"", s.Languages["ar"].Dir)
	}
}

func TestNormalizeI18n_NoOpWhenNotMultiLang(t *testing.T) {
	s := &I18nSettings{}

	if err := normalizeI18n(s); err != nil {
		t.Fatalf("normalizeI18n() error: %v", err)
	}

	if s.Strategy != "" {
		t.Errorf("Strategy = %q, want empty (no-op for single-lang)", s.Strategy)
	}
}

func TestNormalizeI18n_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		s    I18nSettings
		want string
	}{
		{
			name: "default lang not in languages",
			s: I18nSettings{
				DefaultLanguage: "de",
				Languages: map[string]LanguageConfig{
					"en": {Name: "English"},
				},
			},
			want: "not listed in languages",
		},
		{
			name: "unknown strategy",
			s: I18nSettings{
				DefaultLanguage: "en",
				Strategy:        "prefix-all",
				Languages: map[string]LanguageConfig{
					"en": {Name: "English"},
				},
			},
			want: "unsupported strategy",
		},
		{
			name: "unknown fallback",
			s: I18nSettings{
				DefaultLanguage: "en",
				Fallback:        "ignore",
				Languages: map[string]LanguageConfig{
					"en": {Name: "English"},
				},
			},
			want: "unsupported fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeI18n(&tt.s)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.want) {
				t.Errorf("error = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

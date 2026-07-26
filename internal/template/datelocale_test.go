package template

import (
	"testing"
	"time"
)

func TestDateTranslatorFor(t *testing.T) {
	cases := []struct {
		lang string
		want bool
	}{
		{"en", true},
		{"fr", true},
		{"pt-BR", true}, // hyphen normalized to pt_BR
		{"pt-br", true}, // case-insensitive
		{"fr-CA", true}, // base-language fallback
		{"zh-Hant", true /* exact zh_hant entry */},
		{" de ", true}, // whitespace trimmed
		{"tlh", false}, // Klingon is not in the curated set
		{"", false},
	}
	for _, c := range cases {
		got := dateTranslatorFor(c.lang)
		if (got != nil) != c.want {
			t.Errorf("dateTranslatorFor(%q) = %v, want present=%v", c.lang, got, c.want)
		}
	}
}

func TestLocalizedDateFormat(t *testing.T) {
	date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		format string
		lang   string
		want   string
	}{
		// English presets must be byte-identical to the pre-locale layouts.
		{"short", "en", "Jun 1, 2025"},
		{"long", "en", "June 1, 2025"},
		{"", "en", "Jun 1, 2025"}, // empty defaults to short
		{"iso", "en", "2025-06-01"},
		{"iso", "fr", "2025-06-01"}, // iso is locale-independent
		// Locale-aware presets.
		{"short", "fr", "1 juin 2025"},
		{"long", "fr", "1 juin 2025"},
		{"long", "de", "1. Juni 2025"},
		// Unknown language falls back to the English layouts.
		{"short", "tlh", "Jun 1, 2025"},
		{"long", "", "June 1, 2025"},
		// Custom Go layouts bypass locale data entirely.
		{"2006/01/02", "fr", "2025/06/01"},
		{"January 2006", "fr", "June 2025"},
	}
	for _, c := range cases {
		if got := localizedDateFormat(date, c.format, c.lang); got != c.want {
			t.Errorf("localizedDateFormat(%q, %q) = %q, want %q", c.format, c.lang, got, c.want)
		}
	}
}

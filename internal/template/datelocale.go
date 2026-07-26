package template

import (
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/go-playground/locales"
	"github.com/go-playground/locales/ar"
	"github.com/go-playground/locales/bg"
	"github.com/go-playground/locales/cs"
	"github.com/go-playground/locales/da"
	"github.com/go-playground/locales/de"
	"github.com/go-playground/locales/el"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fa"
	"github.com/go-playground/locales/fi"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/he"
	"github.com/go-playground/locales/hi"
	"github.com/go-playground/locales/hu"
	"github.com/go-playground/locales/id"
	"github.com/go-playground/locales/it"
	"github.com/go-playground/locales/ja"
	"github.com/go-playground/locales/ko"
	"github.com/go-playground/locales/nb"
	"github.com/go-playground/locales/nl"
	"github.com/go-playground/locales/pl"
	"github.com/go-playground/locales/pt"
	"github.com/go-playground/locales/pt_BR"
	"github.com/go-playground/locales/ro"
	"github.com/go-playground/locales/ru"
	"github.com/go-playground/locales/sk"
	"github.com/go-playground/locales/sv"
	"github.com/go-playground/locales/th"
	"github.com/go-playground/locales/tr"
	"github.com/go-playground/locales/uk"
	"github.com/go-playground/locales/vi"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/locales/zh_Hant"
)

// dateTranslators maps site language codes to CLDR date translators for the
// locale-aware date_format presets. A curated set keeps the binary small;
// adding a locale is one import plus one entry here. Languages absent from
// the map fall back to the English Go-layout path in the dateFormat template
// function, so an unlisted language degrades rather than erroring.
var dateTranslators = map[string]locales.Translator{
	"ar":      ar.New(),
	"bg":      bg.New(),
	"cs":      cs.New(),
	"da":      da.New(),
	"de":      de.New(),
	"el":      el.New(),
	"en":      en.New(),
	"es":      es.New(),
	"fa":      fa.New(),
	"fi":      fi.New(),
	"fr":      fr.New(),
	"he":      he.New(),
	"hi":      hi.New(),
	"hu":      hu.New(),
	"id":      id.New(),
	"it":      it.New(),
	"ja":      ja.New(),
	"ko":      ko.New(),
	"nb":      nb.New(),
	"nl":      nl.New(),
	"no":      nb.New(),
	"pl":      pl.New(),
	"pt":      pt.New(),
	"pt_br":   pt_BR.New(),
	"ro":      ro.New(),
	"ru":      ru.New(),
	"sk":      sk.New(),
	"sv":      sv.New(),
	"th":      th.New(),
	"tr":      tr.New(),
	"uk":      uk.New(),
	"vi":      vi.New(),
	"zh":      zh.New(),
	"zh_hant": zh_Hant.New(),
}

// dateTranslatorFor resolves a site language code ("en", "fr", "pt-BR") to a
// CLDR translator: exact match first after lowercasing and mapping "-" to
// "_", then the base language before any subtag. Returns nil when the
// language is not in the curated set.
func dateTranslatorFor(lang string) locales.Translator {
	code := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(lang)), "-", "_")
	if code == "" {
		return nil
	}
	if tr, ok := dateTranslators[code]; ok {
		return tr
	}
	if base, _, ok := strings.Cut(code, "_"); ok {
		if tr, ok := dateTranslators[base]; ok {
			return tr
		}
	}
	return nil
}

// localizedDateFormat renders t according to theme.date_format for a page
// language. The presets are locale-aware ("short" is the CLDR medium date,
// "long" the CLDR long date, "iso" always 2006-01-02); a custom Go layout, or
// a language outside the curated translator set, falls back to plain
// time.Format via NormalizeDateFormat, which is the pre-locale behavior and
// renders English month names. For "en" the CLDR output is byte-identical to
// the old layouts.
func localizedDateFormat(t time.Time, format, lang string) string {
	preset := strings.ToLower(strings.TrimSpace(format))
	if preset == "" {
		preset = "short"
	}
	switch preset {
	case "iso":
		return t.Format("2006-01-02")
	case "short":
		if tr := dateTranslatorFor(lang); tr != nil {
			return tr.FmtDateMedium(t)
		}
	case "long":
		if tr := dateTranslatorFor(lang); tr != nil {
			return tr.FmtDateLong(t)
		}
	}
	return t.Format(config.NormalizeDateFormat(format))
}

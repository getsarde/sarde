package template

import (
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/i18n"
)

// buildI18nFuncs returns the "t" and "tWithData" template functions bound to
// the given string table and current-language resolver. Both fall back to
// returning the key unchanged when the string table or resolver is unset.
func buildI18nFuncs(i18nStrings *i18n.StringTable, currentLang func() string) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"t": func(key string) string {
			if i18nStrings == nil || currentLang == nil {
				return key
			}
			return i18nStrings.Resolve(currentLang(), key)
		},
		"tWithData": func(key string, data any) string {
			if i18nStrings == nil || currentLang == nil {
				return key
			}
			return i18nStrings.Resolve(currentLang(), key, data)
		},
	}
}

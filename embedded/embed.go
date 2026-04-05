package embedded

import (
	"embed"
	"io/fs"
	"strings"
)

// DefaultsYAML contains the embedded default site configuration.
//
//go:embed defaults/site.yaml
var DefaultsYAML []byte

// LiveReloadJS contains the embedded live reload client script.
//
//go:embed livereload/livereload.js
var LiveReloadJS []byte

// i18nFS contains embedded default i18n translation strings.
//
//go:embed all:i18n
var i18nFS embed.FS

// themeFS contains all embedded theme templates and components.
//
//go:embed all:theme
var themeFS embed.FS

// I18nFS returns the embedded i18n filesystem rooted at "i18n/".
func I18nFS() fs.FS {
	sub, _ := fs.Sub(i18nFS, "i18n")
	return sub
}

// ThemeFS returns the embedded theme filesystem rooted at "theme/".
func ThemeFS() fs.FS {
	sub, _ := fs.Sub(themeFS, "theme")
	return sub
}

// ThemeCSS returns all embedded CSS files concatenated in the correct order.
// The order is: tokens, base, layout, content, components, dark.
func ThemeCSS() string {
	cssOrder := []string{
		"css/tokens.css",
		"css/base.css",
		"css/layout.css",
		"css/content.css",
		"css/components.css",
		"css/dark.css",
	}

	efs := ThemeFS()
	var sb strings.Builder
	for _, name := range cssOrder {
		data, err := fs.ReadFile(efs, name)
		if err != nil {
			continue
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}
	return sb.String()
}

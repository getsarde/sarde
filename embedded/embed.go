package embedded

import (
	"embed"
	"io/fs"
)

// DefaultsYAML contains the embedded default site configuration.
//
//go:embed defaults/site.yaml
var DefaultsYAML []byte

// themeFS contains all embedded theme templates and components.
//
//go:embed theme
var themeFS embed.FS

// ThemeFS returns the embedded theme filesystem rooted at "theme/".
func ThemeFS() fs.FS {
	sub, _ := fs.Sub(themeFS, "theme")
	return sub
}

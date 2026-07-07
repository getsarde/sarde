package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
)

// cssOrder defines the load order for theme CSS files.
// Both loadEmbeddedCSS and loadThemeCSS use this order.
var cssOrder = []string{
	"css/tokens.css",
	"css/base.css",
	"css/layout.css",
	"css/content.css",
	"css/components.css",
	"css/extensions/callouts.css",
	"css/extensions/code.css",
	"css/extensions/details.css",
	"css/extensions/steps.css",
	"css/extensions/lists.css",
	"css/extensions/cards.css",
	"css/extensions/columns.css",
	"css/extensions/inline.css",
	"css/extensions/media.css",
	"css/style.css",
	"css/blog.css",
	"css/taxonomy.css",
	"css/search.css",
	"css/homepage.css",
	"css/utilities.css",
	"css/print.css",
	"css/dark.css",
}

const cssLayerPrefix = "@layer sarde.base, sarde.reset, sarde.core, sarde.content, sarde.components, sarde.variants, sarde.utils, sarde.plugins, sarde.user;\n"

// loadThemeCSS reads and concatenates CSS files from an external theme's css/ directory.
// Returns "" if the theme has no css/ directory, allowing fallback to embedded CSS.
func loadThemeCSS(projectDir, themeName string) string {
	cssDir := filepath.Join(projectDir, consts.DirThemes, themeName, "css")
	info, err := os.Stat(cssDir)
	if err != nil || !info.IsDir() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(cssLayerPrefix)
	found := false
	for _, name := range cssOrder {
		data, err := os.ReadFile(filepath.Join(projectDir, consts.DirThemes, themeName, name))
		if err != nil {
			continue
		}
		sb.Write(data)
		sb.WriteByte('\n')
		found = true
	}
	if !found {
		return ""
	}
	return sb.String()
}

// loadEmbeddedCSS reads and concatenates all CSS files from the embedded FS.
// Each Engine instance caches the result in e.cachedCSS during Load().
func loadEmbeddedCSS(efs fs.FS) string {
	if efs == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(cssLayerPrefix)
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

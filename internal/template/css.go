package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
)

// cssOrder defines the load order for theme CSS files. A theme may ship any
// subset of these under themes/<name>/css/; the rest come from the embedded
// theme. Files outside this list are ignored.
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
	"css/slides.css",
	"css/labs.css",
	"css/taxonomy.css",
	"css/search.css",
	"css/homepage.css",
	"css/utilities.css",
	"css/print.css",
	"css/dark.css",
}

const cssLayerPrefix = "@layer sarde.base, sarde.reset, sarde.core, sarde.content, sarde.components, sarde.variants, sarde.utils, sarde.plugins, sarde.user;\n"

// assembleThemeCSS concatenates every stylesheet in cssOrder, taking each one
// from themes/<themeName>/ when it exists there and from the embedded theme
// otherwise. A theme therefore only needs to ship the stylesheets it changes.
// Returns "" when no stylesheet could be read from either source.
func assembleThemeCSS(efs fs.FS, projectDir, themeName string) string {
	var sb strings.Builder
	sb.WriteString(cssLayerPrefix)
	wrote := false
	for _, name := range cssOrder {
		if data, ok := readThemeCSS(projectDir, themeName, name); ok {
			sb.Write(data)
			sb.WriteByte('\n')
			wrote = true
			continue
		}
		if efs == nil {
			continue
		}
		data, err := fs.ReadFile(efs, name)
		if err != nil {
			continue
		}
		sb.Write(data)
		sb.WriteByte('\n')
		wrote = true
	}
	if !wrote {
		return ""
	}
	return sb.String()
}

// readThemeCSS reads one cssOrder entry from the active theme directory.
func readThemeCSS(projectDir, themeName, name string) ([]byte, bool) {
	if themeName == "" {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(projectDir, consts.DirThemes, themeName, filepath.FromSlash(name)))
	if err != nil {
		return nil, false
	}
	return data, true
}

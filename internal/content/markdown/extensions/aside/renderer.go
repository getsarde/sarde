package aside

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
)

// asideIcons maps aside types to their Lucide icon name (classic style).
var asideIcons = map[string]string{
	"note":      "book-open",
	"tip":       "sparkles",
	"info":      "info",
	"danger":    "x-circle",
	"warning":   "flame",
	"important": "flag",
	"caution":   "triangle-alert",
}

// galaxyAsideIcons holds the galaxy-style icon overrides. Types absent here
// fall through to the classic asideIcons map. gh-* types never consult this
// map: the gh- stripped lookup goes straight to asideIcons, so GitHub alerts
// keep their icons in both styles.
var galaxyAsideIcons = map[string]string{
	"note":   "info",
	"tip":    "rocket",
	"danger": "circle-alert",
}

// asideRenderer renders AsideBlock nodes to HTML.
type asideRenderer struct {
	galaxy bool
}

// NewRenderer returns a new aside renderer. style selects the icon set;
// "galaxy" swaps note/tip/danger, any other value is classic.
func NewRenderer(style string) renderer.NodeRenderer {
	return &asideRenderer{galaxy: style == "galaxy"}
}

// RegisterFuncs registers rendering functions.
func (r *asideRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAsideBlock, r.renderAside)
}

func (r *asideRenderer) renderAside(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	aside := node.(*AsideBlock)
	title := aside.GetDisplayTitle()
	var icon string
	if aside.Icon != "" {
		icon = icons.GetWithClass(aside.Icon, "sarde-aside-icon")
	}
	if icon == "" && r.galaxy {
		if name, ok := galaxyAsideIcons[aside.AsideType]; ok {
			icon = icons.GetWithClass(name, "sarde-aside-icon")
		}
	}
	if icon == "" {
		if name, ok := asideIcons[aside.AsideType]; ok {
			icon = icons.GetWithClass(name, "sarde-aside-icon")
		}
	}
	if icon == "" {
		if name, ok := asideIcons[strings.TrimPrefix(aside.AsideType, "gh-")]; ok {
			icon = icons.GetWithClass(name, "sarde-aside-icon")
		}
	}
	if icon == "" {
		icon = icons.GetWithClass("info", "sarde-aside-icon")
	}

	// GitHub-style variants need extra CSS classes to match asides-github.css
	asideClass := "sarde-aside sarde-aside-" + aside.AsideType + " not-content"
	if strings.HasPrefix(aside.AsideType, "gh-") {
		asideClass = "sarde-aside sarde-aside-github sarde-aside-" + aside.AsideType + " not-content"
	}

	if entering {
		_, _ = fmt.Fprintf(w, "<aside class=\"%s\" aria-label=\"%s\">\n", asideClass, htmlutil.EscapeHTML(title))
		_, _ = fmt.Fprintf(w, "<p class=\"sarde-aside-title\">%s %s</p>\n", icon, htmlutil.EscapeHTML(title))
		_, _ = w.WriteString("<div class=\"sarde-aside-content\">\n")
	} else {
		_, _ = w.WriteString("</div>\n</aside>\n")
	}

	return ast.WalkContinue, nil
}


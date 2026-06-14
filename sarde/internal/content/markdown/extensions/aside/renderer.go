package aside

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
)

// Icons maps aside types to Lucide SVG icons.
var Icons = map[string]string{
	"note":      `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M12 7v14"/><path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z"/></svg>`,
	"tip":       `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z"/></svg>`,
	"info":      `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>`,
	"danger":    `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/></svg>`,
	"warning":   `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"/></svg>`,
	"important": `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" x2="4" y1="22" y2="15"/></svg>`,
	"caution":   `<svg class="sarde-aside-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>`,
}

// asideRenderer renders AsideBlock nodes to HTML.
type asideRenderer struct{}

// NewRenderer returns a new aside renderer.
func NewRenderer() renderer.NodeRenderer {
	return &asideRenderer{}
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
	if icon == "" {
		icon = Icons[aside.AsideType]
	}
	if icon == "" {
		// gh- variants reuse their base type's icon
		icon = Icons[strings.TrimPrefix(aside.AsideType, "gh-")]
	}
	if icon == "" {
		icon = Icons["info"]
	}

	// GitHub-style variants need extra CSS classes to match asides-github.css
	asideClass := "sarde-aside sarde-aside-" + aside.AsideType
	if strings.HasPrefix(aside.AsideType, "gh-") {
		asideClass = "sarde-aside sarde-aside-github sarde-aside-" + aside.AsideType
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


package linkcard

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
)

type linkCardRenderer struct {
	blockedSchemes []string
}

func NewRenderer(blockedSchemes []string) renderer.NodeRenderer {
	return &linkCardRenderer{blockedSchemes: blockedSchemes}
}

func (r *linkCardRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindLinkCardBlock, r.render)
}

func (r *linkCardRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lc := node.(*LinkCardBlock)

		if !htmlutil.IsAllowedHref(lc.Href, r.blockedSchemes) {
			return ast.WalkSkipChildren, nil
		}

		title := lc.Title
		if title == "" {
			title = domainFromURL(lc.Href)
		}
		_, _ = fmt.Fprintf(w, "<a href=\"%s\" class=\"sarde-link-card\" target=\"_blank\" rel=\"noopener noreferrer\">\n",
			htmlutil.EscapeHTML(lc.Href))
		if lc.Icon != "" {
			if svg := icons.GetWithClass(lc.Icon, "sarde-link-card-icon"); svg != "" {
				_, _ = w.WriteString(svg)
				_, _ = w.WriteString("\n")
			}
		}
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-link-card-content\">\n<span class=\"sarde-link-card-title\">%s</span>\n",
			htmlutil.EscapeHTML(title))
		if lc.Description != "" {
			_, _ = fmt.Fprintf(w, "<p class=\"sarde-link-card-description\">%s</p>\n", htmlutil.EscapeHTML(lc.Description))
		}
		_, _ = w.WriteString("</div>\n<span class=\"sarde-link-card-arrow\">\u2192</span>\n</a>\n")
	}
	return ast.WalkSkipChildren, nil
}

func domainFromURL(href string) string {
	u, err := url.Parse(href)
	if err != nil || u.Host == "" {
		return href
	}
	host := u.Host
	host = strings.TrimPrefix(host, "www.")
	return host
}


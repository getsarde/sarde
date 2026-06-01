package linkbutton

import (
	"fmt"
	"strings"

	"github.com/frostybee/sarde/internal/content/markdown/extensions/linkrender"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type linkButtonRenderer struct {
	blockedSchemes []string
	linkRenderer   *linkrender.Renderer
}

func NewRenderer(blockedSchemes []string, lr *linkrender.Renderer) renderer.NodeRenderer {
	return &linkButtonRenderer{blockedSchemes: blockedSchemes, linkRenderer: lr}
}

func (r *linkButtonRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindLinkButtonBlock, r.render)
}

func (r *linkButtonRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lb := node.(*LinkButtonBlock)

		if !htmlutil.IsAllowedHref(lb.Href, r.blockedSchemes) {
			return ast.WalkSkipChildren, nil
		}

		href := lb.Href
		if r.linkRenderer != nil {
			href = r.linkRenderer.ResolveHref(href)
		}

		classes := []string{"sarde-link-button", "sarde-link-button-" + lb.Variant}
		isExternal := strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")

		attrs := fmt.Sprintf("href=\"%s\" class=\"%s\"", htmlutil.EscapeHTML(href), htmlutil.EscapeHTML(strings.Join(classes, " ")))

		if isExternal {
			attrs += ` target="_blank" rel="noopener noreferrer"`
		}

		var content strings.Builder

		if lb.Icon != "" && lb.IconPlacement == "start" {
			content.WriteString(icons.GetWithClass(lb.Icon, "sarde-link-button-icon"))
		}

		content.WriteString(htmlutil.EscapeHTML(lb.Content))

		if lb.Icon != "" && lb.IconPlacement == "end" {
			content.WriteString(icons.GetWithClass(lb.Icon, "sarde-link-button-icon"))
		}

		_, _ = fmt.Fprintf(w, "<a %s>%s</a>\n", attrs, content.String())
	}
	return ast.WalkSkipChildren, nil
}



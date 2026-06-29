package copytext

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

type copyTextRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &copyTextRenderer{} }

func (r *copyTextRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCopyText, r.render)
}

func (r *copyTextRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*CopyText)
		escaped := htmlutil.EscapeHTML(n.Content)

		_, _ = w.WriteString(`<span class="sarde-copy-text" data-copy-text="`)
		_, _ = w.WriteString(escaped)
		_, _ = w.WriteString(`"><code class="sarde-copy-text__code">`)
		_, _ = w.WriteString(escaped)
		_, _ = w.WriteString(`</code><button class="sarde-copy-text__btn" type="button" aria-label="Copy to clipboard">`)
		_, _ = w.WriteString(`<svg class="sarde-copy-text__icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`)
		_, _ = w.WriteString(`<svg class="sarde-copy-text__icon sarde-copy-text__icon--check" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`)
		_, _ = w.WriteString(`</button></span>`)
	}
	return ast.WalkSkipChildren, nil
}

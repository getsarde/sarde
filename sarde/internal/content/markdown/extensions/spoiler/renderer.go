package spoiler

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type spoilerRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &spoilerRenderer{} }

func (r *spoilerRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindSpoilerInline, r.render)
}

func (r *spoilerRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<span class="sarde-spoiler" tabindex="0" role="button" aria-label="Reveal spoiler">`)
	} else {
		_, _ = w.WriteString("</span>")
	}
	return ast.WalkContinue, nil
}

package icon

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/frostybee/sarde/internal/content/markdown/icons"
)

type iconRenderer struct{}

// NewRenderer returns the node renderer for inline ::icon[...] tokens.
func NewRenderer() renderer.NodeRenderer { return &iconRenderer{} }

func (r *iconRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindIcon, r.render)
}

func (r *iconRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*Icon)
		_, _ = w.WriteString(icons.Render(n.Name, "sarde-icon-inline", n.Attrs))
	}
	return ast.WalkSkipChildren, nil
}

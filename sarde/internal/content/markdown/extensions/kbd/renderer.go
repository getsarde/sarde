package kbd

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type kbdRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &kbdRenderer{} }

func (r *kbdRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindKbd, r.render)
}

func (r *kbdRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		k := node.(*Kbd)
		_, _ = fmt.Fprintf(w, `<kbd class="sarde-kbd">%s</kbd>`, htmlutil.EscapeHTML(k.Keys))
	}
	return ast.WalkSkipChildren, nil
}


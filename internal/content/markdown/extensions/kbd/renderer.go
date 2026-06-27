package kbd

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

type kbdRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &kbdRenderer{} }

func (r *kbdRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindKbd, r.render)
}

func (r *kbdRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		k := node.(*Kbd)
		parts := splitKeys(k.Keys)
		if len(parts) == 1 {
			_, _ = w.WriteString(`<kbd class="sarde-kbd">`)
			_, _ = w.WriteString(htmlutil.EscapeHTML(parts[0]))
			_, _ = w.WriteString(`</kbd>`)
		} else {
			_, _ = w.WriteString(`<span class="sarde-kbd-group">`)
			for i, p := range parts {
				if i > 0 {
					_, _ = w.WriteString(`<span class="sarde-kbd-separator">+</span>`)
				}
				_, _ = w.WriteString(`<kbd class="sarde-kbd">`)
				_, _ = w.WriteString(htmlutil.EscapeHTML(p))
				_, _ = w.WriteString(`</kbd>`)
			}
			_, _ = w.WriteString(`</span>`)
		}
	}
	return ast.WalkSkipChildren, nil
}

// splitKeys splits a key string on "+" into individual key names.
// If the result would be empty (e.g. the input is "+" or "++"),
// the original string is returned as a single-element slice.
func splitKeys(raw string) []string {
	var parts []string
	for _, s := range strings.Split(raw, "+") {
		s = strings.TrimSpace(s)
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return []string{raw}
	}
	return parts
}


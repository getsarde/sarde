package genericdirective

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type directiveRenderer struct {
	registry *directive.Registry
	// inner is the shared goldmark renderer, captured at Extend time.
	// Container bodies are buffer-rendered through it so nested built-in and
	// generic directives dispatch recursively.
	inner gmrenderer.Renderer
}

// NewRenderer returns a node renderer for generic directive blocks.
func NewRenderer(registry *directive.Registry, inner gmrenderer.Renderer) gmrenderer.NodeRenderer {
	return &directiveRenderer{registry: registry, inner: inner}
}

func (r *directiveRenderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(KindGenericDirective, r.render)
}

func (r *directiveRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*Node)
	def := r.registry.Lookup(n.Name)
	if def == nil {
		return ast.WalkSkipChildren, nil
	}

	var body string
	if def.Kind == directive.KindContainer {
		var buf bytes.Buffer
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if err := r.inner.Render(&buf, source, c); err != nil {
				return ast.WalkStop, fmt.Errorf("rendering directive %q body: %w", n.Name, err)
			}
		}
		body = buf.String()
	} else {
		body = htmlutil.EscapeHTML(n.RawBody)
	}

	data := directive.TemplateData{
		Name:  n.Name,
		Label: n.Label,
		Attrs: n.Attrs,
		Body:  htmltemplate.HTML(body),
	}
	var out bytes.Buffer
	if err := def.Template.Execute(&out, data); err != nil {
		return ast.WalkStop, fmt.Errorf("executing directive %q template: %w", n.Name, err)
	}
	_, _ = w.Write(out.Bytes())
	return ast.WalkSkipChildren, nil
}

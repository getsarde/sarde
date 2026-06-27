package linkcollector

import (
	"github.com/getsarde/sarde/internal/engine"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Collector collects link destinations via AST transformation.
// Mutable fields are reset before each page render.
type Collector struct {
	Links   []engine.CollectedLink
	Enabled bool
}

// NewCollector creates a Collector with collection enabled.
func NewCollector() *Collector {
	return &Collector{Enabled: true}
}

// Reset clears collected links before each page render.
func (c *Collector) Reset() {
	c.Links = c.Links[:0]
}

// Transform implements parser.ASTTransformer. It walks the AST and collects
// all link and autolink destinations without modifying the tree.
func (c *Collector) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	if !c.Enabled {
		return
	}
	source := reader.Source()
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			c.Links = append(c.Links, engine.CollectedLink{
				Href: string(v.Destination),
			})
		case *ast.AutoLink:
			c.Links = append(c.Links, engine.CollectedLink{
				Href: string(v.URL(source)),
			})
		case *ast.Image:
			c.Links = append(c.Links, engine.CollectedLink{
				Href:    string(v.Destination),
				IsImage: true,
			})
		}
		return ast.WalkContinue, nil
	})
}

package imagerender

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// attributeParser reads an attribute list written right after an image,
// such as ![Ridge](ridge.png){width=800 op=fill}, onto the image node, where
// parseImageAttrs reads the per-image processing options. Goldmark's own
// attribute syntax covers headings only. A "{" that does not follow an image
// directly, or does not parse as an attribute list, is left as text.
type attributeParser struct{}

func (attributeParser) Trigger() []byte { return []byte{'{'} }

func (attributeParser) Parse(parent ast.Node, block text.Reader, _ parser.Context) ast.Node {
	img, ok := parent.LastChild().(*ast.Image)
	if !ok {
		return nil
	}
	attrs, ok := parser.ParseAttributes(block)
	if !ok {
		return nil
	}
	for _, attr := range attrs {
		img.SetAttribute(attr.Name, attr.Value)
	}
	// The attribute list is consumed; an empty string node renders nothing.
	return ast.NewString(nil)
}

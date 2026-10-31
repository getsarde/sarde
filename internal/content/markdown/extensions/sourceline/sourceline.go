// Package sourceline marks rendered blocks with the source line they start
// on, as a data-sd-line attribute, so an editor beside the dev server's
// preview can scroll the two together (Sarde Studio's scroll sync). The
// builder enables it in dev mode only; a production build carries no such
// attribute.
package sourceline

import (
	"sort"
	"strconv"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Attribute is the attribute name on each marked block.
const Attribute = "data-sd-line"

// Marker is the AST transformer. Offset is added to every line: the
// renderer sees a page's body only, so it is the page's frontmatter line
// count, set before each render (the link renderer's convention).
type Marker struct {
	Offset int
}

// Transform implements parser.ASTTransformer.
func (m *Marker) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var starts []int
	lineOf := func(offset int) int {
		if starts == nil {
			starts = []int{0}
			for i, b := range source {
				if b == '\n' {
					starts = append(starts, i+1)
				}
			}
		}
		// The number of line starts at or before offset is its 1-based line.
		return sort.SearchInts(starts, offset+1)
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !marked(n) {
			return ast.WalkContinue, nil
		}
		if offset, ok := firstOffset(n); ok {
			n.SetAttributeString(Attribute, []byte(strconv.Itoa(lineOf(offset)+m.Offset)))
		}
		return ast.WalkContinue, nil
	})
}

// marked reports whether n is a block whose HTML renderer writes its
// attributes: paragraphs, headings, quotes, lists and tables. Code blocks
// (Kazari renders them) and raw HTML have none; a frame interpolates
// between the blocks around them.
func marked(n ast.Node) bool {
	switch n.(type) {
	case *ast.Paragraph, *ast.Heading, *ast.Blockquote, *ast.List, *ast.ListItem, *extast.Table:
		return true
	}
	return false
}

// firstOffset is the byte offset of the first source text in n or its
// descendants: container blocks hold no lines of their own.
func firstOffset(n ast.Node) (int, bool) {
	if n.Type() == ast.TypeBlock {
		if lines := n.Lines(); lines != nil && lines.Len() > 0 {
			return lines.At(0).Start, true
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() != ast.TypeBlock {
			continue
		}
		if offset, ok := firstOffset(c); ok {
			return offset, true
		}
	}
	return 0, false
}

// Extension registers a Marker.
type Extension struct {
	Marker *Marker
}

// Extend implements goldmark.Extender.
func (e *Extension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(e.Marker, 1000)))
}

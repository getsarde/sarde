package filetree

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*file-tree\s*$`)

type filetreeParser struct{}

func NewParser() parser.BlockParser       { return &filetreeParser{} }
func (p *filetreeParser) Trigger() []byte { return []byte{':'} }

func (p *filetreeParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &FileTreeBlock{}, parser.HasChildren
}

func (p *filetreeParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "file-tree")
}

func (p *filetreeParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *filetreeParser) CanInterruptParagraph() bool { return false }
func (p *filetreeParser) CanAcceptIndentedLine() bool { return false }

package columns

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	minCols     = 1
	maxCols     = 4
	defaultCols = 2
)

var (
	columnsOpeningRegex = regexp.MustCompile(`^:{3,}\s*columns(?:\((.+)\))?\s*$`)
	columnOpeningRegex  = regexp.MustCompile(`^:{3,}\s*column\s*$`)
	closingRegex        = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
	nestedOpenRegex     = regexp.MustCompile(`^:{3,}\s*\w+`)
)

type columnsParser struct{}

func NewColumnsParser() parser.BlockParser { return &columnsParser{} }
func (p *columnsParser) Trigger() []byte   { return []byte{':'} }

func (p *columnsParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := columnsOpeningRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))

	cols := defaultCols
	if matches[1] != "" {
		attrs := attrutil.Parse(matches[1])
		if v, err := strconv.Atoi(attrs["cols"]); err == nil && v >= minCols && v <= maxCols {
			cols = v
		}
	}

	return &ColumnsBlock{Cols: cols}, parser.HasChildren
}

func (p *columnsParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueContainer(node, reader, pc, "columns")
}

func (p *columnsParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}
func (p *columnsParser) CanInterruptParagraph() bool { return false }
func (p *columnsParser) CanAcceptIndentedLine() bool { return false }

type columnParser struct{}

func NewColumnParser() parser.BlockParser { return &columnParser{} }
func (p *columnParser) Trigger() []byte   { return []byte{':'} }

func (p *columnParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !columnOpeningRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	return &ColumnBlock{}, parser.HasChildren
}

func (p *columnParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueContainer(node, reader, pc, "column")
}

func (p *columnParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}
func (p *columnParser) CanInterruptParagraph() bool { return false }
func (p *columnParser) CanAcceptIndentedLine() bool { return false }

func continueContainer(node ast.Node, reader text.Reader, pc parser.Context, name string) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	depth := blockutil.GetDepth(pc, node)
	if strings.HasPrefix(trimmed, ":::") {
		if nestedOpenRegex.MatchString(trimmed) && !closingRegex.MatchString(trimmed) {
			blockutil.SetDepth(pc, node, depth+1)
			return parser.Continue | parser.HasChildren
		}
		if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
			if depth > 0 {
				blockutil.SetDepth(pc, node, depth-1)
				return parser.Continue | parser.HasChildren
			}
			if m[1] == name {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !blockutil.HasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}
	return parser.Continue | parser.HasChildren
}

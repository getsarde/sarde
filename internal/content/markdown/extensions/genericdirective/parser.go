package genericdirective

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*([\w-]+)(?:\[([^\]]*)\])?(?:\s+(.*))?$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type directiveParser struct {
	registry *directive.Registry
}

// NewParser returns a block parser dispatching on registry names. It declines
// unregistered names so unknown ::: fences fall through as plain text, and it
// registers after every built-in parser, so built-ins always win their names.
func NewParser(registry *directive.Registry) parser.BlockParser {
	return &directiveParser{registry: registry}
}

func (p *directiveParser) Trigger() []byte { return []byte{':'} }

func (p *directiveParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	name := strings.ToLower(matches[1])
	def := p.registry.Lookup(name)
	if def == nil {
		return nil, parser.NoChildren
	}

	node := &Node{
		Name:  name,
		Label: matches[2],
		Attrs: attrutil.Parse(matches[3]),
	}
	if def.Kind == directive.KindContainer {
		// Consume the fence so the framework's same-line child-open retry
		// (triggered by HasChildren) finds nothing left on this line.
		reader.Advance(len(line))
		return node, parser.HasChildren
	}
	// Leaf: leave the fence line to the framework, which advances past it
	// after Open; advancing here would skip the first body line.
	return node, parser.NoChildren
}

func (p *directiveParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	n := node.(*Node)
	def := p.registry.Lookup(n.Name)
	if def != nil && def.Kind == directive.KindLeaf {
		return p.continueLeaf(n, reader)
	}
	return p.continueContainer(n, reader, pc)
}

// continueContainer mirrors the card parser's nested-depth bookkeeping,
// parameterized on the node's name.
func (p *directiveParser) continueContainer(n *Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	depth := blockutil.GetDepth(pc, n)
	if strings.HasPrefix(trimmed, ":::") {
		if nestedOpenRegex.MatchString(trimmed) && !closingRegex.MatchString(trimmed) {
			blockutil.SetDepth(pc, n, depth+1)
			return parser.Continue | parser.HasChildren
		}
		if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
			if depth > 0 {
				blockutil.SetDepth(pc, n, depth-1)
				return parser.Continue | parser.HasChildren
			}
			if m[1] == n.Name {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !blockutil.HasInnerOpenBlocks(pc, n) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}
	return parser.Continue | parser.HasChildren
}

// continueLeaf accumulates raw body lines until the closing fence. The
// parser framework advances the reader line by line itself, so body lines
// are only peeked, never consumed here.
func (p *directiveParser) continueLeaf(n *Node, reader text.Reader) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
		if m[1] == "" || m[1] == n.Name {
			reader.AdvanceToEOL()
			return parser.Close
		}
	}

	lineContent := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
	if n.RawBody != "" {
		n.RawBody += "\n"
	}
	n.RawBody += lineContent
	return parser.Continue | parser.NoChildren
}

func (p *directiveParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}

func (p *directiveParser) CanInterruptParagraph() bool { return false }
func (p *directiveParser) CanAcceptIndentedLine() bool { return false }

package genericdirective

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/getsarde/sarde/internal/content/markdown/fence"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*([\w-]+)(?:\[([^\]]*)\])?(?:\s+(.*))?$`)

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
	line, seg := reader.PeekLine()
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
		reader.AdvanceToEOL()
		node.SourceStart = seg.Stop
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
	// PeekLine does not consume, so seg is the candidate closer line even
	// after the helper advances past it.
	_, seg := reader.PeekLine()
	state := blockutil.ContinueContainer(pc, n, reader, n.Name)
	if state == parser.Close {
		n.SourceStop = seg.Start
	}
	return state
}

// continueLeaf accumulates raw body lines until the closing fence. The
// parser framework advances the reader line by line itself, so body lines
// are only peeked, never consumed here.
func (p *directiveParser) continueLeaf(n *Node, reader text.Reader) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	if f := fence.Classify(trimmed); f.Kind == fence.Close && (f.Name == "" || fence.Closes(f.Name, n.Name)) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	lineContent := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
	if n.RawBody != "" {
		n.RawBody += "\n"
	}
	n.RawBody += lineContent
	return parser.Continue | parser.NoChildren
}

func (p *directiveParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}

func (p *directiveParser) CanInterruptParagraph() bool { return false }
func (p *directiveParser) CanAcceptIndentedLine() bool { return false }

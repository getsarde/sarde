package blockutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// stubBlock is a minimal ":::<name>" container built on ContinueContainer.
type stubBlock struct {
	ast.BaseBlock
	name string
}

var stubKind = ast.NewNodeKind("StubBlock")

func (n *stubBlock) Kind() ast.NodeKind            { return stubKind }
func (n *stubBlock) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type stubParser struct{ name string }

func (p *stubParser) Trigger() []byte { return []byte{':'} }

func (p *stubParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	if trimmed != ":::"+p.name && trimmed != "::::"+p.name {
		return nil, parser.NoChildren
	}
	// Leave the newline, as Goldmark's own block parsers do: consuming it
	// would hand the next line to the same-line child retry, and this
	// block's Continue would never see it.
	reader.AdvanceToEOL()
	return &stubBlock{name: p.name}, parser.HasChildren
}

func (p *stubParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return ContinueContainer(pc, node, reader, p.name)
}

func (p *stubParser) Close(node ast.Node, reader text.Reader, pc parser.Context) { Release(pc, node) }
func (p *stubParser) CanInterruptParagraph() bool                                { return false }
func (p *stubParser) CanAcceptIndentedLine() bool                                { return false }

type stubRenderer struct{}

func (r *stubRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(stubKind, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		n := node.(*stubBlock)
		if entering {
			w.WriteString("<" + n.name + ">")
		} else {
			w.WriteString("</" + n.name + ">")
		}
		return ast.WalkContinue, nil
	})
}

// render parses md with two stub containers, "box" and "note", and renders
// them as <box>...</box> and <note>...</note> around normal HTML.
func render(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(
		goldmark.WithParserOptions(parser.WithBlockParsers(
			util.Prioritized(&stubParser{name: "box"}, 100),
			util.Prioritized(&stubParser{name: "note"}, 101),
		)),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(&stubRenderer{}, 100),
			util.Prioritized(html.NewRenderer(), 1000),
		)),
	)
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatal(err)
	}
	// Collapse whitespace and drop it between tags so expectations stay short.
	out := strings.Join(strings.Fields(buf.String()), " ")
	out = strings.ReplaceAll(out, "> <", "><")
	return out
}

func TestContinueContainer(t *testing.T) {
	tests := []struct {
		name string
		md   string
		want string
	}{
		{"bare close", ":::box\nhi\n:::\nafter\n", "<box><p>hi</p></box><p>after</p>"},
		{"named close", ":::box\nhi\n:::/box\nafter\n", "<box><p>hi</p></box><p>after</p>"},
		{"named close is case-insensitive", ":::box\nhi\n:::/BOX\nafter\n", "<box><p>hi</p></box><p>after</p>"},
		{"longer outer fence, compat", "::::box\n:::note\nin\n:::\n::::\nafter\n", "<box><note><p>in</p></note></box><p>after</p>"},
		{"same-length nesting with bare closers", ":::box\n:::note\nin\n:::\n:::\nafter\n", "<box><note><p>in</p></note></box><p>after</p>"},
		{"mismatched name is content", ":::box\n:::/note\n:::\nafter\n", "<box><p>:::/note</p></box><p>after</p>"},
		{"same-name nesting, two named closers", ":::box\n:::box\nin\n:::/box\nout\n:::/box\nafter\n", "<box><box><p>in</p></box><p>out</p></box><p>after</p>"},
		{"recovery: outer named closer closes unclosed inner", ":::box\n:::note\nin\n:::/box\nafter\n", "<box><note><p>in</p></note></box><p>after</p>"},
		{"recovery two levels", ":::box\n:::note\n:::note\nin\n:::/box\nafter\n", "<box><note><note><p>in</p></note></note></box><p>after</p>"},
		{"named closer of an inner block pops through what it left open", ":::box\n:::note\n:::unknownthing\nx\n:::/note\nout\n:::\nafter\n", "<box><note><p>:::unknownthing x</p></note><p>out</p></box><p>after</p>"},
		{"phantom opener and bare closer cancel (the bare fence is lazy paragraph text)", ":::box\n:::nosuchthing\nx\n:::\nstill\n:::/box\nafter\n", "<box><p>:::nosuchthing x ::: still</p></box><p>after</p>"},
		{"fence lines inside a code block are content", ":::box\n```md\n:::note\n```\n:::\nafter\n", "<box><pre><code class=\"language-md\">:::note </code></pre></box><p>after</p>"},
		{"malformed closer is content", ":::box\n:::/ box\n:::\nafter\n", "<box><p>:::/ box</p></box><p>after</p>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := render(t, tt.md); got != tt.want {
				t.Errorf("\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

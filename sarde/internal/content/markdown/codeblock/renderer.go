package codeblock

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Renderer renders fenced code blocks with Chroma syntax highlighting.
type Renderer struct {
	HasCodeBlocks bool
	Languages     map[string]bool
}

// NewRenderer returns a new code block renderer.
func NewRenderer() *Renderer {
	return &Renderer{Languages: make(map[string]bool)}
}

// RegisterFuncs registers the fenced code block renderer.
func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	fcb := node.(*ast.FencedCodeBlock)

	// Extract raw code
	var codeBuilder strings.Builder
	lines := fcb.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		codeBuilder.Write(line.Value(source))
	}
	code := codeBuilder.String()

	// Parse info string
	infoStr := ""
	if fcb.Info != nil {
		infoStr = string(fcb.Info.Text(source))
	}

	// Mermaid blocks bypass Chroma and render as a plain div for client-side rendering
	if strings.TrimSpace(infoStr) == "mermaid" {
		_, _ = w.WriteString("<div class=\"sarde-mermaid\" role=\"img\" aria-label=\"Mermaid diagram\">\n")
		_, _ = w.WriteString(escapeHTML(code))
		_, _ = w.WriteString("</div>\n")
		return ast.WalkSkipChildren, nil
	}

	r.HasCodeBlocks = true

	info := ParseInfoString(infoStr)
	if info.Language != "" {
		r.Languages[info.Language] = true
	}

	// Get Chroma lexer
	lexer := getLexer(info.Language)

	// Tokenize
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Fallback: render plain
		renderPlainBlock(w, code, info)
		return ast.WalkSkipChildren, nil
	}

	// Split tokens into lines
	tokenLines := splitTokensByLine(iterator)

	// Render
	renderCodeBlock(w, tokenLines, code, info)

	return ast.WalkSkipChildren, nil
}

func getLexer(language string) chroma.Lexer {
	if language == "" {
		return lexers.Fallback
	}
	lexer := lexers.Get(language)
	if lexer == nil {
		return lexers.Fallback
	}
	return chroma.Coalesce(lexer)
}

// tokenLine represents tokens on a single line.
type tokenLine []chroma.Token

// splitTokensByLine splits a token stream into per-line groups.
func splitTokensByLine(iterator chroma.Iterator) []tokenLine {
	var lines []tokenLine
	var current tokenLine

	for _, token := range iterator.Tokens() {
		if strings.Contains(token.Value, "\n") {
			parts := strings.Split(token.Value, "\n")
			for i, part := range parts {
				if i > 0 {
					lines = append(lines, current)
					current = nil
				}
				if part != "" {
					current = append(current, chroma.Token{Type: token.Type, Value: part})
				}
			}
		} else {
			current = append(current, token)
		}
	}
	if len(current) > 0 {
		lines = append(lines, current)
	}

	return lines
}

func renderCodeBlock(w util.BufWriter, tokenLines []tokenLine, rawCode string, info CodeBlockInfo) {
	// Container classes
	classes := "sarde-code-block"
	if info.ShowLineNumbers {
		classes += " sarde-show-line-numbers"
	}
	if info.IsTerminal {
		classes += " sarde-terminal-frame"
	}

	// Data attributes for client-side plugins (code-collapsible)
	dataAttrs := fmt.Sprintf(" data-lines=\"%d\"", len(tokenLines))
	if info.Collapse {
		dataAttrs += ` data-collapse="force"`
	} else if info.NoCollapse {
		dataAttrs += ` data-collapse="none"`
	}

	_, _ = fmt.Fprintf(w, "<div class=\"%s\"%s>\n", classes, dataAttrs)

	frameClass := "sarde-frame"
	if info.Title != "" || info.IsTerminal {
		frameClass += " sarde-has-title"
	}
	_, _ = fmt.Fprintf(w, "<figure class=\"%s\">\n", frameClass)

	// Title bar
	if info.Title != "" || info.IsTerminal {
		renderTitleBar(w, info)
	}

	// Copy button + code wrapper
	_, _ = w.WriteString("<div class=\"sarde-code-block-wrapper\">\n")
	_, _ = fmt.Fprintf(w, "<button class=\"sarde-copy-btn\" aria-label=\"Copy code\" data-code=\"%s\">Copy</button>\n", escapeAttr(rawCode))

	langClass := ""
	if info.Language != "" {
		langClass = fmt.Sprintf(" language-%s", escapeHTML(info.Language))
	}
	_, _ = fmt.Fprintf(w, "<pre class=\"chroma\"><code class=\"sarde-chroma-code%s\">", langClass)

	// Render each line
	for i, line := range tokenLines {
		lineNum := i + 1
		lineClass := "sarde-code-line"
		if info.HighlightLines[lineNum] {
			lineClass += " sarde-highlight"
		}
		if info.InsertedLines[lineNum] {
			lineClass += " ins"
		}
		if info.DeletedLines[lineNum] {
			lineClass += " del"
		}

		_, _ = fmt.Fprintf(w, "<span class=\"%s\" data-line=\"%d\">", lineClass, lineNum)

		for _, token := range line {
			cls := tokenTypeClass(token.Type)
			if cls != "" {
				_, _ = fmt.Fprintf(w, "<span class=\"%s\">%s</span>", cls, escapeHTML(token.Value))
			} else {
				_, _ = w.WriteString(escapeHTML(token.Value))
			}
		}

		_, _ = w.WriteString("\n</span>")
	}

	_, _ = w.WriteString("</code></pre>\n</div>\n</figure>\n</div>\n")
}

func renderTitleBar(w util.BufWriter, info CodeBlockInfo) {
	if info.IsTerminal {
		_, _ = w.WriteString("<div class=\"sarde-code-title sarde-terminal-title\">\n")
		_, _ = w.WriteString("<span class=\"sarde-terminal-dots\" aria-hidden=\"true\"><span></span><span></span><span></span></span>\n")
		if info.Title != "" {
			_, _ = fmt.Fprintf(w, "<span class=\"sarde-code-title-text\">%s</span>\n", escapeHTML(info.Title))
		} else {
			_, _ = w.WriteString("<span class=\"sarde-code-title-text\">Terminal</span>\n")
		}
		_, _ = w.WriteString("</div>\n")
	} else {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-code-title\"><span class=\"sarde-code-title-text\">%s</span></div>\n", escapeHTML(info.Title))
	}
}

func renderPlainBlock(w util.BufWriter, code string, info CodeBlockInfo) {
	_, _ = w.WriteString("<div class=\"sarde-code-block\">\n")
	frameClass := "sarde-frame"
	if info.Title != "" {
		frameClass += " sarde-has-title"
	}
	_, _ = fmt.Fprintf(w, "<figure class=\"%s\">\n", frameClass)
	if info.Title != "" {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-code-title\"><span class=\"sarde-code-title-text\">%s</span></div>\n", escapeHTML(info.Title))
	}
	_, _ = w.WriteString("<div class=\"sarde-code-block-wrapper\">\n")
	_, _ = fmt.Fprintf(w, "<button class=\"sarde-copy-btn\" aria-label=\"Copy code\" data-code=\"%s\">Copy</button>\n", escapeAttr(code))
	_, _ = fmt.Fprintf(w, "<pre class=\"chroma\"><code>%s</code></pre>\n", escapeHTML(code))
	_, _ = w.WriteString("</div>\n</figure>\n</div>\n")
}

// tokenTypeClass maps Chroma token types to CSS class names.
func tokenTypeClass(t chroma.TokenType) string {
	// Use Chroma's standard short CSS class names
	switch {
	// Keywords
	case t == chroma.Keyword:
		return "k"
	case t == chroma.KeywordConstant:
		return "kc"
	case t == chroma.KeywordDeclaration:
		return "kd"
	case t == chroma.KeywordNamespace:
		return "kn"
	case t == chroma.KeywordPseudo:
		return "kp"
	case t == chroma.KeywordReserved:
		return "kr"
	case t == chroma.KeywordType:
		return "kt"

	// Names
	case t == chroma.Name:
		return "n"
	case t == chroma.NameAttribute:
		return "na"
	case t == chroma.NameBuiltin:
		return "nb"
	case t == chroma.NameBuiltinPseudo:
		return "bp"
	case t == chroma.NameClass:
		return "nc"
	case t == chroma.NameConstant:
		return "no"
	case t == chroma.NameDecorator:
		return "nd"
	case t == chroma.NameEntity:
		return "ni"
	case t == chroma.NameException:
		return "ne"
	case t == chroma.NameFunction:
		return "nf"
	case t == chroma.NameFunctionMagic:
		return "fm"
	case t == chroma.NameLabel:
		return "nl"
	case t == chroma.NameNamespace:
		return "nn"
	case t == chroma.NameOther:
		return "nx"
	case t == chroma.NameProperty:
		return "py"
	case t == chroma.NameTag:
		return "nt"
	case t == chroma.NameVariable:
		return "nv"
	case t == chroma.NameVariableClass:
		return "vc"
	case t == chroma.NameVariableGlobal:
		return "vg"
	case t == chroma.NameVariableInstance:
		return "vi"
	case t == chroma.NameVariableMagic:
		return "vm"

	// Literals
	case t == chroma.LiteralString:
		return "s"
	case t == chroma.LiteralStringAffix:
		return "sa"
	case t == chroma.LiteralStringBacktick:
		return "sb"
	case t == chroma.LiteralStringChar:
		return "sc"
	case t == chroma.LiteralStringDelimiter:
		return "dl"
	case t == chroma.LiteralStringDoc:
		return "sd"
	case t == chroma.LiteralStringDouble:
		return "s2"
	case t == chroma.LiteralStringEscape:
		return "se"
	case t == chroma.LiteralStringHeredoc:
		return "sh"
	case t == chroma.LiteralStringInterpol:
		return "si"
	case t == chroma.LiteralStringOther:
		return "sx"
	case t == chroma.LiteralStringRegex:
		return "sr"
	case t == chroma.LiteralStringSingle:
		return "s1"
	case t == chroma.LiteralStringSymbol:
		return "ss"
	case t == chroma.LiteralNumber:
		return "m"
	case t == chroma.LiteralNumberBin:
		return "mb"
	case t == chroma.LiteralNumberFloat:
		return "mf"
	case t == chroma.LiteralNumberHex:
		return "mh"
	case t == chroma.LiteralNumberInteger:
		return "mi"
	case t == chroma.LiteralNumberIntegerLong:
		return "il"
	case t == chroma.LiteralNumberOct:
		return "mo"

	// Operators
	case t == chroma.Operator:
		return "o"
	case t == chroma.OperatorWord:
		return "ow"

	// Punctuation
	case t == chroma.Punctuation:
		return "p"

	// Comments
	case t == chroma.Comment:
		return "c"
	case t == chroma.CommentHashbang:
		return "ch"
	case t == chroma.CommentMultiline:
		return "cm"
	case t == chroma.CommentPreproc:
		return "cp"
	case t == chroma.CommentPreprocFile:
		return "cpf"
	case t == chroma.CommentSingle:
		return "c1"
	case t == chroma.CommentSpecial:
		return "cs"

	// Generic
	case t == chroma.GenericDeleted:
		return "gd"
	case t == chroma.GenericEmph:
		return "ge"
	case t == chroma.GenericError:
		return "gr"
	case t == chroma.GenericHeading:
		return "gh"
	case t == chroma.GenericInserted:
		return "gi"
	case t == chroma.GenericOutput:
		return "go"
	case t == chroma.GenericPrompt:
		return "gp"
	case t == chroma.GenericStrong:
		return "gs"
	case t == chroma.GenericSubheading:
		return "gu"
	case t == chroma.GenericTraceback:
		return "gt"
	case t == chroma.GenericUnderline:
		return "gl"

	default:
		return ""
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	s = strings.ReplaceAll(s, "\n", "&#10;")
	return s
}

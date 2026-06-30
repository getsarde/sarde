package terminal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

var commandRegex = regexp.MustCompile(`^(\$|>|#)\s+(.+)$`)
var errorRegex = regexp.MustCompile(`(?i)✗|✘|ERROR|error:|failed`)
var warningRegex = regexp.MustCompile(`(?i)⚠|WARNING|warning:`)
var successRegex = regexp.MustCompile(`(?i)✓|✔|SUCCESS|successfully`)
var infoRegex = regexp.MustCompile(`https?://[^\s]+|/[^\s]+`)

type terminalRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &terminalRenderer{} }

func (r *terminalRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindTerminalBlock, r.render)
}

func (r *terminalRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	tb := node.(*TerminalBlock)

	content := tb.Content
	if strings.TrimSpace(content) == "" {
		content = extractAllText(node, source)
	}
	content = strings.TrimRight(content, "\n")
	lines := strings.Split(content, "\n")

	_, _ = w.WriteString("<div class=\"sarde-terminal not-content\">\n")
	_, _ = w.WriteString("<div class=\"sarde-terminal-header\">\n")
	_, _ = w.WriteString("<div class=\"sarde-terminal-buttons\">\n")
	_, _ = w.WriteString("<span class=\"sarde-terminal-button close\"></span>\n")
	_, _ = w.WriteString("<span class=\"sarde-terminal-button minimize\"></span>\n")
	_, _ = w.WriteString("<span class=\"sarde-terminal-button maximize\"></span>\n")
	_, _ = w.WriteString("</div>\n")
	_, _ = w.WriteString("<span class=\"sarde-terminal-title\">Terminal</span>\n")
	_, _ = w.WriteString("</div>\n")
	_, _ = w.WriteString("<div class=\"sarde-terminal-body\"><pre><code>")

	for i, line := range lines {
		if i > 0 {
			_, _ = w.WriteString("\n")
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		styled := styleLine(line)
		_, _ = w.WriteString(styled)
	}

	_, _ = w.WriteString("</code></pre></div>\n</div>\n")

	return ast.WalkSkipChildren, nil
}

func styleLine(line string) string {
	if m := commandRegex.FindStringSubmatch(line); m != nil {
		return fmt.Sprintf(`<span class="sarde-terminal-prompt">%s</span> <span class="sarde-terminal-command">%s</span>`,
			htmlutil.EscapeHTML(m[1]), htmlutil.EscapeHTML(m[2]))
	}

	escaped := htmlutil.EscapeHTML(line)

	if errorRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-error">%s</span>`, escaped)
	}

	if warningRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-warning">%s</span>`, escaped)
	}

	if successRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-success">%s</span>`, escaped)
	}

	if infoRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-info">%s</span>`, escaped)
	}

	return fmt.Sprintf(`<span class="sarde-terminal-output">%s</span>`, escaped)
}

func extractAllText(n ast.Node, source []byte) string {
	var sb strings.Builder
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		if node.Kind() == ast.KindText {
			t := node.(*ast.Text)
			sb.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				sb.WriteString("\n")
			}
		}
		if node.Kind() == ast.KindParagraph && node != n.FirstChild() {
			sb.WriteString("\n")
		}
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

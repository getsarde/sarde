package terminal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
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

	// Extract text content from children if Content is empty
	content := tb.Content
	if strings.TrimSpace(content) == "" {
		content = extractAllText(node, source)
	}
	content = strings.TrimRight(content, "\n")
	lines := strings.Split(content, "\n")

	_, _ = w.WriteString("<div class=\"sarde-terminal\"><pre><code>")

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

	_, _ = w.WriteString("</code></pre></div>\n")

	return ast.WalkSkipChildren, nil
}

func styleLine(line string) string {
	// Command lines (starting with $, >, #)
	if m := commandRegex.FindStringSubmatch(line); m != nil {
		return fmt.Sprintf(`<span class="sarde-terminal-prompt">%s</span> <span class="sarde-terminal-command">%s</span>`,
			htmlutil.EscapeHTML(m[1]), htmlutil.EscapeHTML(m[2]))
	}

	escaped := htmlutil.EscapeHTML(line)

	// Error lines
	if errorRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-error">%s</span>`, escaped)
	}

	// Warning lines
	if warningRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-warning">%s</span>`, escaped)
	}

	// Success lines
	if successRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-success">%s</span>`, escaped)
	}

	// Info lines (URLs, paths)
	if infoRegex.MatchString(line) {
		return fmt.Sprintf(`<span class="sarde-terminal-info">%s</span>`, escaped)
	}

	// Default output
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
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}


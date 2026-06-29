package video

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openRegex = regexp.MustCompile(`^:{3,}\s*video(?:\{([^}]+)\})?\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/video)?\s*$`)

var (
	youtubeRegex = regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/|youtube\.com/shorts/)([a-zA-Z0-9_-]+)`)
	vimeoRegex   = regexp.MustCompile(`(?:vimeo\.com/)(\d+)`)
)

var (
	reAutoplay = regexp.MustCompile(`\bautoplay\b`)
	reMuted    = regexp.MustCompile(`\bmuted\b`)
	reLoop     = regexp.MustCompile(`\bloop\b`)
)

type videoParser struct{}

func NewParser() parser.BlockParser { return &videoParser{} }
func (p *videoParser) Trigger() []byte { return []byte{':'} }

func (p *videoParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	attrs := matches[1]
	src := parseAttr(attrs, "src", "")
	if src == "" {
		return nil, parser.NoChildren
	}

	title := parseAttr(attrs, "title", "")
	ratio := parseAttr(attrs, "ratio", "")
	if ratio != "4:3" && ratio != "1:1" && ratio != "9:16" {
		ratio = ""
	}

	platform, videoID := extractVideoInfo(src)

	return &VideoBlock{
		URL:      src,
		Platform: platform,
		VideoID:  videoID,
		Title:    title,
		Ratio:    ratio,
		Autoplay: reAutoplay.MatchString(attrs),
		Muted:    reMuted.MatchString(attrs),
		Loop:     reLoop.MatchString(attrs),
	}, parser.NoChildren
}

func (p *videoParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	if closingRegex.MatchString(trimmed) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p *videoParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}
func (p *videoParser) CanInterruptParagraph() bool                                 { return false }
func (p *videoParser) CanAcceptIndentedLine() bool                                 { return false }

func extractVideoInfo(url string) (platform, videoID string) {
	if m := youtubeRegex.FindStringSubmatch(url); m != nil {
		return "youtube", m[1]
	}
	if m := vimeoRegex.FindStringSubmatch(url); m != nil {
		return "vimeo", m[1]
	}
	return "", ""
}

func parseAttr(attrs, key, fallback string) string {
	re := regexp.MustCompile(key + `\s*=\s*"([^"]*)"`)
	if m := re.FindStringSubmatch(attrs); m != nil {
		return m[1]
	}
	re2 := regexp.MustCompile(key + `\s*=\s*'([^']*)'`)
	if m := re2.FindStringSubmatch(attrs); m != nil {
		return m[1]
	}
	return fallback
}

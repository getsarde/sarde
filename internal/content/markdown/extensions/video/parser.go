package video

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var videoRegex = regexp.MustCompile(`^:::video\{src=["']([^"']+)["']\}\s*$`)

var (
	youtubeRegex = regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/)([a-zA-Z0-9_-]+)`)
	vimeoRegex   = regexp.MustCompile(`(?:vimeo\.com/)(\d+)`)
)

var closingRegex = regexp.MustCompile(`^:{3,}(?:/video)?\s*$`)

type videoParser struct{}

func NewParser() parser.BlockParser { return &videoParser{} }
func (p *videoParser) Trigger() []byte { return []byte{':'} }

func (p *videoParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := videoRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	url := matches[1]
	platform, videoID := extractVideoInfo(url)

	return &VideoBlock{
		URL:      url,
		Platform: platform,
		VideoID:  videoID,
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

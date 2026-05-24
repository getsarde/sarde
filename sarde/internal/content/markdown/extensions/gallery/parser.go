package gallery

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*gallery(?:\[([^\]]+)\])?\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type galleryParser struct{}

func NewParser() parser.BlockParser { return &galleryParser{} }
func (p *galleryParser) Trigger() []byte { return []byte{':'} }

func (p *galleryParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	label := ""
	if len(matches) > 1 {
		label = matches[1]
	}
	return &GalleryBlock{Label: label}, parser.HasChildren
}

func (p *galleryParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	depth := getDepth(pc, node)

	if strings.HasPrefix(trimmed, ":::") {
		if nestedOpenRegex.MatchString(trimmed) && !closingRegex.MatchString(trimmed) {
			setDepth(pc, node, depth+1)
			return parser.Continue | parser.HasChildren
		}
		if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
			if depth > 0 {
				setDepth(pc, node, depth-1)
				return parser.Continue | parser.HasChildren
			}
			if m[1] == "gallery" {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !hasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}

	return parser.Continue | parser.HasChildren
}

func (p *galleryParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	gallery := node.(*GalleryBlock)
	deleteDepth(pc, node)

	// Walk children to find image nodes parsed by goldmark
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n.Kind() == ast.KindImage {
			img := n.(*ast.Image)
			gallery.Images = append(gallery.Images, GalleryImage{
				Src: string(img.Destination),
				Alt: string(img.Title),
			})
			// If Title is empty, try to get alt text from children
			if gallery.Images[len(gallery.Images)-1].Alt == "" {
				alt := extractText(n, reader.Source())
				gallery.Images[len(gallery.Images)-1].Alt = alt
			}
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(node)
}

func extractText(n ast.Node, source []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindText {
			t := c.(*ast.Text)
			sb.Write(t.Segment.Value(source))
		}
	}
	return sb.String()
}

func (p *galleryParser) CanInterruptParagraph() bool { return false }
func (p *galleryParser) CanAcceptIndentedLine() bool { return false }

var contextKeyDepth = parser.NewContextKey()

func getDepth(pc parser.Context, node ast.Node) int {
	if v := pc.Get(contextKeyDepth); v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			return m[node]
		}
	}
	return 0
}

func setDepth(pc parser.Context, node ast.Node, depth int) {
	v := pc.Get(contextKeyDepth)
	var m map[ast.Node]int
	if v == nil {
		m = make(map[ast.Node]int)
	} else {
		m = v.(map[ast.Node]int)
	}
	m[node] = depth
	pc.Set(contextKeyDepth, m)
}

func hasInnerOpenBlocks(pc parser.Context, node ast.Node) bool {
	blocks := pc.OpenedBlocks()
	for i, b := range blocks {
		if b.Node == node {
			for j := i + 1; j < len(blocks); j++ {
				for _, t := range blocks[j].Parser.Trigger() {
					if t == ':' {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}

func deleteDepth(pc parser.Context, node ast.Node) {
	if v := pc.Get(contextKeyDepth); v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			delete(m, node)
		}
	}
}

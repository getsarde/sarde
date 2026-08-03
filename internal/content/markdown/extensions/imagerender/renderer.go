package imagerender

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// ImageLookupFunc looks up processed image data for a page's bundle-relative image.
// opts carries per-image overrides parsed from Goldmark attributes.
type ImageLookupFunc func(imageName string, opts asset.ImageOptions) *asset.ProcessedImage

// Renderer replaces the default Goldmark image renderer for bundle-relative images.
// External/absolute URLs pass through with standard <img> rendering.
// The Lookup field can be swapped at runtime without rebuilding the Goldmark instance.
type Renderer struct {
	html.Config
	Lookup      ImageLookupFunc
	HasImages   bool // set true when at least one image is rendered
	LazyLoading bool // site config; the first image per page is always eager
	imageIndex  int  // per-page image counter, reset via ResetPage
}

// NewRenderer creates a Goldmark image node renderer.
// If lookup is nil, all images render as standard <img> tags.
// The Lookup can be changed later via direct field assignment.
func NewRenderer(lookup ImageLookupFunc) *Renderer {
	return &Renderer{Lookup: lookup, LazyLoading: true}
}

// ResetPage zeroes the per-page image counter before each page render.
func (r *Renderer) ResetPage() {
	r.imageIndex = 0
}

// RegisterFuncs registers the renderer for ast.KindImage.
func (r *Renderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *Renderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	r.HasImages = true
	r.imageIndex++
	// The first image on a page is the likely LCP element; loading it lazily
	// delays LCP, so it is always eager regardless of the site-wide setting.
	lazy := r.LazyLoading && r.imageIndex > 1

	n := node.(*ast.Image)
	dest := string(n.Destination)
	alt := extractAlt(n, source)

	// Only intercept relative paths (bundle-relative images).
	if isRelativePath(dest) && r.Lookup != nil {
		opts := parseImageAttrs(n)
		processed := r.Lookup(dest, opts)
		if processed != nil {
			html := asset.RenderPicture(
				processed.Src,
				alt,
				processed.Width,
				processed.Height,
				processed.Variants,
				processed.LQIP,
				processed.Loading == "lazy" && lazy,
			)
			w.WriteString(html)
			return ast.WalkSkipChildren, nil
		}
	}

	// Fallback: standard <img> rendering.
	w.WriteString("<img src=\"")
	w.WriteString(escapeAttr(dest))
	w.WriteString("\" alt=\"")
	w.WriteString(escapeAttr(alt))
	w.WriteByte('"')

	if n.Title != nil {
		w.WriteString(` title="`)
		w.WriteString(escapeAttr(string(n.Title)))
		w.WriteByte('"')
	}

	// Lazy-load below-the-fold images, honoring the site-wide setting.
	if lazy {
		w.WriteString(` loading="lazy" decoding="async"`)
	} else {
		w.WriteString(` decoding="async"`)
	}
	w.WriteByte('>')

	return ast.WalkSkipChildren, nil
}

// isRelativePath returns true if the path is bundle-relative (not absolute, not external).
func isRelativePath(dest string) bool {
	if dest == "" {
		return false
	}
	// Absolute URLs.
	if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") ||
		strings.HasPrefix(dest, "//") {
		return false
	}
	// Absolute paths.
	if strings.HasPrefix(dest, "/") {
		return false
	}
	// Data URIs.
	if strings.HasPrefix(dest, "data:") {
		return false
	}
	return true
}

// extractAlt extracts the alt text from an image node's children.
func extractAlt(n *ast.Image, source []byte) string {
	var sb strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			sb.Write(text.Segment.Value(source))
		}
	}
	return sb.String()
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// parseImageAttrs extracts ImageOptions from Goldmark AST attributes.
// Attributes come from the `{key=value}` syntax after images in Markdown.
func parseImageAttrs(n ast.Node) asset.ImageOptions {
	var opts asset.ImageOptions
	if v, ok := n.AttributeString("op"); ok {
		opts.Op = asset.ResizeOp(attrString(v))
	}
	if v, ok := n.AttributeString("width"); ok {
		opts.Width = attrInt(v)
	}
	if v, ok := n.AttributeString("height"); ok {
		opts.Height = attrInt(v)
	}
	if v, ok := n.AttributeString("quality"); ok {
		opts.Quality = attrInt(v)
	}
	if v, ok := n.AttributeString("format"); ok {
		if s := attrString(v); s != "" {
			opts.Formats = []string{s}
		}
	}
	return opts
}

func attrString(v interface{}) string {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func attrInt(v interface{}) int {
	switch val := v.(type) {
	case []byte:
		n, _ := strconv.Atoi(string(val))
		return n
	case string:
		n, _ := strconv.Atoi(val)
		return n
	case int:
		return val
	case float64:
		return int(val)
	default:
		return 0
	}
}

// ForPage creates a NodeRendererFunc priority pair for use in goldmark options.
// Priority 50 overrides the default image renderer (priority 1000).
func ForPage(lookup ImageLookupFunc) util.PrioritizedValue {
	return util.Prioritized(NewRenderer(lookup), 50)
}

// String implements fmt.Stringer for debugging.
func (r *Renderer) String() string {
	return fmt.Sprintf("imagerender.Renderer{hasLookup: %v}", r.Lookup != nil)
}

package markdown

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/frostybee/kazari"
	kazarimd "github.com/frostybee/kazari/goldmark"
	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/annotation"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/aside"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/badge"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/card"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/cardgrid"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/details"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/figure"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/filetree"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/gallery"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/highlight"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/icon"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/imagecompare"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/kbd"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkbutton"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkbuttongroup"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkcard"
	extmath "github.com/getsarde/sarde/internal/content/markdown/extensions/math"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/mermaid"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/spoiler"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/steps"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/tabs"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/terminal"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/timeline"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/imagerender"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkcollector"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkrender"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/video"
	"github.com/getsarde/sarde/internal/engine"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// defaultBlockedHrefSchemes is the built-in denylist for URL schemes.
var defaultBlockedHrefSchemes = []string{"javascript:", "data:", "vbscript:"}

// Renderer handles markdown to HTML conversion with heading extraction.
// Implements engine.MarkdownRenderer.
type Renderer struct {
	md            goldmark.Markdown
	imgRend       *imagerender.Renderer       // mutable reference — swap lookup without rebuilding Goldmark
	linkCollector *linkcollector.Collector     // mutable reference — collects link destinations per page
	linkRend      *linkrender.Renderer         // mutable reference — resolves internal links per page
	kazariEngine  *kazari.Engine               // code block rendering engine; nil disables Kazari extensions
	fingerprint   string                       // cache-busting digest; computed once at construction
}

// SetImageLookup sets the lookup function for bundle-relative image processing.
// Swaps the lookup on the existing image renderer without rebuilding the Goldmark instance.
func (r *Renderer) SetImageLookup(lookup imagerender.ImageLookupFunc) {
	r.imgRend.Lookup = lookup
}

// SetLinkContext sets the current page for internal link resolution.
// Must be called before Render() for each page.
func (r *Renderer) SetLinkContext(page *engine.Page) {
	r.linkRend.SetPage(page)
}

// LinkRenderer returns the internal link renderer for external configuration
// (setting PageIndex, URLResolver, Policy).
func (r *Renderer) LinkRenderer() *linkrender.Renderer {
	return r.linkRend
}

// ImageLookupForPage creates a lookup function for a specific page's resources.
func ImageLookupForPage(page *engine.Page, processor *asset.ImageProcessor) imagerender.ImageLookupFunc {
	if processor == nil {
		return nil
	}
	loading := "lazy"
	if processor.Config.LazyLoading != nil && !*processor.Config.LazyLoading {
		loading = "eager"
	}

	return func(imageName string, opts asset.ImageOptions) *asset.ProcessedImage {
		// Strip "./" prefix if present.
		name := imageName
		if len(name) > 2 && name[:2] == "./" {
			name = name[2:]
		}

		// Find the resource in the page's bundle.
		res := asset.GetResource(page.Resources, name)
		if res == nil {
			return nil
		}

		// In dev mode, return simple image without variants.
		if processor.DevMode {
			return &asset.ProcessedImage{
				Src:     res.RelPermalink,
				Width:   res.Width,
				Height:  res.Height,
				Loading: loading,
			}
		}

		// Process the image with per-image overrides from Markdown attributes.
		srcPath := filepath.Join(filepath.Dir(page.FilePath), name)
		variants, lqip, err := processor.ProcessImage(srcPath, opts)
		if err != nil {
			// Fallback to unprocessed.
			return &asset.ProcessedImage{
				Src:     res.RelPermalink,
				Width:   res.Width,
				Height:  res.Height,
				Loading: loading,
			}
		}

		return &asset.ProcessedImage{
			Src:      res.RelPermalink,
			Width:    res.Width,
			Height:   res.Height,
			LQIP:     lqip,
			Variants: variants,
			Loading:  loading,
		}
	}
}

// execIdentity returns a cheap identifier for the running binary (modtime + size).
// Any recompilation produces a new binary, changing this value and auto-invalidating
// the page cache — no manual salt bumping needed.
func execIdentity() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	info, err := os.Stat(exe)
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}

// RendererConfig controls markdown renderer behavior.
type RendererConfig struct {
	BlockedHrefSchemes []string
	HeadingLinks       bool
	KazariEngine       *kazari.Engine
}

// NewRenderer creates a markdown renderer with all extensions and default security.
func NewRenderer() *Renderer {
	return NewRendererWithConfig(defaultBlockedHrefSchemes)
}

// NewRendererWithConfig creates a markdown renderer with the provided blocked href schemes.
// Deprecated: use NewRendererFromConfig for full configuration.
func NewRendererWithConfig(blockedHrefSchemes []string) *Renderer {
	return NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: blockedHrefSchemes,
		HeadingLinks:       true,
	})
}

// NewRendererFromConfig creates a markdown renderer from a full configuration.
func NewRendererFromConfig(cfg RendererConfig) *Renderer {
	r := &Renderer{
		imgRend:       imagerender.NewRenderer(nil),
		linkCollector: linkcollector.NewCollector(),
		linkRend:      linkrender.NewRenderer(),
		kazariEngine:  cfg.KazariEngine,
	}
	r.md, r.fingerprint = r.buildMarkdown(cfg)
	return r
}

// Fingerprint returns a content-addressed digest of the renderer's identity:
// registered extension types, config values, and Kazari CSS. The builder
// includes this in the page-cache key so the cache auto-invalidates when
// the rendering pipeline changes.
func (r *Renderer) Fingerprint() string {
	return r.fingerprint
}

func (r *Renderer) buildMarkdown(cfg RendererConfig) (goldmark.Markdown, string) {
	extensions := []goldmark.Extender{
		extension.GFM,
		extension.Footnote,
		extension.DefinitionList,
		// Block extensions
		&aside.Extension{},
		&steps.Extension{},
		&tabs.Extension{},
		&details.Extension{},
		&extmath.Extension{},
		&spoiler.Extension{},
		&filetree.Extension{},
		&timeline.Extension{},
		&card.Extension{},
		&cardgrid.Extension{},
		&linkcard.Extension{BlockedSchemes: cfg.BlockedHrefSchemes, LinkRenderer: r.linkRend},
		&linkbutton.Extension{BlockedSchemes: cfg.BlockedHrefSchemes, LinkRenderer: r.linkRend},
		&linkbuttongroup.Extension{},
		&figure.Extension{},
		&gallery.Extension{},
		&imagecompare.Extension{},
		&video.Extension{},
		&terminal.Extension{},
		&mermaid.Extension{},
		// Inline extensions
		&badge.Extension{},
		&kbd.Extension{},
		&icon.Extension{},
		&highlight.Extension{},
		&annotation.Extension{},
		// Image renderer (always present; lookup swapped at runtime via r.imgRend.Lookup)
		&imagerender.Extension{Renderer: r.imgRend},
		// Link collector (AST transformer; collects destinations for post-build validation)
		&linkcollector.Extension{Collector: r.linkCollector},
	}

	// Kazari handles fenced code blocks and :::code-group containers.
	// When no engine is configured (e.g. test helpers), fall back to Goldmark's default renderer.
	if r.kazariEngine != nil {
		extensions = append(extensions,
			kazarimd.New(r.kazariEngine),
			kazarimd.CodeGroups(r.kazariEngine),
		)
	}

	var parserOpts []parser.Option
	parserOpts = append(parserOpts, parser.WithAttribute())
	if cfg.HeadingLinks {
		parserOpts = append(parserOpts, parser.WithAutoHeadingID())
	}

	md := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOpts...),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			gmhtml.WithUnsafe(),
			gmrenderer.WithNodeRenderers(
				util.Prioritized(r.linkRend, 49),
			),
		),
	)
	return md, computeFingerprint(extensions, cfg)
}

func computeFingerprint(extensions []goldmark.Extender, cfg RendererConfig) string {
	names := make([]string, len(extensions))
	for i, ext := range extensions {
		names[i] = reflect.TypeOf(ext).String()
	}
	sort.Strings(names)

	schemes := make([]string, len(cfg.BlockedHrefSchemes))
	copy(schemes, cfg.BlockedHrefSchemes)
	sort.Strings(schemes)

	kazariCSS := ""
	if cfg.KazariEngine != nil {
		kazariCSS = cfg.KazariEngine.CSS()
	}

	raw := fmt.Sprintf("bin=%s\x00exts=%s\x00hl=%t\x00schemes=%s\x00kazari=%t\x00css=%s",
		execIdentity(),
		strings.Join(names, "|"),
		cfg.HeadingLinks,
		strings.Join(schemes, ","),
		cfg.KazariEngine != nil,
		kazariCSS,
	)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// ResetFlags zeroes content feature flags before each page render.
func (r *Renderer) ResetFlags() {
	r.imgRend.HasImages = false
	r.linkCollector.Reset()
	r.linkRend.Reset()
}

// Render converts markdown to HTML, extracts headings, and returns both.
// Implements engine.MarkdownRenderer.
func (r *Renderer) Render(markdown string) (engine.RenderResult, error) {
	r.ResetFlags()

	var buf bytes.Buffer
	if err := r.md.Convert([]byte(markdown), &buf); err != nil {
		return engine.RenderResult{}, err
	}

	html := buf.String()
	headings := extractHeadings(&html)

	var links []engine.CollectedLink
	if len(r.linkCollector.Links) > 0 {
		links = make([]engine.CollectedLink, len(r.linkCollector.Links))
		copy(links, r.linkCollector.Links)
	}

	// Detect code blocks: Kazari renders kz-* classes; without an engine, Goldmark
	// produces plain <pre> blocks. Both indicate fenced code block presence.
	hasCodeBlocks := strings.Contains(html, `class="kz-`) || strings.Contains(html, "<pre")

	return engine.RenderResult{
		HTML:          html,
		Headings:      headings,
		HasCodeBlocks: hasCodeBlocks,
		HasImages:     r.imgRend.HasImages,
		Links:         links,
	}, nil
}

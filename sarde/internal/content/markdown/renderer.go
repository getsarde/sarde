package markdown

import (
	"bytes"
	"path/filepath"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/content/markdown/codeblock"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/annotation"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/aside"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/badge"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/card"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/cardgrid"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/codegroup"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/details"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/figure"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/filetree"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/gallery"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/highlight"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/imagecompare"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/kbd"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/linkbutton"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/linkcard"
	extmath "github.com/frostybee/sarde/internal/content/markdown/extensions/math"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/spoiler"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/steps"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/tabs"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/terminal"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/timeline"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/imagerender"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/linkcollector"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/linkrender"
	"github.com/frostybee/sarde/internal/content/markdown/extensions/video"
	"github.com/frostybee/sarde/internal/engine"

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
	codeRend      *codeblock.Renderer         // mutable reference — read HasCodeBlocks flag after render
	linkCollector *linkcollector.Collector     // mutable reference — collects link destinations per page
	linkRend      *linkrender.Renderer         // mutable reference — resolves internal links per page
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

// RendererConfig controls markdown renderer behavior.
type RendererConfig struct {
	BlockedHrefSchemes []string
	HeadingLinks       bool
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
		codeRend:      codeblock.NewRenderer(),
		linkCollector: linkcollector.NewCollector(),
		linkRend:      linkrender.NewRenderer(),
	}
	r.md = r.buildMarkdown(cfg)
	return r
}

func (r *Renderer) buildMarkdown(cfg RendererConfig) goldmark.Markdown {
	extensions := []goldmark.Extender{
		extension.GFM,
		extension.Footnote,
		extension.DefinitionList,
		// Block extensions
		&aside.Extension{},
		&steps.Extension{},
		&tabs.Extension{},
		&details.Extension{},
		&codegroup.Extension{},
		&extmath.Extension{},
		&spoiler.Extension{},
		&filetree.Extension{},
		&timeline.Extension{},
		&card.Extension{},
		&cardgrid.Extension{},
		&linkcard.Extension{BlockedSchemes: cfg.BlockedHrefSchemes, LinkRenderer: r.linkRend},
		&linkbutton.Extension{BlockedSchemes: cfg.BlockedHrefSchemes, LinkRenderer: r.linkRend},
		&figure.Extension{},
		&gallery.Extension{},
		&imagecompare.Extension{},
		&video.Extension{},
		&terminal.Extension{},
		// Inline extensions
		&badge.Extension{},
		&kbd.Extension{},
		&highlight.Extension{},
		&annotation.Extension{},
		// Image renderer (always present; lookup swapped at runtime via r.imgRend.Lookup)
		&imagerender.Extension{Renderer: r.imgRend},
		// Link collector (AST transformer; collects destinations for post-build validation)
		&linkcollector.Extension{Collector: r.linkCollector},
	}

	var parserOpts []parser.Option
	parserOpts = append(parserOpts, parser.WithAttribute())
	if cfg.HeadingLinks {
		parserOpts = append(parserOpts, parser.WithAutoHeadingID())
	}

	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOpts...),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			gmhtml.WithUnsafe(),
			gmrenderer.WithNodeRenderers(
				util.Prioritized(r.codeRend, 100),
				util.Prioritized(r.linkRend, 49),
			),
		),
	)
}

// ResetFlags zeroes content feature flags before each page render.
func (r *Renderer) ResetFlags() {
	r.codeRend.HasCodeBlocks = false
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

	return engine.RenderResult{
		HTML:          html,
		Headings:      headings,
		HasCodeBlocks: r.codeRend.HasCodeBlocks,
		HasImages:     r.imgRend.HasImages,
		Links:         links,
	}, nil
}

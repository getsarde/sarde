package socialcards

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
	"github.com/getsarde/sarde/internal/workers"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font/opentype"
)

//go:embed all:assets
var assetsFS embed.FS

// New constructs the social_cards plugin from its config map.
func New(cfg map[string]any) *plugin.Plugin {
	pending := &sync.Map{}

	return &plugin.Plugin{
		Name: "social_cards",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				return beforeRender(ctx, cfg, pending)
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				return buildDone(ctx, cfg, pending)
			},
		},
	}
}

func beforeRender(ctx *plugin.BeforeRenderContext, cfg map[string]any, pending *sync.Map) error {
	page := ctx.Page
	if page == nil {
		return nil
	}

	skipIfImage := cfgutil.Bool(cfg,"skip_if_image", true)
	if skipIfImage && page.Image != "" {
		return nil
	}

	collections := cfgutil.StringSlice(cfg,"collections")
	if len(collections) > 0 {
		if page.Collection == nil || !inSlice(collections, page.Collection.Name) {
			return nil
		}
	}

	seo, _ := page.Params["seo"].(map[string]any)
	if seo != nil {
		if existing, _ := seo["og_image"].(string); existing != "" {
			return nil
		}
	}

	format := cfgutil.String(cfg,"format", "png")
	relPath := computeCardPath(page, format)

	cardURL := ctx.AbsURL("/"+relPath, page.Lang, "")

	if page.Params == nil {
		page.Params = make(map[string]any)
	}
	if seo == nil {
		seo = make(map[string]any)
		page.Params["seo"] = seo
	}
	seo["og_image"] = cardURL
	seo["twitter_image"] = cardURL
	seo["og_image_width"] = "1200"
	seo["og_image_height"] = "630"
	seo["og_image_alt"] = page.Title
	seo["twitter_image_alt"] = page.Title
	if format == "jpeg" || format == "jpg" {
		seo["og_image_type"] = "image/jpeg"
	} else {
		seo["og_image_type"] = "image/png"
	}

	pending.Store(relPath, page)
	return nil
}

func buildDone(ctx *plugin.BuildDoneContext, cfg map[string]any, pending *sync.Map) error {
	if ctx.DevMode {
		return nil
	}

	type job struct {
		relPath string
		page    *engine.Page
	}
	var jobs []job
	pending.Range(func(key, value any) bool {
		jobs = append(jobs, job{relPath: key.(string), page: value.(*engine.Page)})
		return true
	})
	if len(jobs) == 0 {
		return nil
	}

	regularFont, boldFont, fontRegHash, fontBoldHash, err := loadCardFonts(cfg, ctx.ProjectDir, ctx.Log)
	if err != nil {
		return fmt.Errorf("social_cards: loading fonts: %w", err)
	}

	bgColor := resolveBackground(cfg, ctx.Config)
	accentColor := resolveAccent(cfg, ctx.Config)
	textColor := parseHexColor(cfgutil.String(cfg,"text_color", "#ffffff"))

	var accentColor2 *color.NRGBA
	if v := cfgutil.String(cfg, "accent_color_2", ""); v != "" {
		c := parseHexColor(v)
		accentColor2 = &c
	}

	logoSize := cfgutil.Int(cfg, "logo_size", logoDrawSize)
	// Clamp to sane on-card bounds: below 16px a mark is unreadable, above
	// 256px it collides with the content block.
	if logoSize < 16 {
		logoSize = 16
	}
	if logoSize > 256 {
		logoSize = 256
	}

	// Resolve logo and background images once, before the worker pool:
	// goroutines share the decoded images read-only.
	logoMark, watermarkSrc := loadLogoImages(cfg, ctx.Config, ctx.ProjectDir, logoSize, ctx.Log)
	var watermarkImg *image.NRGBA
	if cfgutil.Bool(cfg, "watermark", false) && watermarkSrc != nil {
		watermarkImg = watermarkSrc
	}
	watermarkOpacity := cfgutil.Float(cfg, "watermark_opacity", watermarkOpacityDefault)

	bgImage := loadBgImage(cfg, ctx.ProjectDir, ctx.Log)
	bgImageOpacity := cfgutil.Float(cfg, "bg_image_opacity", 1.0)
	if bgImageOpacity <= 0 || bgImageOpacity > 1 {
		bgImageOpacity = 1.0
	}
	gradientOverride := parseGradientOverride(cfg, ctx.Log)

	siteTitle := ""
	if ctx.Site != nil {
		siteTitle = html.UnescapeString(ctx.Site.Title)
	}

	format := cfgutil.String(cfg,"format", "png")
	quality := cfgutil.Int(cfg,"quality", 90)

	// Cross-build disk cache. Asset digests are computed once here; a page
	// that hides its logo or watermark blanks the matching digest per job so
	// the toggle is part of the key.
	var cache *cardCache
	if cfgutil.Bool(cfg, "cache", true) && ctx.ProjectDir != "" {
		cache = newCardCache(ctx.ProjectDir)
	}
	baseHashes := cardAssetHashes{
		Logo:        hashImage(logoMark),
		Watermark:   hashImage(watermarkImg),
		BgImage:     hashImage(bgImage),
		FontRegular: fontRegHash,
		FontBold:    fontBoldHash,
	}
	var cacheHits atomic.Int64

	poolSize := workers.Count()
	if poolSize > len(jobs) {
		poolSize = len(jobs)
	}
	jobCh := make(chan job, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	var wg sync.WaitGroup
	errCh := make(chan error, poolSize)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			faces, err := newFaceSet(regularFont, boldFont)
			if err != nil {
				errCh <- err
				return
			}

			for j := range jobCh {
				collectionName := ""
				if j.page.Collection != nil {
					collectionName = j.page.Collection.Title
				}

				params := CardParams{
					Title:            html.UnescapeString(j.page.Title),
					Description:      cardDescription(j.page),
					SiteTitle:        siteTitle,
					CollectionName:   collectionName,
					Date:             j.page.Date,
					DateExplicit:     j.page.DateExplicit,
					BgColor:          bgColor,
					AccentColor:      accentColor,
					AccentColor2:     accentColor2,
					TextColor:        textColor,
					LogoImage:        logoMark,
					WatermarkImage:   watermarkImg,
					WatermarkOpacity: watermarkOpacity,
					BgImage:          bgImage,
					BgImageOpacity:   bgImageOpacity,
					GradientOverride: gradientOverride,
					Faces:            faces,
					BoldFont:         boldFont,
				}
				oc, _ := j.page.Params["og_card"].(*engine.OGCard)
				applyOGCard(&params, oc)

				var key string
				if cache != nil {
					hashes := baseHashes
					if params.LogoImage == nil {
						hashes.Logo = ""
					}
					if params.WatermarkImage == nil {
						hashes.Watermark = ""
					}
					key = cardKey(cardCacheVersion, params, format, quality, hashes)
					if data := cache.Get(key, format); data != nil {
						if err := ctx.WriteFile(j.relPath, data); err != nil {
							errCh <- fmt.Errorf("writing card for %s: %w", j.page.RelPermalink, err)
							return
						}
						cacheHits.Add(1)
						continue
					}
				}

				img := renderCard(params)
				data, err := encodeImage(img, format, quality)
				if err != nil {
					errCh <- fmt.Errorf("encoding card for %s: %w", j.page.RelPermalink, err)
					return
				}

				if err := ctx.WriteFile(j.relPath, data); err != nil {
					errCh <- fmt.Errorf("writing card for %s: %w", j.page.RelPermalink, err)
					return
				}
				if cache != nil {
					cache.Put(key, format, data)
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if cache != nil {
		ctx.Log(fmt.Sprintf("Generated %d social card(s) (%d from cache)", len(jobs), cacheHits.Load()))
	} else {
		ctx.Log(fmt.Sprintf("Generated %d social card(s)", len(jobs)))
	}
	return nil
}

// cardDescription returns the drawable description for a page: the explicit
// description, else the tag-stripped summary, else plain text extracted from
// the rendered HTML (covering bodies that are entirely directive blocks,
// which the raw-markdown summary extractor rightly skips), with HTML
// entities decoded once at the end so text like "Tips &amp; Tricks" draws a
// literal ampersand.
func cardDescription(page *engine.Page) string {
	d := page.Description
	if d == "" {
		d = plugin.StripHTML(string(page.Summary))
	}
	if d == "" {
		d = plugin.TrimTitlePrefix(plugin.RenderedTextFallback(page.Content, plugin.RenderedFallbackMaxChars), page.Title)
	}
	return html.UnescapeString(d)
}

// applyOGCard overlays a page's og_card frontmatter block onto the resolved
// card params: non-empty colors replace the plugin-level ones and the hide
// toggles blank the matching artwork. A nil block is a no-op.
func applyOGCard(params *CardParams, oc *engine.OGCard) {
	if oc == nil {
		return
	}
	if oc.BgColor != "" {
		params.BgColor = parseHexColor(oc.BgColor)
	}
	if oc.AccentColor != "" {
		params.AccentColor = parseHexColor(oc.AccentColor)
	}
	if oc.AccentColor2 != "" {
		c := parseHexColor(oc.AccentColor2)
		params.AccentColor2 = &c
	}
	if oc.TextColor != "" {
		params.TextColor = parseHexColor(oc.TextColor)
	}
	if oc.HideLogo {
		params.LogoImage = nil
	}
	if oc.HideWatermark {
		params.WatermarkImage = nil
	}
}

func loadFonts() (*opentype.Font, *opentype.Font, error) {
	regBytes, err := fs.ReadFile(assetsFS, "assets/fonts/Inter-Regular.ttf")
	if err != nil {
		return nil, nil, err
	}
	boldBytes, err := fs.ReadFile(assetsFS, "assets/fonts/Inter-Bold.ttf")
	if err != nil {
		return nil, nil, err
	}
	regFont, err := opentype.Parse(regBytes)
	if err != nil {
		return nil, nil, err
	}
	boldFont, err := opentype.Parse(boldBytes)
	if err != nil {
		return nil, nil, err
	}
	return regFont, boldFont, nil
}

// loadCardFonts returns the fonts for card text: the embedded Inter faces,
// with either slot replaced by a user-supplied TTF/OTF file from the
// "fonts.regular" / "fonts.bold" config keys (paths resolved relative to the
// project directory). A slot whose file is missing or unparsable logs a
// warning and keeps the embedded face, matching the logo warn-and-continue
// precedent. The returned hashes are content digests of the custom font
// bytes for cache keying; an empty hash means the embedded face, which only
// changes with the binary and is covered by cardCacheVersion.
func loadCardFonts(cfg map[string]any, projectDir string, logf func(string)) (regular, bold *opentype.Font, regularHash, boldHash string, err error) {
	regular, bold, err = loadFonts()
	if err != nil {
		return nil, nil, "", "", err
	}
	fonts, _ := cfg["fonts"].(map[string]any)
	if f, h, ok := loadFontFile(cfgutil.String(fonts, "regular", ""), projectDir, logf); ok {
		regular, regularHash = f, h
	}
	if f, h, ok := loadFontFile(cfgutil.String(fonts, "bold", ""), projectDir, logf); ok {
		bold, boldHash = f, h
	}
	return regular, bold, regularHash, boldHash, nil
}

// loadFontFile reads and parses one font file. Returns ok=false (after
// logging) on an empty path, a missing file, or a parse failure.
func loadFontFile(path, projectDir string, logf func(string)) (*opentype.Font, string, bool) {
	if path == "" {
		return nil, "", false
	}
	srcPath := filepath.FromSlash(path)
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(projectDir, srcPath)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		logf(fmt.Sprintf("social_cards: font %s: %v (cards keep the embedded Inter face)", path, err))
		return nil, "", false
	}
	f, err := opentype.Parse(data)
	if err != nil {
		logf(fmt.Sprintf("social_cards: font %s: %v (cards keep the embedded Inter face)", path, err))
		return nil, "", false
	}
	return f, hashBytes(data), true
}

// loadBgImage loads the optional card background image from the "bg_image"
// config key (a path under public/, like "logo") and pre-fits it to the card:
// "cover" crops to exactly 1200x630, "contain" letterboxes inside it. Returns
// nil when unset or on load failure (already logged).
func loadBgImage(cfg map[string]any, projectDir string, logf func(string)) *image.NRGBA {
	path := cfgutil.String(cfg, "bg_image", "")
	if path == "" {
		return nil
	}
	img := loadProjectImage(path, projectDir, logf)
	if img == nil {
		return nil
	}
	if cfgutil.String(cfg, "bg_image_fit", "cover") == "contain" {
		// imaging.Fit never upscales, but a contain background should always
		// reach the card edge on its long axis, so resize by aspect instead:
		// sources wider than the card are width-bound, others height-bound.
		b := img.Bounds()
		if b.Dx()*cardHeight >= b.Dy()*cardWidth {
			return imaging.Resize(img, cardWidth, 0, imaging.Lanczos)
		}
		return imaging.Resize(img, 0, cardHeight, imaging.Lanczos)
	}
	return imaging.Fill(img, cardWidth, cardHeight, imaging.Center, imaging.Lanczos)
}

// parseGradientOverride parses the "bg_gradient" config list into gradient
// stops for CardParams.GradientOverride. Entries beyond the second are
// ignored with a warning; an empty list keeps the automatic gradient.
func parseGradientOverride(cfg map[string]any, logf func(string)) []color.NRGBA {
	hexes := cfgutil.StringSlice(cfg, "bg_gradient")
	if len(hexes) == 0 {
		return nil
	}
	if len(hexes) > 2 {
		logf(fmt.Sprintf("social_cards: bg_gradient takes at most two colors, ignoring %d extra", len(hexes)-2))
		hexes = hexes[:2]
	}
	stops := make([]color.NRGBA, len(hexes))
	for i, h := range hexes {
		stops[i] = parseHexColor(h)
	}
	return stops
}

// loadLogoImages resolves the card logo and returns the top-left mark
// (resized to logoSize) and the watermark source (resized to
// watermarkLongEdge). Resolution order for the "logo" config key:
//
//	"none":   no logo, even when site.logo is set
//	"sarde":  the embedded Sarde mark and ribbon
//	"<path>": an image under the project's public/ directory
//	"":       fall back to site.logo (dark variant preferred, since cards
//	          have dark backgrounds by default), else no logo
//
// The "watermark_image" config key, when set, replaces the watermark source
// with its own image so a site can pair a compact corner mark with different
// watermark artwork, mirroring the built-in Sarde mark/ribbon split.
//
// Only raster formats are supported: SVG logos are skipped with a log note
// (there is no SVG rasterizer dependency). Failures never fail the build;
// they log and fall back to no logo.
func loadLogoImages(cfg map[string]any, siteCfg *config.SiteConfig, projectDir string, logoSize int, logf func(string)) (mark, watermark *image.NRGBA) {
	mark, watermark = resolveLogoImages(cfg, siteCfg, projectDir, logoSize, logf)
	if override := cfgutil.String(cfg, "watermark_image", ""); override != "" {
		if img := loadProjectImage(override, projectDir, logf); img != nil {
			watermark = resizeWatermark(img)
		}
	}
	return mark, watermark
}

// resolveLogoImages implements the "logo" key resolution described on
// loadLogoImages, without the watermark_image override.
func resolveLogoImages(cfg map[string]any, siteCfg *config.SiteConfig, projectDir string, logoSize int, logf func(string)) (mark, watermark *image.NRGBA) {
	choice := cfgutil.String(cfg, "logo", "")
	switch choice {
	case "none":
		return nil, nil
	case "sarde":
		m, err := decodeEmbeddedPNG("assets/logo/sarde-mark.png")
		if err != nil {
			logf(fmt.Sprintf("social_cards: embedded mark: %v", err))
			return nil, nil
		}
		r, err := decodeEmbeddedPNG("assets/logo/sarde-ribbon.png")
		if err != nil {
			logf(fmt.Sprintf("social_cards: embedded ribbon: %v", err))
			r = m
		}
		return resizeLogoMark(m, logoSize), resizeWatermark(r)
	case "":
		if siteCfg == nil {
			return nil, nil
		}
		path := siteCfg.Site.Logo.Dark
		if path == "" {
			path = siteCfg.Site.Logo.Light
		}
		if path == "" {
			return nil, nil
		}
		img := loadProjectImage(path, projectDir, logf)
		if img == nil {
			return nil, nil
		}
		return resizeLogoMark(img, logoSize), resizeWatermark(img)
	default:
		img := loadProjectImage(choice, projectDir, logf)
		if img == nil {
			return nil, nil
		}
		return resizeLogoMark(img, logoSize), resizeWatermark(img)
	}
}

// loadProjectImage opens a raster image under the project's public/
// directory, mirroring how site.logo paths are resolved. Returns nil (after
// logging) on SVG input, missing files, or decode errors.
func loadProjectImage(path, projectDir string, logf func(string)) *image.NRGBA {
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		logf(fmt.Sprintf("social_cards: logo %s is an SVG, which cards cannot rasterize; provide a PNG or JPEG to brand cards", path))
		return nil
	}
	srcPath := filepath.Join(projectDir, consts.DirPublic, filepath.FromSlash(strings.TrimPrefix(path, "/")))
	img, err := imaging.Open(srcPath)
	if err != nil {
		logf(fmt.Sprintf("social_cards: logo %s: %v (cards render without a logo)", path, err))
		return nil
	}
	return imaging.Clone(img)
}

func decodeEmbeddedPNG(name string) (*image.NRGBA, error) {
	data, err := fs.ReadFile(assetsFS, name)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return imaging.Clone(img), nil
}

// resizeLogoMark fits the logo into a size x size box. Zero or negative size
// keeps the default logoDrawSize box.
func resizeLogoMark(img *image.NRGBA, size int) *image.NRGBA {
	if size <= 0 {
		size = logoDrawSize
	}
	return imaging.Fit(img, size, size, imaging.Lanczos)
}

func resizeWatermark(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()
	if b.Dx() >= b.Dy() {
		return imaging.Resize(img, watermarkLongEdge, 0, imaging.Lanczos)
	}
	return imaging.Resize(img, 0, watermarkLongEdge, imaging.Lanczos)
}

func computeCardPath(page *engine.Page, format string) string {
	ext := ".png"
	if format == "jpeg" || format == "jpg" {
		ext = ".jpg"
	}
	p := strings.Trim(page.RelPermalink, "/")
	if p == "" {
		p = "_index"
	}
	if page.Lang != "" {
		return "og/" + page.Lang + "/" + p + ext
	}
	return "og/" + p + ext
}

func inSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


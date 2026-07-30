package socialcards

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

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

	regularFont, boldFont, err := loadFonts()
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

	// Resolve logo images once, before the worker pool: goroutines share the
	// decoded images read-only.
	logoMark, watermarkSrc := loadLogoImages(cfg, ctx.Config, ctx.ProjectDir, ctx.Log)
	var watermarkImg *image.NRGBA
	if cfgutil.Bool(cfg, "watermark", false) && watermarkSrc != nil {
		watermarkImg = watermarkSrc
	}
	watermarkOpacity := cfgutil.Float(cfg, "watermark_opacity", watermarkOpacityDefault)

	siteTitle := ""
	if ctx.Site != nil {
		siteTitle = ctx.Site.Title
	}

	format := cfgutil.String(cfg,"format", "png")
	quality := cfgutil.Int(cfg,"quality", 90)

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
				description := j.page.Description
				if description == "" {
					description = stripHTML(string(j.page.Summary))
				}

				collectionName := ""
				if j.page.Collection != nil {
					collectionName = j.page.Collection.Title
				}

				params := CardParams{
					Title:            j.page.Title,
					Description:      description,
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
					Faces:            faces,
					BoldFont:         boldFont,
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
	ctx.Log(fmt.Sprintf("Generated %d social card(s)", len(jobs)))
	return nil
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

// loadLogoImages resolves the card logo and returns the top-left mark
// (resized to logoDrawSize) and the watermark source (resized to
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
func loadLogoImages(cfg map[string]any, siteCfg *config.SiteConfig, projectDir string, logf func(string)) (mark, watermark *image.NRGBA) {
	mark, watermark = resolveLogoImages(cfg, siteCfg, projectDir, logf)
	if override := cfgutil.String(cfg, "watermark_image", ""); override != "" {
		if img := loadProjectImage(override, projectDir, logf); img != nil {
			watermark = resizeWatermark(img)
		}
	}
	return mark, watermark
}

// resolveLogoImages implements the "logo" key resolution described on
// loadLogoImages, without the watermark_image override.
func resolveLogoImages(cfg map[string]any, siteCfg *config.SiteConfig, projectDir string, logf func(string)) (mark, watermark *image.NRGBA) {
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
		return resizeLogoMark(m), resizeWatermark(r)
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
		return resizeLogoMark(img), resizeWatermark(img)
	default:
		img := loadProjectImage(choice, projectDir, logf)
		if img == nil {
			return nil, nil
		}
		return resizeLogoMark(img), resizeWatermark(img)
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

func resizeLogoMark(img *image.NRGBA) *image.NRGBA {
	return imaging.Fit(img, logoDrawSize, logoDrawSize, imaging.Lanczos)
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

func stripHTML(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func inSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


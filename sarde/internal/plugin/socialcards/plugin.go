package socialcards

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"
	"github.com/frostybee/sarde/internal/plugin/cfgutil"
	"github.com/frostybee/sarde/internal/workers"

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

	cardURL := ctx.AbsURL("/"+relPath, "", "")

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
					Title:          j.page.Title,
					Description:    description,
					SiteTitle:      siteTitle,
					CollectionName: collectionName,
					Date:           j.page.Date,
					BgColor:        bgColor,
					AccentColor:    accentColor,
					TextColor:      textColor,
					Faces:          faces,
					BoldFont:       boldFont,
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

func computeCardPath(page *engine.Page, format string) string {
	ext := ".png"
	if format == "jpeg" || format == "jpg" {
		ext = ".jpg"
	}
	p := strings.Trim(page.RelPermalink, "/")
	if p == "" {
		p = "_index"
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


package socialcards

import (
	"embed"
	"fmt"
	"io/fs"
	"runtime"
	"strings"
	"sync"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"

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

	skipIfImage := cfgBool(cfg, "skip_if_image", true)
	if skipIfImage && page.Image != "" {
		return nil
	}

	collections := cfgStringSlice(cfg, "collections")
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

	format := cfgString(cfg, "format", "png")
	relPath := computeCardPath(page, format)

	baseURL := ""
	if ctx.Site != nil {
		baseURL = strings.TrimRight(ctx.Site.BaseURL, "/")
	}
	cardURL := baseURL + "/" + relPath

	if page.Params == nil {
		page.Params = make(map[string]any)
	}
	if seo == nil {
		seo = make(map[string]any)
		page.Params["seo"] = seo
	}
	seo["og_image"] = cardURL
	seo["twitter_image"] = cardURL

	pending.Store(relPath, page)
	return nil
}

func buildDone(ctx *plugin.BuildDoneContext, cfg map[string]any, pending *sync.Map) error {
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
	textColor := parseHexColor(cfgString(cfg, "text_color", "#ffffff"))

	siteTitle := ""
	if ctx.Site != nil {
		siteTitle = ctx.Site.Title
	}

	format := cfgString(cfg, "format", "png")
	quality := cfgInt(cfg, "quality", 90)

	poolSize := runtime.NumCPU()
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
	for err := range errCh {
		return err
	}
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

func cfgString(cfg map[string]any, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

func cfgBool(cfg map[string]any, key string, fallback bool) bool {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

func cfgInt(cfg map[string]any, key string, fallback int) int {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return fallback
	}
}

func cfgStringSlice(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

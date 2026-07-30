package socialcards

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
)

func TestBeforeRender_SkipsWhenImageSet(t *testing.T) {
	pending := &sync.Map{}
	cfg := map[string]any{}
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Test", RelPermalink: "/blog/test/"},
		PageMeta:     engine.PageMeta{Image: "/images/existing.png"},
		Params:       make(map[string]any),
	}
	ctx := &plugin.BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
	}

	err := beforeRender(ctx, cfg, pending)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	pending.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Error("expected no pending cards when page.Image is set")
	}
}

func TestBeforeRender_SkipsWhenOgImageAlreadySet(t *testing.T) {
	pending := &sync.Map{}
	cfg := map[string]any{}
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Test", RelPermalink: "/blog/test/"},
		Params: map[string]any{
			"seo": map[string]any{
				"og_image": "https://example.com/default.png",
			},
		},
	}
	ctx := &plugin.BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
	}

	err := beforeRender(ctx, cfg, pending)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	pending.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Error("expected no pending cards when og_image is already set")
	}
}

func TestBeforeRender_InjectsWhenEmpty(t *testing.T) {
	pending := &sync.Map{}
	cfg := map[string]any{}
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Hello World", RelPermalink: "/blog/hello-world/"},
		Params: map[string]any{
			"seo": map[string]any{
				"og_image": "",
			},
		},
	}
	ctx := &plugin.BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
	}

	err := beforeRender(ctx, cfg, pending)
	if err != nil {
		t.Fatal(err)
	}

	seo := page.Params["seo"].(map[string]any)
	ogImage := seo["og_image"].(string)
	if ogImage != "https://example.com/og/blog/hello-world.png" {
		t.Errorf("expected og_image URL, got %q", ogImage)
	}

	twitterImage := seo["twitter_image"].(string)
	if twitterImage != ogImage {
		t.Errorf("twitter_image should match og_image, got %q", twitterImage)
	}

	count := 0
	pending.Range(func(_, _ any) bool { count++; return true })
	if count != 1 {
		t.Errorf("expected 1 pending card, got %d", count)
	}
}

func TestBeforeRender_CollectionFilter(t *testing.T) {
	pending := &sync.Map{}
	cfg := map[string]any{
		"collections": []any{"blog"},
	}
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Docs Page", RelPermalink: "/docs/getting-started/"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "docs"}},
		Params: map[string]any{
			"seo": map[string]any{"og_image": ""},
		},
	}
	ctx := &plugin.BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
	}

	err := beforeRender(ctx, cfg, pending)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	pending.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Error("expected no pending cards for docs when collections filter is ['blog']")
	}
}

func TestBeforeRender_NoSEOPlugin(t *testing.T) {
	pending := &sync.Map{}
	cfg := map[string]any{}
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "New Page", RelPermalink: "/about/"},
	}
	ctx := &plugin.BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
	}

	err := beforeRender(ctx, cfg, pending)
	if err != nil {
		t.Fatal(err)
	}

	if page.Params == nil {
		t.Fatal("Params should be initialized")
	}
	seo, ok := page.Params["seo"].(map[string]any)
	if !ok {
		t.Fatal("seo map should be created")
	}
	if seo["og_image"] != "https://example.com/og/about.png" {
		t.Errorf("unexpected og_image: %v", seo["og_image"])
	}
}

func TestComputeCardPath(t *testing.T) {
	tests := []struct {
		name     string
		page     *engine.Page
		format   string
		expected string
	}{
		{
			name:     "collection page",
			page:     &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/blog/hello-world/"}},
			format:   "png",
			expected: "og/blog/hello-world.png",
		},
		{
			name:     "standalone page",
			page:     &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/about/"}},
			format:   "png",
			expected: "og/about.png",
		},
		{
			name:     "home page",
			page:     &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/"}},
			format:   "png",
			expected: "og/_index.png",
		},
		{
			name:     "jpeg format",
			page:     &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/blog/post/"}},
			format:   "jpeg",
			expected: "og/blog/post.jpg",
		},
		{
			name:     "nested docs page",
			page:     &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/setup/"}},
			format:   "png",
			expected: "og/docs/guides/setup.png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCardPath(tt.page, tt.format)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input    string
		expected color.NRGBA
	}{
		{"#ffffff", color.NRGBA{255, 255, 255, 255}},
		{"#000000", color.NRGBA{0, 0, 0, 255}},
		{"#e94560", color.NRGBA{233, 69, 96, 255}},
		{"#fff", color.NRGBA{255, 255, 255, 255}},
		{"#abc", color.NRGBA{170, 187, 204, 255}},
		{"fff", color.NRGBA{255, 255, 255, 255}},
		// Invalid lengths (incl. empty) fall back to the default; the old
		// hand-rolled trimPrefix panicked on inputs shorter than the prefix.
		{"", color.NRGBA{0x1a, 0x1a, 0x2e, 255}},
		{"#f", color.NRGBA{0x1a, 0x1a, 0x2e, 255}},
	}
	for _, tt := range tests {
		got := parseHexColor(tt.input)
		if got != tt.expected {
			t.Errorf("parseHexColor(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestBuildFooter(t *testing.T) {
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		collection   string
		title        string
		date         time.Time
		dateExplicit bool
		expected     string
	}{
		{"Blog", "My Post", date, true, "Blog · Mar 15, 2024"},
		{"Blog", "My Post", date, false, "Blog"},
		{"", "My Post", date, true, "Mar 15, 2024"},
		{"", "My Post", date, false, ""},
		{"Blog", "My Post", time.Time{}, true, "Blog"},
		{"", "My Post", time.Time{}, false, ""},
		// A collection index page titles itself after the collection; the
		// footer must not repeat the title directly under itself.
		{"Blog", "Blog", date, true, "Mar 15, 2024"},
		{"Blog", "blog", date, false, ""},
	}
	for _, tt := range tests {
		got := buildFooter(tt.collection, tt.title, tt.date, tt.dateExplicit)
		if got != tt.expected {
			t.Errorf("buildFooter(%q, %q, %v, %v) = %q, want %q", tt.collection, tt.title, tt.date, tt.dateExplicit, got, tt.expected)
		}
	}
}

func TestLoadLogoImages_Empty(t *testing.T) {
	mark, watermark := loadLogoImages(nil, &config.SiteConfig{}, "", logoDrawSize, func(string) {})
	if mark != nil || watermark != nil {
		t.Error("expected no logo images when logo is unset and site.logo is empty")
	}
}

func TestLoadLogoImages_None(t *testing.T) {
	siteCfg := &config.SiteConfig{}
	siteCfg.Site.Logo.Dark = "/images/logo.png"
	cfg := map[string]any{"logo": "none"}
	mark, watermark := loadLogoImages(cfg, siteCfg, "", logoDrawSize, func(string) {})
	if mark != nil || watermark != nil {
		t.Error("logo \"none\" must suppress the logo even when site.logo is set")
	}
}

func TestLoadLogoImages_Sarde(t *testing.T) {
	cfg := map[string]any{"logo": "sarde"}
	mark, watermark := loadLogoImages(cfg, nil, "", logoDrawSize, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected embedded Sarde mark and ribbon to decode")
	}
	if got := mark.Bounds().Dx(); got != logoDrawSize {
		t.Errorf("mark width = %d, want %d", got, logoDrawSize)
	}
	wb := watermark.Bounds()
	long := wb.Dx()
	if wb.Dy() > long {
		long = wb.Dy()
	}
	if long != watermarkLongEdge {
		t.Errorf("watermark long edge = %d, want %d", long, watermarkLongEdge)
	}
}

func TestLoadLogoImages_UserFile(t *testing.T) {
	projectDir := t.TempDir()
	imgDir := filepath.Join(projectDir, "public", "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, "mark.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{"logo": "/images/mark.png"}
	mark, watermark := loadLogoImages(cfg, nil, projectDir, logoDrawSize, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected user logo to load")
	}

	// Missing file: warn and continue without a logo.
	var logged string
	cfg = map[string]any{"logo": "/images/missing.png"}
	mark, watermark = loadLogoImages(cfg, nil, projectDir, logoDrawSize, func(msg string) { logged = msg })
	if mark != nil || watermark != nil {
		t.Error("missing logo file must resolve to no logo")
	}
	if logged == "" {
		t.Error("missing logo file should log a warning")
	}
}

func TestLoadLogoImages_WatermarkImageOverride(t *testing.T) {
	projectDir := t.TempDir()
	imgDir := filepath.Join(projectDir, "public", "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG := func(name string, size int) {
		src := image.NewNRGBA(image.Rect(0, 0, size, size))
		var buf bytes.Buffer
		if err := png.Encode(&buf, src); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(imgDir, name), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePNG("mark.png", 32)
	writePNG("ribbon.png", 64)

	// watermark_image replaces only the watermark source.
	cfg := map[string]any{"logo": "/images/mark.png", "watermark_image": "/images/ribbon.png"}
	mark, watermark := loadLogoImages(cfg, nil, projectDir, logoDrawSize, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected both images to load")
	}

	// watermark_image alone still provides a watermark source with no mark.
	cfg = map[string]any{"logo": "none", "watermark_image": "/images/ribbon.png"}
	mark, watermark = loadLogoImages(cfg, nil, projectDir, logoDrawSize, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark != nil {
		t.Error("logo \"none\" must suppress the mark")
	}
	if watermark == nil {
		t.Error("an explicit watermark_image should survive logo \"none\"")
	}
}

func TestLoadLogoImages_SiteLogoFallback(t *testing.T) {
	projectDir := t.TempDir()
	imgDir := filepath.Join(projectDir, "public", "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, "dark.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Dark variant preferred.
	siteCfg := &config.SiteConfig{}
	siteCfg.Site.Logo.Light = "/images/light.png"
	siteCfg.Site.Logo.Dark = "/images/dark.png"
	mark, watermark := loadLogoImages(nil, siteCfg, projectDir, logoDrawSize, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected site.logo dark variant to load")
	}

	// SVG site logo: skipped with a log note, no logo.
	svgCfg := &config.SiteConfig{}
	svgCfg.Site.Logo.Dark = "/images/logo.svg"
	var logged string
	mark, watermark = loadLogoImages(nil, svgCfg, projectDir, logoDrawSize, func(msg string) { logged = msg })
	if mark != nil || watermark != nil {
		t.Error("SVG site logo must resolve to no logo")
	}
	if !strings.Contains(logged, "SVG") {
		t.Errorf("expected an SVG note in the log, got %q", logged)
	}
}

func TestApplyOGCard(t *testing.T) {
	logo := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	base := func() CardParams {
		return CardParams{
			BgColor:        color.NRGBA{26, 26, 46, 255},
			AccentColor:    color.NRGBA{233, 69, 96, 255},
			TextColor:      color.NRGBA{255, 255, 255, 255},
			LogoImage:      logo,
			WatermarkImage: logo,
		}
	}

	// Nil block: no-op.
	params := base()
	applyOGCard(&params, nil)
	if params.BgColor != base().BgColor || params.LogoImage == nil {
		t.Error("nil og_card block must not change params")
	}

	// Non-empty colors override, empty ones fall back to the resolved config.
	params = base()
	applyOGCard(&params, &engine.OGCard{
		BgColor:      "#0d1117",
		AccentColor2: "#58a6ff",
	})
	if params.BgColor != (color.NRGBA{0x0d, 0x11, 0x17, 255}) {
		t.Errorf("bg_color override not applied: %v", params.BgColor)
	}
	if params.AccentColor2 == nil || *params.AccentColor2 != (color.NRGBA{0x58, 0xa6, 0xff, 255}) {
		t.Errorf("accent_color_2 override not applied: %v", params.AccentColor2)
	}
	if params.AccentColor != base().AccentColor {
		t.Error("empty accent_color must keep the resolved config color")
	}
	if params.TextColor != base().TextColor {
		t.Error("empty text_color must keep the resolved config color")
	}

	// Hide toggles blank the artwork.
	params = base()
	applyOGCard(&params, &engine.OGCard{HideLogo: true, HideWatermark: true})
	if params.LogoImage != nil {
		t.Error("hide_logo must blank the logo image")
	}
	if params.WatermarkImage != nil {
		t.Error("hide_watermark must blank the watermark image")
	}
}

func TestCardDescription_DecodesEntities(t *testing.T) {
	tests := []struct {
		name     string
		page     *engine.Page
		expected string
	}{
		{
			name: "named entities in description",
			page: &engine.Page{PageMeta: engine.PageMeta{
				Description: "Tips &amp; Tricks with &quot;quotes&quot;",
			}},
			expected: `Tips & Tricks with "quotes"`,
		},
		{
			name: "numeric entities",
			page: &engine.Page{PageMeta: engine.PageMeta{
				Description: "A &#38; B &#x26; C",
			}},
			expected: "A & B & C",
		},
		{
			name: "summary fallback strips tags then decodes",
			page: &engine.Page{PageContent: engine.PageContent{
				Summary: "<p>Fast &amp; safe</p>",
			}},
			expected: "Fast & safe",
		},
		{
			name: "rendered-content fallback for directive-only bodies",
			page: &engine.Page{
				PageIdentity: engine.PageIdentity{Title: "Welcome"},
				PageContent: engine.PageContent{
					Content: `<h1>Welcome</h1><div class="card-grid"><a href="/docs/"><h3>Getting Started</h3><p>Install Sarde &amp; build</p></a></div>`,
				},
			},
			expected: "Getting Started Install Sarde & build",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cardDescription(tt.page); got != tt.expected {
				t.Errorf("cardDescription = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadCardFonts_MissingFileFallsBack(t *testing.T) {
	var logged string
	cfg := map[string]any{
		"fonts": map[string]any{"regular": "fonts/missing.ttf"},
	}
	reg, bold, regHash, boldHash, err := loadCardFonts(cfg, t.TempDir(), func(msg string) { logged = msg })
	if err != nil {
		t.Fatal(err)
	}
	if reg == nil || bold == nil {
		t.Fatal("fallback must return the embedded faces")
	}
	if logged == "" {
		t.Error("a missing font file should log a warning")
	}
	if regHash != "" || boldHash != "" {
		t.Error("embedded fallback slots must report an empty hash")
	}
}

func TestLoadCardFonts_CustomFile(t *testing.T) {
	projectDir := t.TempDir()
	data, err := fs.ReadFile(assetsFS, "assets/fonts/Inter-Bold.ttf")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "fonts", "custom.ttf"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{
		"fonts": map[string]any{"bold": "fonts/custom.ttf"},
	}
	reg, bold, regHash, boldHash, err := loadCardFonts(cfg, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if err != nil {
		t.Fatal(err)
	}
	if reg == nil || bold == nil {
		t.Fatal("both faces must resolve")
	}
	if regHash != "" {
		t.Error("untouched regular slot must report an empty hash")
	}
	if boldHash == "" {
		t.Error("custom bold slot must report a content hash")
	}
}

func TestLoadBgImage_CoverAndContain(t *testing.T) {
	projectDir := t.TempDir()
	imgDir := filepath.Join(projectDir, "public", "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, "bg.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := loadBgImage(map[string]any{}, projectDir, func(string) {}); got != nil {
		t.Error("unset bg_image must load nothing")
	}

	cover := loadBgImage(map[string]any{"bg_image": "/images/bg.png"}, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if cover == nil {
		t.Fatal("cover image should load")
	}
	if cover.Bounds().Dx() != cardWidth || cover.Bounds().Dy() != cardHeight {
		t.Errorf("cover must crop to %dx%d, got %dx%d", cardWidth, cardHeight, cover.Bounds().Dx(), cover.Bounds().Dy())
	}

	contain := loadBgImage(map[string]any{"bg_image": "/images/bg.png", "bg_image_fit": "contain"}, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if contain == nil {
		t.Fatal("contain image should load")
	}
	// A square source fits the 630px card height, letterboxed horizontally.
	if contain.Bounds().Dx() != cardHeight || contain.Bounds().Dy() != cardHeight {
		t.Errorf("contain must letterbox to %dx%d, got %dx%d", cardHeight, cardHeight, contain.Bounds().Dx(), contain.Bounds().Dy())
	}

	// Missing file: warn and continue without a background.
	var logged string
	if got := loadBgImage(map[string]any{"bg_image": "/images/missing.png"}, projectDir, func(msg string) { logged = msg }); got != nil {
		t.Error("missing bg_image must resolve to nil")
	}
	if logged == "" {
		t.Error("missing bg_image should log a warning")
	}
}

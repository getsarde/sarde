package socialcards

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>hello</p>", "hello"},
		{"no tags", "no tags"},
		{"<strong>bold</strong> and plain", "bold and plain"},
		{"", ""},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.expected {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildFooter(t *testing.T) {
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		collection   string
		date         time.Time
		dateExplicit bool
		expected     string
	}{
		{"Blog", date, true, "Blog · Mar 15, 2024"},
		{"Blog", date, false, "Blog"},
		{"", date, true, "Mar 15, 2024"},
		{"", date, false, ""},
		{"Blog", time.Time{}, true, "Blog"},
		{"", time.Time{}, false, ""},
	}
	for _, tt := range tests {
		got := buildFooter(tt.collection, tt.date, tt.dateExplicit)
		if got != tt.expected {
			t.Errorf("buildFooter(%q, %v, %v) = %q, want %q", tt.collection, tt.date, tt.dateExplicit, got, tt.expected)
		}
	}
}

func TestLoadLogoImages_Empty(t *testing.T) {
	mark, watermark := loadLogoImages(nil, &config.SiteConfig{}, "", func(string) {})
	if mark != nil || watermark != nil {
		t.Error("expected no logo images when logo is unset and site.logo is empty")
	}
}

func TestLoadLogoImages_None(t *testing.T) {
	siteCfg := &config.SiteConfig{}
	siteCfg.Site.Logo.Dark = "/images/logo.png"
	cfg := map[string]any{"logo": "none"}
	mark, watermark := loadLogoImages(cfg, siteCfg, "", func(string) {})
	if mark != nil || watermark != nil {
		t.Error("logo \"none\" must suppress the logo even when site.logo is set")
	}
}

func TestLoadLogoImages_Sarde(t *testing.T) {
	cfg := map[string]any{"logo": "sarde"}
	mark, watermark := loadLogoImages(cfg, nil, "", func(msg string) { t.Errorf("unexpected log: %s", msg) })
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
	mark, watermark := loadLogoImages(cfg, nil, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected user logo to load")
	}

	// Missing file: warn and continue without a logo.
	var logged string
	cfg = map[string]any{"logo": "/images/missing.png"}
	mark, watermark = loadLogoImages(cfg, nil, projectDir, func(msg string) { logged = msg })
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
	mark, watermark := loadLogoImages(cfg, nil, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected both images to load")
	}

	// watermark_image alone still provides a watermark source with no mark.
	cfg = map[string]any{"logo": "none", "watermark_image": "/images/ribbon.png"}
	mark, watermark = loadLogoImages(cfg, nil, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
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
	mark, watermark := loadLogoImages(nil, siteCfg, projectDir, func(msg string) { t.Errorf("unexpected log: %s", msg) })
	if mark == nil || watermark == nil {
		t.Fatal("expected site.logo dark variant to load")
	}

	// SVG site logo: skipped with a log note, no logo.
	svgCfg := &config.SiteConfig{}
	svgCfg.Site.Logo.Dark = "/images/logo.svg"
	var logged string
	mark, watermark = loadLogoImages(nil, svgCfg, projectDir, func(msg string) { logged = msg })
	if mark != nil || watermark != nil {
		t.Error("SVG site logo must resolve to no logo")
	}
	if !strings.Contains(logged, "SVG") {
		t.Errorf("expected an SVG note in the log, got %q", logged)
	}
}

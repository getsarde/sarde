package template

import (
	htmltemplate "html/template"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func nilFuncMap() htmltemplate.FuncMap {
	e := &Engine{}
	return e.buildFuncMap(nil, nil)
}

func testSite() *engine.SiteContext {
	return &engine.SiteContext{
		Title:    "Test Site",
		BaseURL:  "https://example.com",
		Language: "en",
		Collections: map[string]*engine.Collection{
			"blog": {
				Name:  "blog",
				Title: "Blog",
				Pages: []*engine.Page{
					{PageIdentity: engine.PageIdentity{Title: "Post A", Slug: "post-a", RelPermalink: "/blog/post-a/"}},
					{PageIdentity: engine.PageIdentity{Title: "Post B", Slug: "post-b", RelPermalink: "/blog/post-b/"}},
					{PageIdentity: engine.PageIdentity{Title: "Post C", Slug: "post-c", RelPermalink: "/blog/post-c/"}},
				},
			},
		},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Post A", Slug: "post-a", Permalink: "https://example.com/blog/post-a/", RelPermalink: "/blog/post-a/"}},
		},
	}
}

func testFuncMapBuild() htmltemplate.FuncMap {
	e := &Engine{
		resolver: &engine.ThemeResolver{},
		site:     testSite(),
	}
	e.currentLang = "en"
	return e.buildFuncMap(nil, nil)
}

// ── String tests ──

func TestFnIcon(t *testing.T) {
	// name only → decorative, no class
	got := string(fnIcon("rocket"))
	if !strings.Contains(got, "<svg") || !strings.Contains(got, `aria-hidden="true"`) {
		t.Errorf("fnIcon(rocket) = %q", got)
	}
	// name + class
	got = string(fnIcon("rocket", "sarde-icon"))
	if !strings.Contains(got, `class="sarde-icon"`) {
		t.Errorf("fnIcon with class = %q", got)
	}
	// name + class + key/val pair
	got = string(fnIcon("arrow-up", "sarde-icon", "rotate", "90"))
	if !strings.Contains(got, `style="transform: rotate(90deg)"`) {
		t.Errorf("fnIcon with rotate = %q", got)
	}
	// trailing odd arg is ignored (no panic, still renders)
	got = string(fnIcon("rocket", "sarde-icon", "rotate"))
	if !strings.Contains(got, `class="sarde-icon"`) {
		t.Errorf("fnIcon odd arg = %q", got)
	}
}

func TestFnTitle(t *testing.T) {
	if got := fnTitle("hello world"); got != "Hello World" {
		t.Errorf("got %q", got)
	}
}

func TestFnTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"Hello World", 5, "He..."},
		{"Hi", 5, "Hi"},
		{"Hi", 2, "Hi"},
		{"Hello", 3, "Hel"},
		{"Hello", 0, ""},
		{"Hello", -5, ""}, // negative counts must clamp, not panic
		{"", -1, ""},
	}
	for _, tt := range tests {
		got := fnTruncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestFnPlainify(t *testing.T) {
	got := fnPlainify("<p>Hello <b>World</b></p>")
	if got != "Hello World" {
		t.Errorf("got %q", got)
	}
}

func TestFnSafeHTML(t *testing.T) {
	got := fnSafeHTML("<b>bold</b>")
	if got != htmltemplate.HTML("<b>bold</b>") {
		t.Errorf("got %v", got)
	}
}

func TestFnMarkdownify_Bold(t *testing.T) {
	got := string(fnMarkdownify("**bold**"))
	want := "<strong>bold</strong>"
	if got != want {
		t.Errorf("markdownify(**bold**) = %q, want %q", got, want)
	}
}

func TestFnMarkdownify_XSS(t *testing.T) {
	got := string(fnMarkdownify("<script>alert('xss')</script>"))
	if strings.Contains(got, "<script>") {
		t.Errorf("markdownify should not pass through script tags, got: %q", got)
	}
}

func TestFnMarkdownify_InlineCode(t *testing.T) {
	got := string(fnMarkdownify("use `fmt.Println`"))
	want := "use <code>fmt.Println</code>"
	if got != want {
		t.Errorf("markdownify(inline code) = %q, want %q", got, want)
	}
}

func TestFnMarkdownify_NoParagraphWrapper(t *testing.T) {
	got := string(fnMarkdownify("`IconManager` handles icons"))
	if strings.HasPrefix(got, "<p>") || strings.HasSuffix(got, "</p>") {
		t.Errorf("markdownify should not wrap in <p> tags, got: %q", got)
	}
	want := "<code>IconManager</code> handles icons"
	if got != want {
		t.Errorf("markdownify = %q, want %q", got, want)
	}
}

// ── Date tests ──

func TestFnDateFormat(t *testing.T) {
	tm := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got := fnDateFormat(tm, "2006-01-02")
	if got != "2025-03-15" {
		t.Errorf("got %q", got)
	}
}

// ── Math tests ──

func TestFnAdd(t *testing.T) {
	if got := fnAdd(3, 4); got != 7 {
		t.Errorf("int+int: got %v", got)
	}
	if got := fnAdd(3.5, 1.5); got != 5.0 {
		t.Errorf("float+float: got %v", got)
	}
}

func TestFnSub(t *testing.T) {
	if got := fnSub(10, 3); got != 7 {
		t.Errorf("got %v", got)
	}
}

func TestFnMul(t *testing.T) {
	if got := fnMul(3, 4); got != 12 {
		t.Errorf("got %v", got)
	}
}

func TestFnDiv(t *testing.T) {
	if got := fnDiv(10, 3); got != 3 {
		t.Errorf("int div: got %v", got)
	}
	if got := fnDiv(10, 0); got != 0 {
		t.Errorf("div by zero: got %v", got)
	}
}

func TestFnMod(t *testing.T) {
	if got := fnMod(10, 3); got != 1 {
		t.Errorf("got %v", got)
	}
}

// ── Logic tests ──

func TestFnCond(t *testing.T) {
	if got := fnCond(true, "yes", "no"); got != "yes" {
		t.Errorf("got %v", got)
	}
	if got := fnCond(false, "yes", "no"); got != "no" {
		t.Errorf("got %v", got)
	}
}

func TestFnDefault(t *testing.T) {
	if got := fnDefault("", "fallback"); got != "fallback" {
		t.Errorf("got %v", got)
	}
	if got := fnDefault("value", "fallback"); got != "value" {
		t.Errorf("got %v", got)
	}
	if got := fnDefault(nil, "fallback"); got != "fallback" {
		t.Errorf("got %v", got)
	}
}

func TestFnIsset(t *testing.T) {
	m := map[string]any{"key": "val"}
	if !fnIsset(m, "key") {
		t.Error("expected true for existing key")
	}
	if fnIsset(m, "missing") {
		t.Error("expected false for missing key")
	}
}

// ── Collection tests ──

func TestFnFirst(t *testing.T) {
	list := []string{"a", "b", "c", "d"}
	got := fnFirst(2, list).([]string)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %v", got)
	}
}

func TestFnLast(t *testing.T) {
	list := []string{"a", "b", "c", "d"}
	got := fnLast(2, list).([]string)
	if len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Errorf("got %v", got)
	}
}

func TestFnAfter(t *testing.T) {
	list := []string{"a", "b", "c"}
	got := fnAfter(1, list).([]string)
	if len(got) != 2 || got[0] != "b" {
		t.Errorf("got %v", got)
	}
}

// Negative n values must be clamped, not panic via reflect.Slice / make.
func TestFnFirst_NegativeN(t *testing.T) {
	list := []string{"a", "b", "c"}
	got := fnFirst(-1, list).([]string)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestFnLast_NegativeN(t *testing.T) {
	list := []string{"a", "b", "c"}
	got := fnLast(-1, list).([]string)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestFnAfter_NegativeN(t *testing.T) {
	list := []string{"a", "b", "c"}
	got := fnAfter(-1, list).([]string)
	if len(got) != 3 {
		t.Errorf("got %v, want full list", got)
	}
}

func TestFnSeq_NegativeN(t *testing.T) {
	if got := fnSeq(-1); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestFnIn(t *testing.T) {
	list := []string{"a", "b", "c"}
	if !fnIn(list, "b") {
		t.Error("expected true")
	}
	if fnIn(list, "d") {
		t.Error("expected false")
	}
}

func TestFnUniq(t *testing.T) {
	list := []string{"a", "b", "a", "c", "b"}
	got := fnUniq(list).([]string)
	if len(got) != 3 {
		t.Errorf("got %d items: %v", len(got), got)
	}
}

func TestFnSeq(t *testing.T) {
	got := fnSeq(3)
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("got %v", got)
	}

	got = fnSeq(2, 5)
	if len(got) != 4 || got[0] != 2 || got[3] != 5 {
		t.Errorf("got %v", got)
	}
}

// ── URL tests ──

func TestAbsURL(t *testing.T) {
	fm := testFuncMapBuild()
	fn := fm["absURL"].(func(string) string)

	if got := fn("/blog/post/"); got != "https://example.com/blog/post/" {
		t.Errorf("got %q", got)
	}
}

func TestRelURL(t *testing.T) {
	fm := testFuncMapBuild()
	fn := fm["relURL"].(func(string) string)

	if got := fn("/blog/post/"); got != "/blog/post/" {
		t.Errorf("root deploy: got %q, want %q", got, "/blog/post/")
	}
}

func TestRelURL_WithBasePath(t *testing.T) {
	e := &Engine{
		resolver:    &engine.ThemeResolver{},
		site:        testSite(),
		urlResolver: &engine.URLResolver{BasePath: "/docs/", BaseURL: "https://example.com"},
	}
	e.currentLang = "en"
	fm := e.buildFuncMap(nil, nil)
	fn := fm["relURL"].(func(string) string)

	if got := fn("/blog/post/"); got != "/docs/blog/post/" {
		t.Errorf("subdir deploy: got %q, want %q", got, "/docs/blog/post/")
	}
}

// ── Cross-collection tests ──

func TestRecentEntries(t *testing.T) {
	fm := testFuncMapBuild()
	fn := fm["recentEntries"].(func(string, int) []*engine.Page)

	pages := fn("blog", 2)
	if len(pages) != 2 {
		t.Errorf("got %d pages", len(pages))
	}
	if pages[0].Title != "Post A" {
		t.Errorf("got %q", pages[0].Title)
	}

	// Negative counts (e.g. from template arithmetic) must clamp, not panic.
	if got := fn("blog", -3); len(got) != 0 {
		t.Errorf("negative n: got %d pages, want 0", len(got))
	}
}

func TestFindEntry(t *testing.T) {
	fm := testFuncMapBuild()
	fn := fm["findEntry"].(func(string, string) *engine.Page)

	page := fn("blog", "post-b")
	if page == nil {
		t.Fatal("expected page")
	}
	if page.Title != "Post B" {
		t.Errorf("got %q", page.Title)
	}

	if fn("blog", "nonexistent") != nil {
		t.Error("expected nil for missing page")
	}
}

// ── Debug tests ──

func TestFnJsonify(t *testing.T) {
	got := fnJsonify(map[string]int{"a": 1})
	if got == "" {
		t.Error("expected non-empty output")
	}
}

func TestFnDump(t *testing.T) {
	got := fnDump("hello")
	if got == "" {
		t.Error("expected non-empty output")
	}
}

// ── Navigation helper tests ──

func TestNavFor(t *testing.T) {
	tree := &engine.NavTree{Root: &engine.NavNode{Label: "Docs"}}
	site := &engine.SiteContext{
		Collections: map[string]*engine.Collection{
			"docs": {Name: "docs", NavTree: tree},
		},
	}
	e := &Engine{site: site}
	fm := e.buildFuncMap(nil, nil)
	navFor := fm["navFor"].(func(string) *engine.NavTree)

	if got := navFor("docs"); got != tree {
		t.Errorf("navFor(docs) = %v, want %v", got, tree)
	}
	if got := navFor("nonexistent"); got != nil {
		t.Errorf("navFor(nonexistent) = %v, want nil", got)
	}
}

func TestBreadcrumbs(t *testing.T) {
	fm := nilFuncMap()
	breadcrumbs := fm["breadcrumbs"].(func(any) []engine.BreadcrumbItem)

	items := []engine.BreadcrumbItem{
		{Label: "Home", URL: "/"},
		{Label: "Docs", URL: "/docs/"},
	}
	rd := &engine.RouteData{RouteNav: engine.RouteNav{Breadcrumbs: items}}

	got := breadcrumbs(rd)
	if len(got) != 2 || got[0].Label != "Home" {
		t.Errorf("breadcrumbs(rd) = %v, want %v", got, items)
	}
	if breadcrumbs(nil) != nil {
		t.Error("breadcrumbs(nil) should return nil")
	}
}

func TestSiblings(t *testing.T) {
	fm := nilFuncMap()
	siblings := fm["siblings"].(func(*engine.Page) []*engine.Page)

	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "A"}},
		{PageIdentity: engine.PageIdentity{Title: "B"}},
	}
	section := &engine.Section{Pages: pages}
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "A"},
		PageRelationships: engine.PageRelationships{Section: section},
	}

	if got := siblings(page); len(got) != 2 {
		t.Errorf("siblings() = %d pages, want 2", len(got))
	}
	if siblings(nil) != nil {
		t.Error("siblings(nil) should return nil")
	}
	if siblings(&engine.Page{}) != nil {
		t.Error("siblings(no section) should return nil")
	}
}

func TestTranslations(t *testing.T) {
	fm := nilFuncMap()
	translations := fm["translations"].(func(any) []engine.TranslationLink)

	links := []engine.TranslationLink{
		{Lang: "en", URL: "/about/"},
		{Lang: "fr", URL: "/fr/about/"},
	}
	rd := &engine.RouteData{RouteI18n: engine.RouteI18n{Translations: links}}

	got := translations(rd)
	if len(got) != 2 || got[1].Lang != "fr" {
		t.Errorf("translations(rd) = %v, want %v", got, links)
	}
	if translations(nil) != nil {
		t.Error("translations(nil) should return nil")
	}
}

func TestToString(t *testing.T) {
	fm := nilFuncMap()
	toString := fm["toString"].(func(any) string)

	if got := toString(42); got != "42" {
		t.Errorf("toString(42) = %q, want %q", got, "42")
	}
	if got := toString(3.14); got != "3.14" {
		t.Errorf("toString(3.14) = %q, want %q", got, "3.14")
	}
}

func TestToInt(t *testing.T) {
	fm := nilFuncMap()
	toInt := fm["toInt"].(func(any) int)

	if got := toInt(42); got != 42 {
		t.Errorf("toInt(42) = %d, want 42", got)
	}
	if got := toInt(3.7); got != 3 {
		t.Errorf("toInt(3.7) = %d, want 3", got)
	}
	if got := toInt("not a number"); got != 0 {
		t.Errorf("toInt(string) = %d, want 0", got)
	}
}

func TestLang(t *testing.T) {
	fm := nilFuncMap()
	lang := fm["lang"].(func(any) string)

	rd := &engine.RouteData{RouteI18n: engine.RouteI18n{Lang: "fr"}}
	if got := lang(rd); got != "fr" {
		t.Errorf("lang(rd) = %q, want %q", got, "fr")
	}
	if got := lang(nil); got != "" {
		t.Errorf("lang(nil) = %q, want empty", got)
	}
}

func TestResizeImageFunc_NilProcessor(t *testing.T) {
	fm := nilFuncMap()
	resizeImage := fm["resize_image"].(func(engine.Resource, string) htmltemplate.HTML)

	res := engine.Resource{
		Name:         "hero.jpg",
		RelPermalink: "/blog/post/hero.jpg",
		Width:        1200,
		Height:       800,
		Title:        "Hero",
	}

	got := resizeImage(res, "width=800&op=fill")
	if got == "" {
		t.Error("expected non-empty HTML from resize_image fallback")
	}
	// With nil processor, should fall back to simple image rendering.
	if !strings.Contains(string(got), `src="/blog/post/hero.jpg"`) {
		t.Errorf("expected fallback src, got %s", got)
	}
}

func TestResizeImageFunc_WithProcessor(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestPNGForTemplate(t, srcDir, "hero.png", 1600, 900)

	processor := &asset.ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400, 800},
			Quality: 80,
		},
		Cache: asset.NewCache(t.TempDir()),
	}

	e := &Engine{imageProcessor: processor}
	fm := e.buildFuncMap(nil, nil)
	resizeImage := fm["resize_image"].(func(engine.Resource, string) htmltemplate.HTML)

	res := engine.Resource{
		Name:         "hero.png",
		RelPermalink: "/blog/post/hero.png",
		Width:        1600,
		Height:       900,
		Title:        "Hero",
		SrcPath:      srcPath,
	}

	got := resizeImage(res, "width=800&format=jpeg")
	html := string(got)

	if !strings.Contains(html, "<picture>") {
		t.Error("expected <picture> element from resize_image with processor")
	}
	if !strings.Contains(html, "srcset=") {
		t.Error("expected srcset attribute")
	}
}

func createTestPNGForTemplate(t *testing.T, dir, name string, width, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("creating test image: %v", err)
	}
	defer f.Close()
	png.Encode(f, img)
	return p
}

// ── fontUsed tests ──

func TestFontUsed(t *testing.T) {
	fm := nilFuncMap()
	fontUsed := fm["fontUsed"].(func(any, string) bool)

	rd := &engine.RouteData{Theme: &engine.ThemeConfig{
		Tokens: map[string]string{
			"font-sans": "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
			"font-mono": "'JetBrains Mono', ui-monospace, monospace",
		},
	}}
	if fontUsed(rd, "Inter") {
		t.Error("fontUsed = true for system-font stack, want false")
	}
	if !fontUsed(rd, "JetBrains Mono") {
		t.Error("fontUsed = false for JetBrains Mono in font-mono, want true")
	}

	rd.Theme.Tokens["font-sans"] = "'Inter', system-ui, sans-serif"
	if !fontUsed(rd, "Inter") {
		t.Error("fontUsed = false when font-sans references Inter, want true")
	}
	if !fontUsed(rd, "inter") {
		t.Error("fontUsed should match case-insensitively")
	}

	if fontUsed(nil, "Inter") {
		t.Error("fontUsed = true for nil data, want false")
	}
	if fontUsed(&engine.RouteData{}, "Inter") {
		t.Error("fontUsed = true for RouteData without theme, want false")
	}
}

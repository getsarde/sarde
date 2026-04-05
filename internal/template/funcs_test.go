package template

import (
	htmltemplate "html/template"
	"sync"
	"testing"
	"time"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

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
					{Title: "Post A", Slug: "post-a", RelPermalink: "/blog/post-a/"},
					{Title: "Post B", Slug: "post-b", RelPermalink: "/blog/post-b/"},
					{Title: "Post C", Slug: "post-c", RelPermalink: "/blog/post-c/"},
				},
			},
		},
		Pages: []*engine.Page{
			{Title: "Post A", Slug: "post-a", Permalink: "https://example.com/blog/post-a/", RelPermalink: "/blog/post-a/"},
		},
	}
}

func testFuncMapBuild() htmltemplate.FuncMap {
	return buildFuncMap(testSite(), &engine.ThemeResolver{}, nil, &sync.Map{})
}

// ── String tests ──

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

	if got := fn("https://example.com/blog/post/"); got != "/blog/post/" {
		t.Errorf("got %q", got)
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

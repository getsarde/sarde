package template

import (
	"html/template"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/frostybee/sarde/internal/engine"
)

func embeddedTestFS() fstest.MapFS {
	return fstest.MapFS{
		"_default/baseof.html": {Data: []byte(`<!DOCTYPE html>
<html lang="{{ .Lang }}">
<head>{{ component "Head" . }}</head>
<body>
{{ block "content" . }}{{ .Page.Content }}{{ end }}
{{ component "Footer" . }}
</body>
</html>`)},
		"_default/single.html": {Data: []byte(`{{ define "content" }}<article><h1>{{ .Page.Title }}</h1>{{ .Page.Content }}{{ if .Pagination }}{{ component "Pagination" . }}{{ end }}</article>{{ end }}`)},
		"_default/list.html": {Data: []byte(`{{ define "content" }}<h1>{{ .Page.Title }}</h1>{{ if .Collection }}<ul>{{ range .Collection.Pages }}<li>{{ .Title }}</li>{{ end }}</ul>{{ end }}{{ end }}`)},
		"_default/home.html": {Data: []byte(`{{ define "content" }}<div class="home"><h1>{{ .Site.Title }}</h1>{{ .Page.Content }}</div>{{ end }}`)},
		"_default/404.html":  {Data: []byte(`{{ define "content" }}<h1>Not Found</h1>{{ end }}`)},
		"_docs/baseof.html": {Data: []byte(`<!DOCTYPE html>
<html lang="{{ .Lang }}">
<head>{{ component "Head" . }}</head>
<body class="docs">
{{ if .HasSidebar }}<aside>sidebar</aside>{{ end }}
<main>{{ block "content" . }}{{ .Page.Content }}{{ end }}</main>
</body>
</html>`)},
		"_docs/single.html": {Data: []byte(`{{ define "content" }}<h1>{{ .Page.Title }}</h1>{{ .Page.Content }}{{ end }}`)},
		"components/Head.html":       {Data: []byte(`<title>{{ if .Page }}{{ .Page.Title }} - {{ end }}{{ .Site.Title }}</title>`)},
		"components/Footer.html":     {Data: []byte(`<footer>&copy; {{ .Site.Title }}</footer>`)},
		"components/Pagination.html": {Data: []byte(`{{ if .Pagination }}<nav>{{ with .Pagination.Prev }}<a href="{{ .URL }}">{{ .Title }}</a>{{ end }}{{ with .Pagination.Next }}<a href="{{ .URL }}">{{ .Title }}</a>{{ end }}</nav>{{ end }}`)},
	}
}

func setupEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()

	site := &engine.SiteContext{
		Title:       "Test Site",
		BaseURL:     "https://example.com",
		Language:    "en",
		Collections: make(map[string]*engine.Collection),
	}

	eng := NewEngine()
	eng.SetSiteContext(site)

	resolver := &engine.ThemeResolver{
		ProjectDir: dir,
		EmbeddedFS: embeddedTestFS(),
	}

	if err := eng.Load(resolver); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	return eng
}

func TestEngine_Load_Succeeds(t *testing.T) {
	setupEngine(t)
}

func TestEngine_Render_SinglePage(t *testing.T) {
	eng := setupEngine(t)

	page := &engine.Page{
		Title:   "Hello World",
		Kind:    engine.KindPage,
		Content: template.HTML("<p>This is content.</p>"),
	}
	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page.Collection = col

	rd := BuildRouteData(page, eng.site, nil)

	html, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	output := string(html)
	if !strings.Contains(output, "<h1>Hello World</h1>") {
		t.Error("missing page title")
	}
	if !strings.Contains(output, "<p>This is content.</p>") {
		t.Error("missing page content")
	}
	if !strings.Contains(output, "Test Site") {
		t.Error("missing site title in head/footer")
	}
	if !strings.Contains(output, `lang="en"`) {
		t.Error("missing lang attribute")
	}
}

func TestEngine_Render_DocsPage(t *testing.T) {
	eng := setupEngine(t)

	page := &engine.Page{
		Title:   "Getting Started",
		Kind:    engine.KindPage,
		Content: template.HTML("<p>Docs content.</p>"),
	}
	col := &engine.Collection{
		Name:   "docs",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDocs},
	}
	page.Collection = col

	rd := BuildRouteData(page, eng.site, nil)

	html, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	output := string(html)
	if !strings.Contains(output, "class=\"docs\"") {
		t.Error("missing docs class")
	}
	if !strings.Contains(output, "<h1>Getting Started</h1>") {
		t.Error("missing page title")
	}
	if !strings.Contains(output, "<aside>sidebar</aside>") {
		t.Error("missing sidebar for docs layout")
	}
}

func TestEngine_Render_HomePage(t *testing.T) {
	eng := setupEngine(t)

	page := &engine.Page{
		Title:   "Welcome",
		Kind:    engine.KindHome,
		Content: template.HTML("<p>Welcome!</p>"),
	}

	rd := BuildRouteData(page, eng.site, nil)

	html, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	output := string(html)
	if !strings.Contains(output, `class="home"`) {
		t.Error("missing home class")
	}
	if !strings.Contains(output, "Test Site") {
		t.Error("missing site title")
	}
}

func TestEngine_Render_WithPagination(t *testing.T) {
	eng := setupEngine(t)

	prev := &engine.Page{Title: "Previous", RelPermalink: "/blog/prev/"}
	next := &engine.Page{Title: "Next", RelPermalink: "/blog/next/"}
	page := &engine.Page{
		Title:    "Current Post",
		Kind:     engine.KindPage,
		Content:  template.HTML("<p>Content.</p>"),
		PrevPage: prev,
		NextPage: next,
	}
	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page.Collection = col

	rd := BuildRouteData(page, eng.site, nil)

	html, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	output := string(html)
	if !strings.Contains(output, "/blog/prev/") {
		t.Error("missing prev link")
	}
	if !strings.Contains(output, "/blog/next/") {
		t.Error("missing next link")
	}
}

func TestEngine_Render_ListPage(t *testing.T) {
	eng := setupEngine(t)

	pages := []*engine.Page{
		{Title: "Post A"},
		{Title: "Post B"},
	}
	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
		Pages:  pages,
	}
	section := &engine.Section{Title: "Blog"}
	page := &engine.Page{
		Title:   "Blog",
		Kind:    engine.KindSection,
		Section: section,
	}
	page.Collection = col

	rd := BuildRouteData(page, eng.site, nil)

	html, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	output := string(html)
	if !strings.Contains(output, "Post A") {
		t.Error("missing Post A in list")
	}
	if !strings.Contains(output, "Post B") {
		t.Error("missing Post B in list")
	}
}

func TestEngine_Render_CachesTemplates(t *testing.T) {
	eng := setupEngine(t)

	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page := &engine.Page{
		Title:      "Post",
		Kind:       engine.KindPage,
		Content:    template.HTML("<p>Content</p>"),
		Collection: col,
	}

	rd := BuildRouteData(page, eng.site, nil)

	// First render
	_, err := eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	// Second render should use cache
	_, err = eng.Render(rd.Template, rd)
	if err != nil {
		t.Fatal(err)
	}

	// Verify cache has the entry
	eng.mu.RLock()
	_, cached := eng.templates["default:blog/single"]
	eng.mu.RUnlock()
	if !cached {
		t.Error("template was not cached")
	}
}

func TestEngine_Render_UnknownTemplate(t *testing.T) {
	eng := setupEngine(t)

	rd := &engine.RouteData{
		Template: "nonexistent/template",
		Layout:   engine.LayoutDefault,
		Lang:     "en",
		Dir:      "ltr",
		Page:     &engine.Page{},
		Site:     eng.site,
	}

	_, err := eng.Render(rd.Template, rd)
	if err == nil {
		t.Error("expected error for unknown template")
	}
}

package component

import (
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/engine"
)

func testFuncMap() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"now": func() struct{ Year int } { return struct{ Year int }{2025} },
		"component": func(name string, data any) htmltemplate.HTML {
			return htmltemplate.HTML("<!-- " + name + " -->")
		},
	}
}

func TestNewRegistry_LoadsEmbeddedComponents(t *testing.T) {
	efs := fstest.MapFS{
		"components/Header.html": {Data: []byte(`<header>Test</header>`)},
		"components/Footer.html": {Data: []byte(`<footer>Test</footer>`)},
	}

	r, err := NewRegistry(efs, testFuncMap())
	if err != nil {
		t.Fatal(err)
	}

	if r.Resolve("Header") == nil {
		t.Error("Header component not loaded")
	}
	if r.Resolve("Footer") == nil {
		t.Error("Footer component not loaded")
	}
}

func TestRegistry_Register_Override(t *testing.T) {
	efs := fstest.MapFS{
		"components/Header.html": {Data: []byte(`<header>Default</header>`)},
	}

	r, err := NewRegistry(efs, testFuncMap())
	if err != nil {
		t.Fatal(err)
	}

	// Override with custom
	if err := r.Register("Header", []byte(`<header>Custom</header>`)); err != nil {
		t.Fatal(err)
	}

	html, err := r.RenderComponent("Header", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(html) != "<header>Custom</header>" {
		t.Errorf("got %q, want custom header", html)
	}
}

func TestRegistry_RenderComponent_UnknownSlot(t *testing.T) {
	r, _ := NewRegistry(nil, testFuncMap())

	html, err := r.RenderComponent("Nonexistent", nil)
	if err != nil {
		t.Fatal("expected no error for unknown slot")
	}
	if html != "" {
		t.Errorf("expected empty HTML for unknown slot, got %q", html)
	}
}

func TestRegistry_RenderComponent_WithData(t *testing.T) {
	r, _ := NewRegistry(nil, testFuncMap())

	err := r.Register("PageTitle", []byte(`<h1>{{ .Title }}</h1>`))
	if err != nil {
		t.Fatal(err)
	}

	data := struct{ Title string }{"Hello World"}
	html, err := r.RenderComponent("PageTitle", data)
	if err != nil {
		t.Fatal(err)
	}
	if string(html) != "<h1>Hello World</h1>" {
		t.Errorf("got %q", html)
	}
}

func TestRegistry_LoadOverridesFromDir(t *testing.T) {
	efs := fstest.MapFS{
		"components/Header.html": {Data: []byte(`<header>Default</header>`)},
		"components/Footer.html": {Data: []byte(`<footer>Default</footer>`)},
	}

	r, err := NewRegistry(efs, testFuncMap())
	if err != nil {
		t.Fatal(err)
	}

	// Create user overrides
	dir := t.TempDir()
	compDir := filepath.Join(dir, "components")
	os.MkdirAll(compDir, 0o755)
	os.WriteFile(filepath.Join(compDir, "Header.html"), []byte(`<header>User Override</header>`), 0o644)

	if err := r.LoadOverridesFromDir(compDir); err != nil {
		t.Fatal(err)
	}

	// Header should be overridden
	html, _ := r.RenderComponent("Header", nil)
	if string(html) != "<header>User Override</header>" {
		t.Errorf("Header: got %q", html)
	}

	// Footer should still be default
	html, _ = r.RenderComponent("Footer", nil)
	if string(html) != "<footer>Default</footer>" {
		t.Errorf("Footer: got %q", html)
	}
}

func TestRegistry_LoadOverridesFromDir_NonexistentDir(t *testing.T) {
	r, _ := NewRegistry(nil, testFuncMap())

	err := r.LoadOverridesFromDir("/nonexistent/path")
	if err != nil {
		t.Errorf("expected nil error for nonexistent dir, got %v", err)
	}
}

func TestAllSlots(t *testing.T) {
	slots := AllSlots()
	if len(slots) != 25 {
		t.Errorf("expected 25 slots, got %d", len(slots))
	}
}

// TestEmbeddedPageTitle_HidesInferredDescription checks that the shipped
// PageTitle component shows an author-written description but skips one
// inferred from the first paragraph, which the body already renders.
func TestEmbeddedPageTitle_HidesInferredDescription(t *testing.T) {
	src, err := fs.ReadFile(embedded.ThemeFS(), "components/PageTitle.html")
	if err != nil {
		t.Fatal(err)
	}
	funcs := testFuncMap()
	funcs["markdownify"] = func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) }
	funcs["icon"] = func(string) htmltemplate.HTML { return "" }
	r, err := NewRegistry(fstest.MapFS{"components/PageTitle.html": {Data: src}}, funcs)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		inferred bool
		want     bool
	}{
		{"explicit", false, true},
		{"inferred", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &engine.Page{
				PageIdentity: engine.PageIdentity{Title: "Title"},
				PageMeta:     engine.PageMeta{Description: "Opening sentence.", DescriptionInferred: tt.inferred},
			}
			html, err := r.RenderComponent("PageTitle", map[string]any{"Page": page})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(html), "sarde-page-description"); got != tt.want {
				t.Errorf("description shown = %v, want %v; html: %s", got, tt.want, html)
			}
		})
	}
}

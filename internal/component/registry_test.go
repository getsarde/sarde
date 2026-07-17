package component

import (
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
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
	if len(slots) != 24 {
		t.Errorf("expected 24 slots, got %d", len(slots))
	}
}

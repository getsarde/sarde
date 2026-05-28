package template

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
)

func TestEngine_Render_ConcurrentLanguagesUseRouteData(t *testing.T) {
	dir := t.TempDir()
	writeTemplateI18n(t, dir, "en.yaml", "label: Search\ncomponent: Component\npartial: Partial\n")
	writeTemplateI18n(t, dir, "fr.yaml", "label: Rechercher\ncomponent: Composant\npartial: Partiel\n")
	writeTemplateFile(t, dir, filepath.Join("layouts", "partials", "translated.html"), `{{ t "partial" }}`)

	st, err := i18n.LoadStrings(nil, dir, "", "en")
	if err != nil {
		t.Fatalf("LoadStrings failed: %v", err)
	}

	site := &engine.SiteContext{Title: "Site", Language: "en", Collections: make(map[string]*engine.Collection)}
	eng := NewEngine()
	eng.SetSiteContext(site)
	eng.SetI18nStrings(st)
	if err := eng.Load(&engine.ThemeResolver{
		ProjectDir: dir,
		EmbeddedFS: fstest.MapFS{
			"_default/baseof.html":       {Data: []byte(`{{ t "label" }} {{ component "Translated" . }} {{ partial "translated.html" . }} {{ block "content" . }}{{ end }}`)},
			"_default/single.html":       {Data: []byte(`{{ define "content" }}{{ .Page.Title }}{{ end }}`)},
			"components/Translated.html": {Data: []byte(`{{ t "component" }}`)},
		},
	}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	enRD := BuildRouteData(&engine.Page{PageIdentity: engine.PageIdentity{Title: "English"}, PageI18n: engine.PageI18n{Lang: "en"}}, site, nil)
	frRD := BuildRouteData(&engine.Page{PageIdentity: engine.PageIdentity{Title: "French"}, PageI18n: engine.PageI18n{Lang: "fr"}}, site, nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			html, err := eng.Render(enRD.Template, enRD)
			if err != nil {
				t.Errorf("English render failed: %v", err)
				return
			}
			out := string(html)
			if !strings.Contains(out, "Search Component Partial English") || strings.Contains(out, "Rechercher") {
				t.Errorf("English render used wrong language: %q", out)
			}
		}()
		go func() {
			defer wg.Done()
			html, err := eng.Render(frRD.Template, frRD)
			if err != nil {
				t.Errorf("French render failed: %v", err)
				return
			}
			out := string(html)
			if !strings.Contains(out, "Rechercher Composant Partiel French") || strings.Contains(out, "Search") {
				t.Errorf("French render used wrong language: %q", out)
			}
		}()
	}
	wg.Wait()
}

func writeTemplateI18n(t *testing.T, dir, rel, body string) {
	t.Helper()
	path := filepath.Join(dir, "i18n", filepath.FromSlash(rel))
	writeTemplateFileAtPath(t, path, body)
}

func writeTemplateFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	writeTemplateFileAtPath(t, filepath.Join(dir, filepath.FromSlash(rel)), body)
}

func writeTemplateFileAtPath(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

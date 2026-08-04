package telescope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/plugin"
)

func testStringTable(t *testing.T) *i18n.StringTable {
	t.Helper()
	dir := t.TempDir()
	i18nDir := filepath.Join(dir, "i18n")
	os.MkdirAll(i18nDir, 0o755)
	os.WriteFile(filepath.Join(i18nDir, "en.yaml"), []byte(`
telescope:
  trigger: "Quick navigation"
  placeholder: "Search pages..."
  pin: "Pin page"
`), 0o644)
	os.WriteFile(filepath.Join(i18nDir, "fr.yaml"), []byte(`
telescope:
  trigger: "Navigation rapide"
  placeholder: "Rechercher des pages..."
  pin: "Épingler la page"
`), 0o644)
	st, err := i18n.LoadStrings(nil, dir, "", "en")
	if err != nil {
		t.Fatalf("LoadStrings: %v", err)
	}
	return st
}

func runBeforeRender(t *testing.T, p *plugin.Plugin, pg *engine.Page) *engine.RouteData {
	t.Helper()
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{Page: pg, RouteData: rd}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	return rd
}

func TestBeforeRender_InjectsAssetsAndConfig(t *testing.T) {
	p := New(nil, nil)
	pg := &engine.Page{}
	pg.Lang = "en"
	rd := runBeforeRender(t, p, pg)

	if len(rd.Styles) != 1 || !strings.Contains(rd.Styles[0], "assets/plugins/telescope/telescope.") {
		t.Errorf("Styles = %v, want fingerprinted telescope CSS", rd.Styles)
	}
	if len(rd.Scripts) != 1 || !strings.Contains(rd.Scripts[0], "assets/plugins/telescope/telescope.") {
		t.Errorf("Scripts = %v, want fingerprinted telescope JS", rd.Scripts)
	}
	if len(rd.InlineScripts) != 1 {
		t.Fatalf("InlineScripts len = %d, want 1", len(rd.InlineScripts))
	}

	script := string(rd.InlineScripts[0])
	// Defensive merge: the script must never replace pluginConfig wholesale,
	// or it would clobber the clientplugins loader's assignment (and vice
	// versa, depending on registration order).
	if !strings.Contains(script, "window.__SARDE__.pluginConfig=window.__SARDE__.pluginConfig||{}") {
		t.Errorf("inline script must merge into pluginConfig, got: %s", script)
	}
	if !strings.Contains(script, `"shortcut_key":"/"`) {
		t.Errorf("inline script missing default shortcut_key: %s", script)
	}
}

func TestBeforeRender_ResolvesI18nStringsPerLang(t *testing.T) {
	st := testStringTable(t)
	p := New(nil, st)

	en := &engine.Page{}
	en.Lang = "en"
	fr := &engine.Page{}
	fr.Lang = "fr"

	enScript := string(runBeforeRender(t, p, en).InlineScripts[0])
	frScript := string(runBeforeRender(t, p, fr).InlineScripts[0])

	if !strings.Contains(enScript, "Quick navigation") {
		t.Errorf("en script missing English label: %s", enScript)
	}
	if !strings.Contains(frScript, "Navigation rapide") {
		t.Errorf("fr script missing French label: %s", frScript)
	}
}

func TestBeforeRender_PlaceholderOverride(t *testing.T) {
	st := testStringTable(t)
	p := New(map[string]any{"placeholder": "Go to..."}, st)
	pg := &engine.Page{}
	pg.Lang = "en"
	script := string(runBeforeRender(t, p, pg).InlineScripts[0])
	if !strings.Contains(script, `"placeholder":"Go to..."`) {
		t.Errorf("configured placeholder should win over i18n default: %s", script)
	}
}

func TestBuildDone_WritesAssetsAndIndex(t *testing.T) {
	p := New(map[string]any{"exclude": []any{"/hidden/*"}}, nil)

	intro := &engine.Page{}
	intro.Kind = engine.KindPage
	intro.RelPermalink = "/docs/intro/"
	intro.Title = "Intro"
	hidden := &engine.Page{}
	hidden.Kind = engine.KindPage
	hidden.RelPermalink = "/hidden/secret/"
	hidden.Title = "Secret"

	outDir := t.TempDir()
	ctx := &plugin.BuildDoneContext{
		OutputDir: outDir,
		Pages:     []*engine.Page{intro, hidden},
	}
	if err := p.Hooks.BuildDone(ctx); err != nil {
		t.Fatalf("BuildDone: %v", err)
	}

	indexData, err := os.ReadFile(filepath.Join(outDir, "telescope-pages.json"))
	if err != nil {
		t.Fatalf("index not written: %v", err)
	}
	var entries []indexEntry
	if err := json.Unmarshal(indexData, &entries); err != nil {
		t.Fatalf("index not valid JSON: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "/docs/intro/" {
		t.Errorf("index entries = %+v, want only /docs/intro/", entries)
	}

	assetDir := filepath.Join(outDir, "assets", "plugins", "telescope")
	files, err := os.ReadDir(assetDir)
	if err != nil {
		t.Fatalf("asset dir missing: %v", err)
	}
	var haveJS, haveCSS bool
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".js") {
			haveJS = true
		}
		if strings.HasSuffix(f.Name(), ".css") {
			haveCSS = true
		}
	}
	if !haveJS || !haveCSS {
		t.Errorf("expected fingerprinted JS and CSS in %s, got %v", assetDir, files)
	}
}

func TestAssets_BundleContainsFuseAndRuntime(t *testing.T) {
	ensureAssets()
	if len(jsData) == 0 {
		t.Fatal("jsData is empty; fuse.min.js or telescope.js missing from embed")
	}
	js := string(jsData)
	if !strings.Contains(js, "Fuse.js") {
		t.Error("bundled JS missing Fuse.js library header")
	}
	if !strings.Contains(js, "sarde-telescope-dialog") {
		t.Error("bundled JS missing telescope runtime")
	}
	if len(cssData) == 0 {
		t.Fatal("cssData is empty")
	}
	if !strings.Contains(string(cssData), "@layer sarde.plugins") {
		t.Error("CSS must live in the sarde.plugins cascade layer")
	}
}

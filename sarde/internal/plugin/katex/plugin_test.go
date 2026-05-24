package katex

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"
)

func TestBeforeRender_InjectsOnMathContent(t *testing.T) {
	p := New(nil)
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page: &engine.Page{
			Content: template.HTML(`<p class="math display">x^2</p>`),
		},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Styles) != 1 {
		t.Errorf("Styles len = %d, want 1 (%v)", len(rd.Styles), rd.Styles)
	}
	if len(rd.Scripts) != 3 {
		t.Errorf("Scripts len = %d, want 3 (%v)", len(rd.Scripts), rd.Scripts)
	}
}

func TestBeforeRender_SkipsWithoutMath(t *testing.T) {
	p := New(nil)
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page:      &engine.Page{Content: template.HTML("<p>just text</p>")},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Styles) != 0 || len(rd.Scripts) != 0 {
		t.Errorf("expected no injection, got Styles=%v Scripts=%v", rd.Styles, rd.Scripts)
	}
}

func TestBeforeRender_AlwaysConfigForces(t *testing.T) {
	p := New(map[string]any{"always": true})
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page:      &engine.Page{Content: template.HTML("<p>plain</p>")},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) == 0 || len(rd.Styles) == 0 {
		t.Error("always=true should force injection even without math")
	}
}

func TestBuildDone_WritesAssetsToVendorDir(t *testing.T) {
	out := t.TempDir()
	p := New(nil)
	ctx := &plugin.BuildDoneContext{OutputDir: out}
	if err := p.Hooks.BuildDone(ctx); err != nil {
		t.Fatalf("BuildDone: %v", err)
	}

	for _, rel := range []string{
		"assets/vendor/katex/katex.min.js",
		"assets/vendor/katex/katex.min.css",
		"assets/vendor/katex/auto-render.min.js",
		"assets/vendor/katex/init.js",
	} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

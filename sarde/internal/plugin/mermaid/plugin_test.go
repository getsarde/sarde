package mermaid

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"
)

func TestBeforeRender_InjectsOnMermaidContent(t *testing.T) {
	p := New(nil)
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page:      &engine.Page{PageContent: engine.PageContent{Content: template.HTML(`<pre class="sarde-mermaid">graph TD</pre>`)}},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) != 2 {
		t.Errorf("Scripts len = %d, want 2 (%v)", len(rd.Scripts), rd.Scripts)
	}
}

func TestBeforeRender_SkipsWithoutMermaid(t *testing.T) {
	p := New(nil)
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page:      &engine.Page{PageContent: engine.PageContent{Content: template.HTML("<p>just text</p>")}},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) != 0 {
		t.Errorf("expected no scripts, got %v", rd.Scripts)
	}
}

func TestBeforeRender_AlwaysConfigForces(t *testing.T) {
	p := New(map[string]any{"always": true})
	rd := &engine.RouteData{}
	ctx := &plugin.BeforeRenderContext{
		Page:      &engine.Page{PageContent: engine.PageContent{Content: template.HTML("<p>plain</p>")}},
		RouteData: rd,
	}
	if err := p.Hooks.BeforeRender(ctx); err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) == 0 {
		t.Error("always=true should force script injection")
	}
}

func TestBuildDone_WritesAssets(t *testing.T) {
	out := t.TempDir()
	p := New(nil)
	ctx := &plugin.BuildDoneContext{OutputDir: out}
	if err := p.Hooks.BuildDone(ctx); err != nil {
		t.Fatalf("BuildDone: %v", err)
	}
	for _, rel := range []string{
		"assets/vendor/mermaid/mermaid.min.js",
		"assets/vendor/mermaid/init.js",
	} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

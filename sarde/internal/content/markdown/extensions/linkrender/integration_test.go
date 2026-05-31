package linkrender_test

import (
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/engine"
)

func TestLinkRenderHookIntegration(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	targetPage := &engine.Page{
		PageIdentity:    engine.PageIdentity{RelPermalink: "/docs/guide/installation/", Permalink: "/docs/guide/installation/", Slug: "installation"},
		PageI18n:        engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/02-installation.md"},
		PageVersioning:  engine.PageVersioning{VersionRelPath: "guide/02-installation.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	currentPage := &engine.Page{
		PageIdentity:    engine.PageIdentity{RelPermalink: "/docs/guide/quick-start/", Permalink: "/docs/guide/quick-start/", Slug: "quick-start"},
		PageI18n:        engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/03-quick-start.md"},
		PageVersioning:  engine.PageVersioning{VersionRelPath: "guide/03-quick-start.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}

	idx := content.BuildPageIndex([]*engine.Page{targetPage, currentPage})

	r := markdown.NewRendererFromConfig(markdown.RendererConfig{HeadingLinks: true})
	lr := r.LinkRenderer()
	lr.PageIndex = idx
	lr.URLResolver = &engine.URLResolver{BasePath: "/"}
	lr.Policy = "warn"

	r.SetLinkContext(currentPage)

	result, err := r.Render("See [Installation](./02-installation.md) for details.")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	t.Logf("HTML output: %s", result.HTML)

	if !strings.Contains(result.HTML, `href="/docs/guide/installation/"`) {
		t.Errorf("Expected resolved href, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "02-installation.md") {
		t.Errorf("Raw .md href should not appear in output: %s", result.HTML)
	}
}

package linkcard_test

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

func setupTestEnv() (*markdown.Renderer, *engine.Page, *links.LinkGraph) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	targetPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/", Slug: "auth"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{VersionRelPath: "guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/quick-start/", Permalink: "/docs/guide/quick-start/", Slug: "quick-start"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/quick-start.md"},
		PageVersioning:    engine.PageVersioning{VersionRelPath: "guide/quick-start.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}

	idx := content.BuildPageIndex([]*engine.Page{targetPage, currentPage})
	graph := links.NewLinkGraph()

	r := markdown.NewRendererFromConfig(markdown.RendererConfig{HeadingLinks: true})
	lr := r.LinkRenderer()
	lr.PageIndex = idx
	lr.URLResolver = &engine.URLResolver{BasePath: "/"}
	lr.LinkGraph = graph

	r.SetLinkContext(currentPage)

	return r, currentPage, graph
}

func TestLinkCardResolvesRelativeHref(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-card[Auth Guide]{href=\"./auth.md\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="/docs/guide/auth/"`) {
		t.Errorf("Expected resolved href in link-card, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "auth.md") {
		t.Errorf("Raw .md href should not appear in link-card output: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("Expected link graph to have recorded a ref")
	}
	if refs[0].Status != links.StatusOK {
		t.Errorf("Expected StatusOK, got: %v", refs[0].Status)
	}
}

func TestLinkCardBrokenTarget(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-card[Missing]{href=\"./nonexistent.md\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="#"`) {
		t.Errorf("Expected href='#' for broken target, got: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("Expected link graph to have recorded a ref for broken target")
	}
	if refs[0].Status != links.StatusBrokenTarget {
		t.Errorf("Expected StatusBrokenTarget, got: %v", refs[0].Status)
	}
}

func TestLinkCardExternalHrefPassesThrough(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-card[GitHub]{href=\"https://github.com\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="https://github.com"`) {
		t.Errorf("Expected external href to pass through, got: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("Expected link graph to have recorded an external ref")
	}
	if refs[0].Status != links.StatusExternal {
		t.Errorf("Expected StatusExternal, got: %v", refs[0].Status)
	}
}

func TestLinkCardSiteAbsoluteResolvesNoNewTab(t *testing.T) {
	r, _, graph := setupTestEnv()

	// Extension-less site-absolute href to an existing page.
	md := ":::link-card[Auth]{href=\"/docs/guide/auth\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="/docs/guide/auth/"`) {
		t.Errorf("Expected resolved site-absolute href, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, `target="_blank"`) {
		t.Errorf("Internal link-card must NOT open in a new tab, got: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 || refs[0].Status != links.StatusOK {
		t.Errorf("Expected StatusOK for resolved site-absolute link, got: %v", refs)
	}
}

func TestLinkCardInternalNoNewTab(t *testing.T) {
	r, _, _ := setupTestEnv()

	// A resolved relative link is internal — it must stay in the same tab.
	md := ":::link-card[Auth Guide]{href=\"./auth.md\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if strings.Contains(result.HTML, `target="_blank"`) {
		t.Errorf("Internal link-card must NOT open in a new tab, got: %s", result.HTML)
	}
}

func TestLinkCardExternalKeepsNewTab(t *testing.T) {
	r, _, _ := setupTestEnv()

	md := ":::link-card[GitHub]{href=\"https://github.com\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `target="_blank"`) {
		t.Errorf("External link-card must open in a new tab, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `rel="noopener noreferrer"`) {
		t.Errorf("External link-card must carry rel=noopener noreferrer, got: %s", result.HTML)
	}
}

func TestLinkCardContentRootHref(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-card[Auth]{href=\"/guide/auth.md\"}\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="/docs/guide/auth/"`) {
		t.Errorf("Expected resolved content-root href, got: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("Expected link graph to have recorded a ref")
	}
	if refs[0].Status != links.StatusOK {
		t.Errorf("Expected StatusOK, got: %v", refs[0].Status)
	}
}

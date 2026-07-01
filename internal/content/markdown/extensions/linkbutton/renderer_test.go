package linkbutton_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkbutton"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
	"github.com/yuin/goldmark"
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

func TestLinkButtonSiteAbsoluteResolvesNoNewTab(t *testing.T) {
	r, _, graph := setupTestEnv()

	// Extension-less site-absolute href resolves through the shared link renderer.
	md := ":::link-button[Auth](href=\"/docs/guide/auth\")\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="/docs/guide/auth/"`) {
		t.Errorf("Expected resolved site-absolute href in link-button, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, `target="_blank"`) {
		t.Errorf("Internal link-button must NOT open in a new tab, got: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 || refs[0].Status != links.StatusOK {
		t.Errorf("Expected StatusOK for resolved site-absolute link, got: %v", refs)
	}
}

func TestLinkButtonResolvesRelativeHref(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-button[Read Auth Guide](href=\"./auth.md\")\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result.HTML, `href="/docs/guide/auth/"`) {
		t.Errorf("Expected resolved href in link-button, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "auth.md") {
		t.Errorf("Raw .md href should not appear in link-button output: %s", result.HTML)
	}

	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("Expected link graph to have recorded a ref")
	}
	if refs[0].Status != links.StatusOK {
		t.Errorf("Expected StatusOK, got: %v", refs[0].Status)
	}
}

func TestLinkButtonBrokenTarget(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-button[Missing](href=\"./nonexistent.md\")\n:::"
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

func TestLinkButtonExternalHrefPassesThrough(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-button[GitHub](href=\"https://github.com\")\n:::"
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

func TestLinkButtonContentRootHref(t *testing.T) {
	r, _, graph := setupTestEnv()

	md := ":::link-button[Auth](href=\"/guide/auth.md\")\n:::"
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

// Per-block label state must not leak between blocks parsed by the same
// (shared) block-parser instance — regression for the hasLabel-on-parser bug.
func TestLinkButtonConsecutiveBlocks_BodyCaptureIndependent(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(&linkbutton.Extension{}))

	src := ":::link-button[Labeled](href=\"https://example.com/a\")\n" +
		"body must be ignored\n" +
		":::\n\n" +
		":::link-button(href=\"https://example.com/b\")\n" +
		"Body Becomes Label\n" +
		":::\n"

	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, ">Labeled</a>") {
		t.Errorf("first block should render its explicit label, got: %s", html)
	}
	if strings.Contains(html, "body must be ignored") {
		t.Errorf("first block has an explicit label; body lines must be ignored, got: %s", html)
	}
	if !strings.Contains(html, ">Body Becomes Label</a>") {
		t.Errorf("second block has no label; body must become the content, got: %s", html)
	}
}

// The same goldmark instance rendered from many goroutines must keep labeled
// and unlabeled blocks independent (per-block state lives on the AST node,
// not the shared parser).
func TestLinkButtonSharedParserConcurrent(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(&linkbutton.Extension{}))

	labeled := ":::link-button[Fixed](href=\"https://example.com/a\")\nignored\n:::\n"
	unlabeled := ":::link-button(href=\"https://example.com/b\")\nFromBody\n:::\n"

	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 6; i++ {
				src, wantContent, banned := labeled, ">Fixed</a>", "ignored"
				if (g+i)%2 == 0 {
					src, wantContent, banned = unlabeled, ">FromBody</a>", ""
				}
				var buf bytes.Buffer
				if err := md.Convert([]byte(src), &buf); err != nil {
					errs <- "convert: " + err.Error()
					return
				}
				out := buf.String()
				if !strings.Contains(out, wantContent) {
					errs <- "missing " + wantContent + " in: " + out
					return
				}
				if banned != "" && strings.Contains(out, banned) {
					errs <- "body leaked into labeled block: " + out
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

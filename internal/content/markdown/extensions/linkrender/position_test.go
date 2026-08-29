package linkrender_test

import (
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// Links recorded during a real goldmark render carry the 1-based line and
// column of their opening bracket — what `sarde check-links` reports and
// what Studio jumps to.
func TestLinkRefs_CarrySourcePosition(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/beta/", Permalink: "/docs/guide/beta/", Slug: "beta"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/beta.md"},
		PageVersioning:    engine.PageVersioning{VersionRelPath: "guide/beta.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	current := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/alpha/", Permalink: "/docs/guide/alpha/", Slug: "alpha"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/alpha.md"},
		PageVersioning:    engine.PageVersioning{VersionRelPath: "guide/alpha.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	// The body the renderer sees starts after the frontmatter; reported
	// lines are file lines, so this offset is added.
	current.FrontmatterLines = 4
	idx := content.BuildPageIndex([]*engine.Page{target, current})

	r := markdown.NewRendererFromConfig(markdown.RendererConfig{})
	lr := r.LinkRenderer()
	lr.PageIndex = idx
	lr.URLResolver = &engine.URLResolver{BasePath: "/"}
	lr.LinkGraph = links.NewLinkGraph()
	r.SetLinkContext(current)

	src := "# Title\n" + // 1
		"\n" + // 2
		"Intro with [good](./beta.md) inline.\n" + // 3, col 12
		"\n" + // 4
		"- item\n" + // 5
		"- [bare](beta.md) sibling\n" + // 6, col 3
		"\n" + // 7
		"[missing](./nope.md) and [](./beta.md) and [frag](./beta.md#setup)\n" // 8
	if _, err := r.Render(src); err != nil {
		t.Fatal(err)
	}

	refs := lr.DrainRecordedRefs()
	want := []struct {
		dest   string
		line   int
		col    int
		status links.LinkStatus
	}{
		{"./beta.md", 7, 12, links.StatusOK},
		{"beta.md", 10, 3, links.StatusAmbiguous},
		{"./nope.md", 12, 1, links.StatusBrokenTarget},
		{"./beta.md", 12, 26, links.StatusOK}, // empty label: position of `[`
	}
	if len(refs) != len(want) {
		t.Fatalf("recorded %d refs, want %d: %+v", len(refs), len(want), refs)
	}
	for i, w := range want {
		got := refs[i]
		if got.RawDest != w.dest || got.Line != w.line || got.Col != w.col || got.Status != w.status {
			t.Errorf("ref %d: got dest=%q line=%d col=%d status=%d, want %+v", i, got.RawDest, got.Line, got.Col, got.Status, w)
		}
	}

	// The fragment link is deferred to the anchor pass with its position.
	pending := lr.DrainPendingAnchors()
	if len(pending) != 1 || pending[0].Line != 12 || pending[0].Col != 44 {
		t.Fatalf("pending anchor position: %+v", pending)
	}
	graph := links.NewLinkGraph()
	links.ValidateAnchors(graph, pending, idx)
	if got := graph.Refs()[0]; got.Line != 12 || got.Col != 44 {
		t.Errorf("anchor ref position lost: line=%d col=%d", got.Line, got.Col)
	}
}

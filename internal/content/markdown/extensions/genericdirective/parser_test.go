package genericdirective

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/card"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/kbd"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark"
)

// testRegistry loads a registry with one container (pullquote) and one leaf
// (rawbox) directive.
func testRegistry(t *testing.T) *directive.Registry {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"pullquote.yaml": "name: pullquote\nkind: container\nlabel: Pull Quote\ndescription: d\nfields:\n  - { name: author, label: Author, type: string }\n",
		"pullquote.html": `<blockquote class="pullquote" data-label="{{.Label}}" data-author="{{.Attrs.author}}">{{.Body}}</blockquote>`,
		"rawbox.yaml":    "name: rawbox\nkind: leaf\nlabel: Raw Box\ndescription: d\n",
		"rawbox.html":    `<pre class="rawbox">{{.Body}}</pre>`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := directive.NewRegistry(nil)
	if warns := r.LoadDir(dir, "site"); len(warns) != 0 {
		t.Fatalf("unexpected warnings: %+v", warns)
	}
	return r
}

// render builds goldmark the way buildMarkdown does: built-ins first, the
// generic extension last.
func render(t *testing.T, reg *directive.Registry, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(
		&card.Extension{},
		&kbd.Extension{},
		&Extension{Registry: reg},
	))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

func TestParser_ContainerRoundTrip(t *testing.T) {
	out := render(t, testRegistry(t), ":::pullquote[Wise Words] author=\"Ada\"\n\nSome **bold** text.\n\n:::\n")
	for _, want := range []string{
		`class="pullquote"`,
		`data-label="Wise Words"`,
		`data-author="Ada"`,
		"<strong>bold</strong>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestParser_LeafRoundTrip(t *testing.T) {
	out := render(t, testRegistry(t), ":::rawbox\nline one\nline **not markdown**\n:::\n")
	if !strings.Contains(out, `class="rawbox"`) {
		t.Fatalf("rawbox wrapper missing:\n%s", out)
	}
	if !strings.Contains(out, "line one\nline **not markdown**") {
		t.Errorf("raw body not preserved verbatim:\n%s", out)
	}
	if strings.Contains(out, "<strong>") {
		t.Errorf("leaf body must not be markdown-rendered:\n%s", out)
	}
}

func TestParser_NamedClosingFence(t *testing.T) {
	out := render(t, testRegistry(t), ":::pullquote\n\nbody\n\n:::/pullquote\n")
	if !strings.Contains(out, `class="pullquote"`) || !strings.Contains(out, "body") {
		t.Fatalf("named closing fence failed:\n%s", out)
	}
}

func TestParser_GenericInsideBuiltin(t *testing.T) {
	out := render(t, testRegistry(t), ":::card[Outer]\n\n:::pullquote\n\ninner\n\n:::\n\n:::\n")
	if !strings.Contains(out, "sarde-card") {
		t.Fatalf("outer card missing:\n%s", out)
	}
	if !strings.Contains(out, `class="pullquote"`) || !strings.Contains(out, "inner") {
		t.Errorf("nested generic directive missing:\n%s", out)
	}
}

func TestParser_BuiltinInsideGeneric(t *testing.T) {
	out := render(t, testRegistry(t), ":::pullquote\n\n:::card[Inner]\n\nbody\n\n:::\n\n:::\n")
	if !strings.Contains(out, `class="pullquote"`) {
		t.Fatalf("outer pullquote missing:\n%s", out)
	}
	if !strings.Contains(out, "sarde-card") || !strings.Contains(out, "Inner") {
		t.Errorf("nested built-in card missing:\n%s", out)
	}
}

func TestParser_UnregisteredNameFallsThrough(t *testing.T) {
	out := render(t, testRegistry(t), ":::nosuchthing\n\ntext\n\n:::\n")
	if !strings.Contains(out, ":::nosuchthing") {
		t.Errorf("unregistered fence must fall through as text:\n%s", out)
	}
}

func TestParser_EmptyRegistryNoOps(t *testing.T) {
	out := render(t, directive.NewRegistry(nil), ":::pullquote\n\ntext\n\n:::\n")
	if !strings.Contains(out, ":::pullquote") {
		t.Errorf("empty registry must leave fences as text:\n%s", out)
	}
}

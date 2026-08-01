package genericdirective

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark"
)

func TestRenderer_LeafBodyEscaped(t *testing.T) {
	out := render(t, testRegistry(t), ":::rawbox\n<script>alert(1)</script>\n:::\n")
	if strings.Contains(out, "<script>") {
		t.Fatalf("leaf body must be HTML-escaped:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("escaped body missing:\n%s", out)
	}
}

func TestRenderer_ContainerRecursiveDispatch(t *testing.T) {
	// A built-in inline (::kbd[x]) inside the container body proves the body
	// goes through the shared renderer, not a bare fallback.
	out := render(t, testRegistry(t), ":::pullquote\n\nPress ::kbd[Ctrl] now.\n\n:::\n")
	if !strings.Contains(out, "<kbd") || !strings.Contains(out, "Ctrl") {
		t.Fatalf("built-in inline not dispatched inside container body:\n%s", out)
	}
}

func TestRenderer_AttrsEscapedThroughTemplate(t *testing.T) {
	out := render(t, testRegistry(t), ":::pullquote author=\"a<b&c\"\n\nbody\n\n:::\n")
	if strings.Contains(out, `data-author="a<b&c"`) {
		t.Fatalf("attr value must be escaped by html/template:\n%s", out)
	}
	if !strings.Contains(out, "a&lt;b&amp;c") {
		t.Errorf("escaped attr missing:\n%s", out)
	}
}

func TestRenderer_TemplateExecError(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"boom.yaml": "name: boom\nkind: leaf\nlabel: Boom\ndescription: d\n",
		// Parses fine, fails at execute time: TemplateData has no such field.
		"boom.html": `{{.NoSuchField}}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg := directive.NewRegistry(nil)
	if warns := reg.LoadDir(dir, "site"); len(warns) != 0 {
		t.Fatalf("unexpected warnings: %+v", warns)
	}

	gm := goldmark.New(goldmark.WithExtensions(&Extension{Registry: reg}))
	var buf bytes.Buffer
	err := gm.Convert([]byte(":::boom\nx\n:::\n"), &buf)
	if err == nil {
		t.Fatal("expected template execution error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error must name the directive: %v", err)
	}
}

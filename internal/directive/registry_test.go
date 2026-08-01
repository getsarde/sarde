package directive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
}

const pullquoteYAML = `name: pullquote
kind: container
label: "Pull Quote"
description: "A styled pull quote"
bracket:
  label: Title
  required: false
fields:
  - { name: author, label: Author, type: string }
  - { name: variant, label: Variant, type: enum, options: [plain, fancy], default: plain }
`

const pullquoteHTML = `<blockquote class="pullquote pullquote-{{.Attrs.variant}}">{{.Body}}<cite>{{.Attrs.author}}</cite></blockquote>`

const noteboxYAML = `name: notebox
kind: leaf
label: Notebox
description: "Raw text box"
`

const noteboxHTML = `<div class="notebox">{{.Body}}</div>`

func TestRegistry_LoadDir_HappyPaths(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"pullquote.yaml": pullquoteYAML,
		"pullquote.html": pullquoteHTML,
		"pullquote.css":  ".pullquote { color: var(--sd-accent); }",
		"notebox.yaml":   noteboxYAML,
		"notebox.html":   noteboxHTML,
	})

	r := NewRegistry(nil)
	warns := r.LoadDir(dir, "site")
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %+v", warns)
	}

	if r.Empty() {
		t.Fatal("registry should not be empty")
	}
	if got := r.Names(); len(got) != 2 || got[0] != "notebox" || got[1] != "pullquote" {
		t.Fatalf("Names() = %v", got)
	}

	pq := r.Lookup("pullquote")
	if pq == nil {
		t.Fatal("pullquote not registered")
	}
	if pq.Kind != KindContainer || pq.Source != "site" || pq.Category != "custom" {
		t.Errorf("pullquote fields wrong: kind=%q source=%q category=%q", pq.Kind, pq.Source, pq.Category)
	}
	if pq.Bracket == nil || pq.Bracket.Label != "Title" {
		t.Errorf("pullquote bracket not parsed: %+v", pq.Bracket)
	}
	if len(pq.Fields) != 2 || pq.Fields[0].Placement != "attr" || pq.Fields[1].Placement != "attr" {
		t.Errorf("pullquote fields not stamped attr: %+v", pq.Fields)
	}
	if len(pq.CSS) == 0 {
		t.Error("pullquote CSS sidecar not loaded")
	}

	nb := r.Lookup("notebox")
	if nb == nil || nb.Kind != KindLeaf {
		t.Fatalf("notebox not registered as leaf: %+v", nb)
	}
	if len(nb.CSS) != 0 {
		t.Error("notebox should have no CSS")
	}
}

func TestRegistry_LoadDir_MissingDir(t *testing.T) {
	r := NewRegistry(nil)
	if warns := r.LoadDir(filepath.Join(t.TempDir(), "nope"), "site"); warns != nil {
		t.Fatalf("missing dir should be silent, got %+v", warns)
	}
	if !r.Empty() {
		t.Fatal("registry should be empty")
	}
}

func TestRegistry_LoadDir_WarnAndSkip(t *testing.T) {
	cases := []struct {
		name    string
		files   map[string]string
		wantMsg string
	}{
		{
			name:    "missing sibling html",
			files:   map[string]string{"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\n"},
			wantMsg: "missing sibling template",
		},
		{
			name: "name stem mismatch",
			files: map[string]string{
				"a.yaml": "name: b\nkind: leaf\nlabel: A\ndescription: d\n",
				"a.html": "<div></div>",
			},
			wantMsg: "does not match filename stem",
		},
		{
			name: "bad kind",
			files: map[string]string{
				"a.yaml": "name: a\nkind: inline\nlabel: A\ndescription: d\n",
				"a.html": "<div></div>",
			},
			wantMsg: "invalid kind",
		},
		{
			name: "bad field type",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\nfields:\n  - { name: x, label: X, type: color }\n",
				"a.html": "<div></div>",
			},
			wantMsg: "invalid type",
		},
		{
			name: "enum without options",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\nfields:\n  - { name: x, label: X, type: enum }\n",
				"a.html": "<div></div>",
			},
			wantMsg: "no options",
		},
		{
			name: "duplicate field names",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\nfields:\n  - { name: x, label: X, type: string }\n  - { name: x, label: X2, type: string }\n",
				"a.html": "<div></div>",
			},
			wantMsg: "duplicate field name",
		},
		{
			name: "unknown yaml key",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\nbogus: true\n",
				"a.html": "<div></div>",
			},
			wantMsg: "bogus",
		},
		{
			name: "template parse error",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\n",
				"a.html": "<div>{{.Broken</div>",
			},
			wantMsg: "parsing template",
		},
		{
			name: "invalid name",
			files: map[string]string{
				"My-Thing.yaml": "name: My-Thing\nkind: leaf\nlabel: A\ndescription: d\n",
				"My-Thing.html": "<div></div>",
			},
			wantMsg: "invalid name",
		},
		{
			name: "missing label",
			files: map[string]string{
				"a.yaml": "name: a\nkind: leaf\ndescription: d\n",
				"a.html": "<div></div>",
			},
			wantMsg: `missing required key "label"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)
			r := NewRegistry(nil)
			warns := r.LoadDir(dir, "site")
			if len(warns) != 1 {
				t.Fatalf("want 1 warning, got %d: %+v", len(warns), warns)
			}
			w := warns[0]
			if w.Field != "directive" || w.Level != "warning" {
				t.Errorf("warning shape wrong: %+v", w)
			}
			if !strings.Contains(w.Message, tc.wantMsg) {
				t.Errorf("warning %q does not contain %q", w.Message, tc.wantMsg)
			}
			if !r.Empty() {
				t.Errorf("bad definition must be skipped, registry has %v", r.Names())
			}
		})
	}
}

func TestRegistry_LoadDir_OverlayOrder(t *testing.T) {
	themeDir := t.TempDir()
	siteDir := t.TempDir()
	writeFiles(t, themeDir, map[string]string{
		"pullquote.yaml": pullquoteYAML,
		"pullquote.html": `<div class="theme"></div>`,
	})
	writeFiles(t, siteDir, map[string]string{
		"pullquote.yaml": pullquoteYAML,
		"pullquote.html": `<div class="site"></div>`,
	})

	r := NewRegistry(nil)
	r.LoadDir(themeDir, "theme")
	r.LoadDir(siteDir, "site")

	def := r.Lookup("pullquote")
	if def == nil || def.Source != "site" {
		t.Fatalf("site definition must overwrite theme: %+v", def)
	}
	if string(def.htmlRaw) != `<div class="site"></div>` {
		t.Errorf("template not overwritten: %s", def.htmlRaw)
	}
}

func TestRegistry_Hash_Sensitivity(t *testing.T) {
	base := map[string]string{
		"a.yaml": "name: a\nkind: leaf\nlabel: A\ndescription: d\n",
		"a.html": "<div>{{.Body}}</div>",
		"a.css":  ".a { color: red; }",
	}
	load := func(files map[string]string) string {
		dir := t.TempDir()
		writeFiles(t, dir, files)
		r := NewRegistry(nil)
		if warns := r.LoadDir(dir, "site"); len(warns) != 0 {
			t.Fatalf("unexpected warnings: %+v", warns)
		}
		return r.Hash()
	}

	h0 := load(base)
	if h0 == "" {
		t.Fatal("hash of non-empty registry must not be empty")
	}

	mutations := map[string]map[string]string{
		"yaml": {"a.yaml": "name: a\nkind: leaf\nlabel: A2\ndescription: d\n", "a.html": base["a.html"], "a.css": base["a.css"]},
		"html": {"a.yaml": base["a.yaml"], "a.html": "<span>{{.Body}}</span>", "a.css": base["a.css"]},
		"css":  {"a.yaml": base["a.yaml"], "a.html": base["a.html"], "a.css": ".a { color: blue; }"},
	}
	for part, files := range mutations {
		if h := load(files); h == h0 {
			t.Errorf("hash did not change when %s changed", part)
		}
	}

	if NewRegistry(nil).Hash() != "" {
		t.Error("empty registry hash must be empty string")
	}
}

func TestRegistry_CSS_Ordering(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"zeta.yaml":  "name: zeta\nkind: leaf\nlabel: Z\ndescription: d\n",
		"zeta.html":  "<div></div>",
		"zeta.css":   ".zeta {}",
		"alpha.yaml": "name: alpha\nkind: leaf\nlabel: A\ndescription: d\n",
		"alpha.html": "<div></div>",
		"alpha.css":  ".alpha {}",
	})
	r := NewRegistry(nil)
	r.LoadDir(dir, "site")

	css := r.CSS()
	ia := strings.Index(css, "/* directive: alpha */")
	iz := strings.Index(css, "/* directive: zeta */")
	if ia < 0 || iz < 0 || ia > iz {
		t.Fatalf("CSS not name-ordered with headers:\n%s", css)
	}
	if !strings.Contains(css, ".alpha {}") || !strings.Contains(css, ".zeta {}") {
		t.Fatalf("CSS bodies missing:\n%s", css)
	}
}

func TestRegistry_ValidateAgainstBuiltins(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"card.yaml":      "name: card\nkind: container\nlabel: Card\ndescription: d\n",
		"card.html":      "<div></div>",
		"pullquote.yaml": pullquoteYAML,
		"pullquote.html": pullquoteHTML,
	})
	r := NewRegistry(nil)
	if warns := r.LoadDir(dir, "site"); len(warns) != 0 {
		t.Fatalf("unexpected load warnings: %+v", warns)
	}

	cat := &engine.DirectiveCatalog{Categories: []engine.DirectiveCategory{
		{Name: "content", Label: "Content", Directives: []engine.CatalogDirective{{Name: "card"}}},
	}}
	warns := r.ValidateAgainstBuiltins(cat)
	if len(warns) != 1 || !strings.Contains(warns[0].Message, "card") {
		t.Fatalf("want one collision warning for card, got %+v", warns)
	}
	if r.Lookup("card") != nil {
		t.Error("colliding directive must be removed")
	}
	if r.Lookup("pullquote") == nil {
		t.Error("non-colliding directive must survive")
	}
}

func TestMergeCatalog(t *testing.T) {
	base := &engine.DirectiveCatalog{Categories: []engine.DirectiveCategory{
		{Name: "content", Label: "Content", Directives: []engine.CatalogDirective{{Name: "card", Label: "Card"}}},
	}}

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"pullquote.yaml": pullquoteYAML,
		"pullquote.html": pullquoteHTML,
	})
	r := NewRegistry(nil)
	r.LoadDir(dir, "site")

	merged := MergeCatalog(base, r)

	if base.Categories[0].Directives[0].Source != "" {
		t.Error("MergeCatalog must not mutate base")
	}
	if len(merged.Categories) != 2 {
		t.Fatalf("want 2 categories, got %d", len(merged.Categories))
	}
	if merged.Categories[0].Directives[0].Source != "builtin" {
		t.Errorf("builtin source not stamped: %+v", merged.Categories[0].Directives[0])
	}
	custom := merged.Categories[1]
	if custom.Name != "custom" || len(custom.Directives) != 1 {
		t.Fatalf("custom category wrong: %+v", custom)
	}
	d := custom.Directives[0]
	if d.Name != "pullquote" || d.Source != "site" || d.Kind != "block" {
		t.Errorf("merged entry wrong: %+v", d)
	}
}

func TestMergeCatalog_EmptyRegistry(t *testing.T) {
	base := &engine.DirectiveCatalog{Categories: []engine.DirectiveCategory{
		{Name: "content", Label: "Content", Directives: []engine.CatalogDirective{{Name: "card"}}},
	}}
	merged := MergeCatalog(base, NewRegistry(nil), nil)
	if len(merged.Categories) != 1 {
		t.Fatalf("empty registries must not add a custom category: %+v", merged.Categories)
	}
}

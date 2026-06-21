package icons

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Phase 3 — <symbol> sprite render mode
// ---------------------------------------------------------------------------

// TestRenderSpriteMode verifies the sprite path emits a <use> reference wrapper
// (not an inlined body) while keeping class + decorative ARIA from the inline
// path.
func TestRenderSpriteMode(t *testing.T) {
	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	svg := Render("rocket", "sarde-icon", nil)
	if !strings.Contains(svg, `<use href="#i-lucide-rocket"></use>`) {
		t.Errorf("sprite Render missing <use> ref: %q", svg)
	}
	if !strings.Contains(svg, `class="sarde-icon"`) {
		t.Errorf("sprite Render dropped class: %q", svg)
	}
	if !strings.Contains(svg, `aria-hidden="true"`) || !strings.Contains(svg, `focusable="false"`) {
		t.Errorf("decorative sprite icon should be aria-hidden + focusable=false: %q", svg)
	}
	// The icon geometry must live in the <symbol>, never inlined in the wrapper.
	if strings.Contains(svg, "<path") {
		t.Errorf("sprite wrapper must not inline the icon body: %q", svg)
	}
}

// TestRenderInlineModeUnchanged guards the refactor: inline mode still inlines
// the body and emits no <use>.
func TestRenderInlineModeUnchanged(t *testing.T) {
	SetRenderMode("inline")
	svg := Render("rocket", "sarde-icon", nil)
	if strings.Contains(svg, "<use ") {
		t.Errorf("inline mode must not emit <use>: %q", svg)
	}
	if !strings.Contains(svg, "<path") {
		t.Errorf("inline mode should inline the icon body: %q", svg)
	}
}

// TestRenderSpriteAria checks the G2 requirement: the <use> wrapper runs through
// the same ARIA decision as inline Render (role=img + <title> when named; the
// <title> precedes the <use>).
func TestRenderSpriteAria(t *testing.T) {
	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	svg := Render("rocket", "", map[string]string{"aria-label": "Launch"})
	if !strings.Contains(svg, `role="img"`) || !strings.Contains(svg, `aria-label="Launch"`) {
		t.Errorf("labeled sprite icon should be role=img with aria-label: %q", svg)
	}
	if strings.Contains(svg, "aria-hidden") {
		t.Errorf("labeled sprite icon must not be aria-hidden: %q", svg)
	}

	svg = Render("rocket", "", map[string]string{"title": "Go"})
	ti := strings.Index(svg, "<title>Go</title>")
	ui := strings.Index(svg, "<use ")
	if ti == -1 || ui == -1 || ti > ui {
		t.Errorf("title must precede <use> in the sprite wrapper: %q", svg)
	}
	if !strings.Contains(svg, `role="img"`) {
		t.Errorf("titled sprite icon should be role=img: %q", svg)
	}
}

// TestSpriteIDScheme covers the element-id construction across sources:
// bare/explicit set, Tabler, Lucide-shorthand canonicalization, and the
// circle-help miss fallback.
func TestSpriteIDScheme(t *testing.T) {
	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	cases := []struct{ name, wantID string }{
		{"rocket", "i-lucide-rocket"},
		{"lucide:rocket", "i-lucide-rocket"},
		{"tabler:brand-github", "i-tabler-brand-github"},
		{"warning", "i-lucide-alert-triangle"},          // Lucide shorthand canonicalizes
		{"definitely-not-an-icon", "i-lucide-circle-help"}, // miss -> fallback icon
	}
	for _, c := range cases {
		svg := Render(c.name, "", nil)
		if !strings.Contains(svg, `<use href="#`+c.wantID+`">`) {
			t.Errorf("%s: want id %s, got %q", c.name, c.wantID, svg)
		}
	}
}

// TestSpriteLocalIDScheme checks local-dir icons get an "i-local-{name}" id.
func TestSpriteLocalIDScheme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "brandmark.svg", `<svg viewBox="0 0 200 40"><rect width="200" height="40"/></svg>`)
	if err := LoadIconDirectory(dir); err != nil {
		t.Fatalf("LoadIconDirectory: %v", err)
	}
	defer func() {
		spriteSymbols.Delete("i-local-brandmark")
	}()

	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	svg := Render("brandmark", "", map[string]string{"width": "100"})
	if !strings.Contains(svg, `<use href="#i-local-brandmark">`) {
		t.Errorf("local icon should use i-local- id: %q", svg)
	}
}

// TestSpriteForHTMLDedupAndOrder verifies SpriteForHTML emits one <symbol> per
// unique referenced icon, in sorted order, skips unregistered ids, and wraps
// the symbols in the hidden sprite container.
func TestSpriteForHTMLDedupAndOrder(t *testing.T) {
	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	_ = Render("rocket", "", nil)
	_ = Render("rocket", "", nil)
	_ = Render("search", "", nil)

	page := []byte(`<html><body>` +
		`<svg><use href="#i-lucide-rocket"></use></svg>` +
		`<svg><use href="#i-lucide-rocket"></use></svg>` +
		`<svg><use href="#i-lucide-search"></use></svg>` +
		`<a href="#i-not-registered">anchor</a>` +
		`</body></html>`)

	sprite := SpriteForHTML(page)
	if sprite == nil {
		t.Fatal("SpriteForHTML returned nil for a page with sprite refs")
	}
	s := string(sprite)
	if n := strings.Count(s, "<symbol "); n != 2 {
		t.Errorf("want 2 deduped <symbol>s, got %d: %q", n, s)
	}
	if strings.Contains(s, "i-not-registered") {
		t.Errorf("unregistered id (author anchor) should be skipped: %q", s)
	}
	if strings.Index(s, "i-lucide-rocket") > strings.Index(s, "i-lucide-search") {
		t.Errorf("symbols must be in sorted order: %q", s)
	}
	if !strings.Contains(s, `position:absolute;width:0;height:0`) {
		t.Errorf("sprite container missing hidden styling: %q", s)
	}
}

// TestSpriteForHTMLEmptyAndOff verifies SpriteForHTML returns nil when sprite
// mode is off or the page references no registered sprite icons.
func TestSpriteForHTMLEmptyAndOff(t *testing.T) {
	SetRenderMode("sprite")
	defer SetRenderMode("inline")
	if got := SpriteForHTML([]byte(`<body><p>no icons here</p></body>`)); got != nil {
		t.Errorf("no sprite refs should yield nil, got %q", got)
	}

	SetRenderMode("inline")
	if got := SpriteForHTML([]byte(`<body><use href="#i-lucide-rocket"></use></body>`)); got != nil {
		t.Errorf("inline mode should yield nil sprite, got %q", got)
	}
}

// TestSpriteSymbolReplaceIDs proves internal ids in a symbol body are namespaced
// with the idBase (via replaceIDs) so repeated symbols can't collide.
func TestSpriteSymbolReplaceIDs(t *testing.T) {
	const data = `{"prefix":"zzsprite","width":24,"height":24,"icons":{"grad":{"body":"<defs><linearGradient id=\"g\"/></defs><rect fill=\"url(#g)\"/>"}}}`
	if err := LoadCollection([]byte(data)); err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}
	defer func() {
		spriteSymbols.Delete("i-zzsprite-grad")
	}()

	SetRenderMode("sprite")
	defer SetRenderMode("inline")

	_ = Render("zzsprite:grad", "", nil)
	s := string(SpriteForHTML([]byte(`<body><svg><use href="#i-zzsprite-grad"></use></svg></body>`)))
	if !strings.Contains(s, `id="g-zzsprite-grad"`) || !strings.Contains(s, `url(#g-zzsprite-grad)`) {
		t.Errorf("symbol body internal ids not namespaced with idBase: %q", s)
	}
	if !strings.Contains(s, `<symbol id="i-zzsprite-grad" viewBox="0 0 24 24">`) {
		t.Errorf("symbol opening tag malformed: %q", s)
	}
}

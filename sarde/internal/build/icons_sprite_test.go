package build

import (
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/content/markdown/icons"
)

// TestInjectBeforeBodyClose verifies the sprite splice lands before the final
// </body> and is a no-op when </body> is absent.
func TestInjectBeforeBodyClose(t *testing.T) {
	out := string(injectBeforeBodyClose([]byte(`<html><body><p>hi</p></body></html>`), []byte("SPRITE")))
	if out != `<html><body><p>hi</p>SPRITE</body></html>` {
		t.Errorf("injection wrong: %q", out)
	}
	// Splices before the LAST </body>.
	out = string(injectBeforeBodyClose([]byte(`<body>a</body><body>b</body>`), []byte("S")))
	if out != `<body>a</body><body>bS</body>` {
		t.Errorf("must inject before the last </body>: %q", out)
	}
	// No </body>: returned unchanged (e.g. redirect stubs).
	if out = string(injectBeforeBodyClose([]byte(`<div>no body</div>`), []byte("S"))); out != `<div>no body</div>` {
		t.Errorf("no </body> should be unchanged: %q", out)
	}
}

// TestSpriteInjectionEndToEnd mirrors renderPage's sprite step: in sprite mode,
// scan finished HTML for <use> refs and inject the matching <symbol> sprite
// before </body> (one <symbol> per unique icon); inline mode injects nothing.
func TestSpriteInjectionEndToEnd(t *testing.T) {
	icons.SetRenderMode("sprite")
	defer icons.SetRenderMode("inline")

	a := icons.Render("rocket", "sarde-icon", nil)
	b := icons.Render("rocket", "sarde-icon", nil) // repeat -> still one symbol
	page := []byte(`<html><body>` + a + b + `</body></html>`)

	if sprite := icons.SpriteForHTML(page); sprite != nil {
		page = injectBeforeBodyClose(page, sprite)
	}
	s := string(page)
	if n := strings.Count(s, "<symbol "); n != 1 {
		t.Errorf("repeated icon should yield exactly one <symbol>, got %d: %q", n, s)
	}
	if n := strings.Count(s, "<use "); n != 2 {
		t.Errorf("two <use> refs expected, got %d: %q", n, s)
	}
	if strings.Index(s, "<symbol ") > strings.Index(s, "</body>") {
		t.Errorf("sprite must be injected before </body>: %q", s)
	}

	// Inline mode: no sprite.
	icons.SetRenderMode("inline")
	page = []byte(`<html><body>` + icons.Render("rocket", "", nil) + `</body></html>`)
	if sprite := icons.SpriteForHTML(page); sprite != nil {
		t.Errorf("inline mode should not produce a sprite, got %q", sprite)
	}
}

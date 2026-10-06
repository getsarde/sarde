package template

import (
	htmltemplate "html/template"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/webfonts"
)

// webFontRoute is a route whose site config sets theme.web_fonts and whose
// resolved tokens name the given sans and mono fonts.
func webFontRoute(provider, sans, mono string) *engine.RouteData {
	cfg := config.Defaults()
	cfg.Theme.WebFonts = provider
	return &engine.RouteData{
		Site:  &engine.SiteContext{Config: cfg},
		Theme: &engine.ThemeConfig{Tokens: map[string]string{"font-sans": sans, "font-mono": mono}},
	}
}

func themeStylesOf(t *testing.T, rd *engine.RouteData) string {
	t.Helper()
	fn := nilFuncMap()["themeStyles"].(func(any) htmltemplate.HTML)
	return string(fn(rd))
}

func TestThemeStyles_WebFontsFromGoogle(t *testing.T) {
	out := themeStylesOf(t, webFontRoute("google", "'Plus Jakarta Sans', system-ui, sans-serif", "'Fira Code', monospace"))
	for _, want := range []string{
		`<link rel="preconnect" href="https://fonts.googleapis.com">`,
		`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`,
		`href="https://fonts.googleapis.com/css?family=Plus+Jakarta+Sans:` + webfonts.Weights +
			`|Fira+Code:` + webfonts.Weights + `&amp;display=swap"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("themeStyles missing %s\n%s", want, out)
		}
	}
}

func TestThemeStyles_WebFontsFromBunny(t *testing.T) {
	out := themeStylesOf(t, webFontRoute("bunny", "Merriweather, serif", "'JetBrains Mono', monospace"))
	if !strings.Contains(out, `href="https://fonts.bunny.net/css?family=Merriweather:`+webfonts.Weights+`&amp;display=swap"`) {
		t.Errorf("no Bunny stylesheet for Merriweather:\n%s", out)
	}
	if strings.Contains(out, "JetBrains") {
		t.Errorf("a bundled font was requested:\n%s", out)
	}
}

func TestThemeStyles_NoWebFontLinksWhenOffOrNothingToLoad(t *testing.T) {
	for name, rd := range map[string]*engine.RouteData{
		"setting off":     webFontRoute("", "'Plus Jakarta Sans', sans-serif", ""),
		"bundled only":    webFontRoute("google", "'Inter', system-ui, sans-serif", "'JetBrains Mono', monospace"),
		"system stack":    webFontRoute("bunny", "system-ui, -apple-system, sans-serif", ""),
		"no site config":  {Theme: &engine.ThemeConfig{Tokens: map[string]string{"font-sans": "Lora"}}},
		"no theme tokens": {Site: &engine.SiteContext{Config: config.Defaults()}},
	} {
		if out := themeStylesOf(t, rd); strings.Contains(out, "fonts.") {
			t.Errorf("%s: unexpected web font links:\n%s", name, out)
		}
	}
}

func TestRenderHeadTags_AcceptsConfigAndCascadeShapes(t *testing.T) {
	fromConfig := fnRenderHeadTags([]config.HeadTag{
		{Tag: "meta", Attrs: map[string]string{"name": "google-site-verification", "content": "abc"}},
	})
	if !strings.Contains(string(fromConfig), `<meta content="abc" name="google-site-verification">`) {
		t.Errorf("config tags not rendered: %q", fromConfig)
	}

	// A section cascade hands over the decoded YAML as-is.
	fromCascade := fnRenderHeadTags([]any{
		map[string]any{"tag": "Link", "attrs": map[string]any{"rel": "me", "href": "https://example.social/@a"}},
		map[string]any{"tag": "iframe", "attrs": map[string]any{"src": "x"}}, // not allowed in head
		"not a map",
	})
	got := string(fromCascade)
	if !strings.Contains(got, `<link href="https://example.social/@a" rel="me">`) {
		t.Errorf("cascade tags not rendered: %q", got)
	}
	if strings.Contains(got, "iframe") {
		t.Errorf("disallowed tag rendered: %q", got)
	}
}

func TestRenderHeadTags_RawTextContentIsNotEscaped(t *testing.T) {
	got := string(fnRenderHeadTags([]engine.HeadTag{
		{Tag: "style", Content: `@font-face { font-family: "Brand"; src: url('/brand.woff2'); }`},
		{Tag: "script", Content: `if (a && b < c) { go("x") }`},
	}))
	for _, want := range []string{
		`<style>@font-face { font-family: "Brand"; src: url('/brand.woff2'); }</style>`,
		`<script>if (a && b < c) { go("x") }</script>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("raw content mangled; want %s in\n%s", want, got)
		}
	}
}

func TestRenderHeadTags_ClosingTagInsideContentCannotEndTheElement(t *testing.T) {
	got := string(fnRenderHeadTags([]engine.HeadTag{
		{Tag: "script", Content: `var s = "</SCRIPT><img src=x onerror=alert(1)>";`},
	}))
	if strings.Count(strings.ToLower(got), "</script") != 1 {
		t.Fatalf("content closed the script element early:\n%s", got)
	}
	if !strings.Contains(got, `<\/SCRIPT>`) {
		t.Errorf("closing tag not neutralized:\n%s", got)
	}
}

func TestSiteHeadTags(t *testing.T) {
	fn := nilFuncMap()["siteHeadTags"].(func(any) htmltemplate.HTML)
	cfg := config.Defaults()
	cfg.Head.Tags = []config.HeadTag{{Tag: "meta", Attrs: map[string]string{"name": "x-site", "content": "1"}}}
	rd := &engine.RouteData{Site: &engine.SiteContext{Config: cfg}}
	if got := string(fn(rd)); !strings.Contains(got, `<meta content="1" name="x-site">`) {
		t.Errorf("site head tags not rendered: %q", got)
	}
	if got := fn(&engine.RouteData{}); got != "" {
		t.Errorf("no config: %q, want empty", got)
	}
	if got := fn(nil); got != "" {
		t.Errorf("nil route: %q, want empty", got)
	}
}

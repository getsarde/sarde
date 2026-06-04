package icons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// iconToSVG — transform baking + viewBox swap
// ---------------------------------------------------------------------------

func TestIconToSVGNoTransform(t *testing.T) {
	body, vb := iconToSVG(resolvedIcon{Body: "X", Width: 24, Height: 24})
	if body != "X" {
		t.Errorf("no-transform body should be unchanged, got %q", body)
	}
	if vb != "0 0 24 24" {
		t.Errorf("viewBox = %q, want 0 0 24 24", vb)
	}
	if strings.Contains(body, "<g") {
		t.Errorf("no-transform icon must not be wrapped: %q", body)
	}
}

func TestIconToSVGTransforms(t *testing.T) {
	tests := []struct {
		name     string
		in       resolvedIcon
		wantBody string
		wantVB   string
	}{
		{"hflip", resolvedIcon{Body: "X", Width: 24, Height: 24, HFlip: true},
			`<g transform="translate(24 0) scale(-1 1)">X</g>`, "0 0 24 24"},
		{"vflip", resolvedIcon{Body: "X", Width: 24, Height: 24, VFlip: true},
			`<g transform="translate(0 24) scale(1 -1)">X</g>`, "0 0 24 24"},
		{"rotate90-square", resolvedIcon{Body: "X", Width: 24, Height: 24, Rotate: 1},
			`<g transform="rotate(90 12 12)">X</g>`, "0 0 24 24"},
		{"rotate90-nonsquare", resolvedIcon{Body: "X", Width: 16, Height: 24, Rotate: 1},
			`<g transform="rotate(90 12 12)">X</g>`, "0 0 24 16"},
		{"rotate180", resolvedIcon{Body: "X", Width: 24, Height: 24, Rotate: 2},
			`<g transform="rotate(180 12 12)">X</g>`, "0 0 24 24"},
		{"rotate270-nonsquare", resolvedIcon{Body: "X", Width: 20, Height: 30, Rotate: 3},
			`<g transform="rotate(-90 10 10)">X</g>`, "0 0 30 20"},
		{"hflip+vflip->180", resolvedIcon{Body: "X", Width: 24, Height: 24, HFlip: true, VFlip: true},
			`<g transform="rotate(180 12 12)">X</g>`, "0 0 24 24"},
	}
	for _, tt := range tests {
		body, vb := iconToSVG(tt.in)
		if body != tt.wantBody {
			t.Errorf("%s body = %q, want %q", tt.name, body, tt.wantBody)
		}
		if vb != tt.wantVB {
			t.Errorf("%s viewBox = %q, want %q", tt.name, vb, tt.wantVB)
		}
	}
}

// ---------------------------------------------------------------------------
// resolveIcon — alias chain transform merge + dim/body override + root defaults
// ---------------------------------------------------------------------------

func testCollection() *IconifyCollection {
	return &IconifyCollection{
		Prefix: "zz", Width: 24, Height: 24,
		Icons: map[string]IconifyIcon{
			"base": {Body: "BASE"},
			"rot1": {Body: "R1", Rotate: 1},
			"star": {Body: "STAR"},
		},
		Aliases: map[string]IconifyIconAlias{
			"rotA":      {Parent: "base", Rotate: 1},
			"rotB":      {Parent: "rotA", Rotate: 1}, // chain rotB->rotA->base = rotate 2
			"flipA":     {Parent: "base", HFlip: true},
			"flipB":     {Parent: "flipA", HFlip: true}, // XOR cancels -> false
			"bodyA":     {Parent: "base", Body: strPtr("OVERRIDE")},
			"dimA":      {Parent: "base", Width: intPtr(40), Height: intPtr(40)},
			"rotOnIcon": {Parent: "rot1", Rotate: 1}, // icon rotate 1 + alias rotate 1 = 2
		},
	}
}

func TestResolveIconAliasMerge(t *testing.T) {
	col := testCollection()
	tests := []struct {
		name        string
		want        resolvedIcon
		ignoreField string
	}{
		{name: "rotate-additive-chain", want: resolvedIcon{Body: "BASE", Width: 24, Height: 24, Rotate: 2}},
		{name: "hflip-single", want: resolvedIcon{Body: "BASE", Width: 24, Height: 24, HFlip: true}},
		{name: "hflip-xor-cancel", want: resolvedIcon{Body: "BASE", Width: 24, Height: 24}},
		{name: "body-override", want: resolvedIcon{Body: "OVERRIDE", Width: 24, Height: 24}},
		{name: "dim-override", want: resolvedIcon{Body: "BASE", Width: 40, Height: 40}},
		{name: "rotate-on-icon-plus-alias", want: resolvedIcon{Body: "R1", Width: 24, Height: 24, Rotate: 2}},
	}
	names := map[string]string{
		"rotate-additive-chain":     "rotB",
		"hflip-single":              "flipA",
		"hflip-xor-cancel":          "flipB",
		"body-override":             "bodyA",
		"dim-override":              "dimA",
		"rotate-on-icon-plus-alias": "rotOnIcon",
	}
	for _, tt := range tests {
		got, ok := resolveIcon(col, names[tt.name])
		if !ok {
			t.Errorf("%s: resolveIcon failed", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: resolveIcon = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestResolveIconRootDefaults(t *testing.T) {
	col := testCollection()
	got, ok := resolveIcon(col, "base")
	if !ok || got.Width != 24 || got.Height != 24 {
		t.Errorf("root-default dims: got %+v ok=%v, want 24x24", got, ok)
	}
	if _, ok := resolveIcon(col, "no-such-icon"); ok {
		t.Error("resolveIcon should fail for an unknown name")
	}
}

// ---------------------------------------------------------------------------
// calculateSize — aspect-ratio / sentinels
// ---------------------------------------------------------------------------

func TestCalculateSize(t *testing.T) {
	tests := []struct {
		name         string
		cw, ch       string
		bw, bh       int
		wantW, wantH string
	}{
		{"both-empty", "", "", 24, 24, "16", "16"},
		{"width-only-square", "32", "", 24, 24, "32", "32"},
		{"width-only-nonsquare", "32", "", 20, 30, "32", "48"},
		{"height-only-nonsquare", "", "48", 20, 30, "32", "48"},
		{"both-supplied", "100", "50", 24, 24, "100", "50"},
		{"unset-both", "unset", "unset", 24, 24, "", ""},
		{"auto-width", "auto", "", 20, 30, "20", "30"},
		{"em-unit", "1em", "", 20, 30, "1em", "1.5em"},
	}
	for _, tt := range tests {
		gotW, gotH := calculateSize(tt.cw, tt.ch, tt.bw, tt.bh)
		if gotW != tt.wantW || gotH != tt.wantH {
			t.Errorf("%s: calculateSize(%q,%q,%d,%d) = (%q,%q), want (%q,%q)",
				tt.name, tt.cw, tt.ch, tt.bw, tt.bh, gotW, gotH, tt.wantW, tt.wantH)
		}
	}
}

// ---------------------------------------------------------------------------
// replaceIDs
// ---------------------------------------------------------------------------

func TestReplaceIDs(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		suffix string
		want   string
	}{
		{"id+url", `<mask id="a"></mask><path fill="url(#a)"/>`, "x",
			`<mask id="a-x"></mask><path fill="url(#a-x)"/>`},
		{"href", `<use href="#a"/>`, "x", `<use href="#a-x"/>`},
		{"xlink-href-single-suffix", `<use xlink:href="#a"/>`, "x", `<use xlink:href="#a-x"/>`},
		{"no-ids", `<path d="M0 0"/>`, "x", `<path d="M0 0"/>`},
		{"empty-suffix", `<mask id="a"/>`, "", `<mask id="a"/>`},
	}
	for _, tt := range tests {
		if got := replaceIDs(tt.in, tt.suffix); got != tt.want {
			t.Errorf("%s: replaceIDs = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Multi-set resolution + SetDefaultPrefix + aliasMap gating
// ---------------------------------------------------------------------------

func TestMultiSetResolution(t *testing.T) {
	const data = `{"prefix":"zztest","width":24,"height":24,"icons":{"star":{"body":"<path d=\"ZZ\"/>"}}}`
	if err := LoadCollection([]byte(data)); err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}
	defer func() {
		resolver.mu.Lock()
		delete(resolver.collections, "zztest")
		resolver.mu.Unlock()
	}()

	if svg := GetWithClass("zztest:star", "c"); !strings.Contains(svg, "ZZ") {
		t.Errorf("zztest:star did not resolve: %q", svg)
	}
	// Default prefix unchanged: a bare lucide icon still resolves from lucide.
	if svg := GetWithClass("rocket", ""); svg == "" {
		t.Error("bare 'rocket' should still resolve from lucide default")
	}

	// Switch default prefix to the test set; bare names resolve there.
	SetDefaultPrefix("zztest")
	defer SetDefaultPrefix("lucide")
	if svg := GetWithClass("star", "c"); !strings.Contains(svg, "ZZ") {
		t.Errorf("bare 'star' under default zztest did not resolve: %q", svg)
	}
	// aliasMap is Lucide-only: "warning" must NOT map to alert-triangle here.
	if svg := GetWithClass("warning", ""); svg != "" {
		t.Errorf("aliasMap should be gated off for non-lucide default, got %q", svg)
	}
}

func TestTablerResolves(t *testing.T) {
	svg := GetWithClass("tabler:brand-github", "c")
	if svg == "" || !strings.Contains(svg, `viewBox="0 0 24 24"`) {
		t.Errorf("tabler:brand-github did not resolve correctly: %q", svg)
	}
}

func TestSimpleIconsResolves(t *testing.T) {
	svg := GetWithClass("simple-icons:github", "c")
	if svg == "" || !strings.Contains(svg, `viewBox="0 0 24 24"`) {
		t.Errorf("simple-icons:github did not resolve correctly: %q", svg)
	}
	// Simple Icons are fill glyphs that bake fill="currentColor" so they inherit
	// the surrounding text color (not a hardcoded black fill).
	if !strings.Contains(svg, `fill="currentColor"`) {
		t.Errorf("simple-icons:github should inherit currentColor, got: %q", svg)
	}
	// The "brands" set alias routes to the same simple-icons collection, so it
	// resolves to a byte-identical SVG.
	if alias := GetWithClass("brands:github", "c"); alias != svg {
		t.Errorf("brands:github should equal simple-icons:github (alias):\n brands = %q\n simple = %q", alias, svg)
	}
}

func TestLoadedSetLicenses(t *testing.T) {
	got := map[string]string{}
	for _, s := range LoadedSetLicenses() {
		got[s.Prefix] = s.SPDX
	}
	if got["lucide"] != "ISC" {
		t.Errorf("lucide license = %q, want ISC", got["lucide"])
	}
	if got["tabler"] != "MIT" {
		t.Errorf("tabler license = %q, want MIT", got["tabler"])
	}
	if got["simple-icons"] != "CC0-1.0" {
		t.Errorf("simple-icons license = %q, want CC0-1.0", got["simple-icons"])
	}
}

// ---------------------------------------------------------------------------
// Local icons/ directory provider
// ---------------------------------------------------------------------------

func TestLocalIconDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rocket.svg", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><path d="LOCALROCKET"/></svg>`)
	writeFile(t, dir, "logo.svg", `<svg viewBox="0 0 200 40"><rect width="200" height="40"/></svg>`)

	if err := LoadIconDirectory(dir); err != nil {
		t.Fatalf("LoadIconDirectory: %v", err)
	}
	defer func() {
		resolver.mu.Lock()
		delete(resolver.localIcons, "rocket")
		delete(resolver.localIcons, "logo")
		resolver.mu.Unlock()
	}()

	// Local "rocket" shadows the lucide set icon of the same name.
	if svg := Get("rocket"); !strings.Contains(svg, "LOCALROCKET") || !strings.Contains(svg, `viewBox="0 0 32 32"`) {
		t.Errorf("local rocket not used: %q", svg)
	}
	// Explicit set prefix bypasses the local dir.
	if svg := GetWithClass("lucide:rocket", ""); strings.Contains(svg, "LOCALROCKET") {
		t.Errorf("lucide:rocket must not use the local override: %q", svg)
	}
	// Non-square viewBox preserved; only-width derives height by aspect ratio.
	svg := Render("logo", "", map[string]string{"width": "100"})
	if !strings.Contains(svg, `viewBox="0 0 200 40"`) {
		t.Errorf("local logo viewBox not preserved: %q", svg)
	}
	if !strings.Contains(svg, `width="100"`) || !strings.Contains(svg, `height="20"`) {
		t.Errorf("aspect-ratio sizing wrong (want 100x20): %q", svg)
	}
	// A name absent from the local dir falls through to the set.
	if svg := Get("circle-help"); svg == "" {
		t.Error("circle-help should fall through to lucide")
	}
}

// ---------------------------------------------------------------------------
// SVG parser
// ---------------------------------------------------------------------------

func TestParseSVGFile(t *testing.T) {
	body, vb, w, h, err := parseSVGFile([]byte(`<svg viewBox="0 0 10 20" width="10" height="20"><path d="X"/></svg>`))
	if err != nil {
		t.Fatalf("parseSVGFile: %v", err)
	}
	if body != `<path d="X"/>` || vb != "0 0 10 20" || w != 10 || h != 20 {
		t.Errorf("parse = (%q,%q,%d,%d)", body, vb, w, h)
	}

	// <script> and on* handlers are stripped.
	body, _, _, _, err = parseSVGFile([]byte(`<svg viewBox="0 0 8 8"><script>alert(1)</script><path onclick="x()" d="Y"/></svg>`))
	if err != nil {
		t.Fatalf("parseSVGFile sanitize: %v", err)
	}
	if strings.Contains(body, "script") || strings.Contains(body, "onclick") {
		t.Errorf("sanitize failed: %q", body)
	}

	// Missing viewBox is synthesized from width/height.
	_, vb, _, _, err = parseSVGFile([]byte(`<svg width="48" height="24"><path d="Z"/></svg>`))
	if err != nil || vb != "0 0 48 24" {
		t.Errorf("synthesized viewBox = %q (err=%v), want 0 0 48 24", vb, err)
	}

	if _, _, _, _, err := parseSVGFile([]byte(`<div>not svg</div>`)); err == nil {
		t.Error("parseSVGFile should error on non-SVG input")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

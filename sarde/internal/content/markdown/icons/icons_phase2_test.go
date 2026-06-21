package icons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	// The "brands" set alias routes to the same simple-icons collection.
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
	writeFile(t, dir, "mylogo.svg", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 40"><rect width="200" height="40"/></svg>`)

	if err := LoadIconDirectory(dir); err != nil {
		t.Fatalf("LoadIconDirectory: %v", err)
	}

	// Local "mylogo" resolves via the local provider.
	if svg := Get("mylogo"); svg == "" {
		t.Error("local mylogo should resolve")
	} else if !strings.Contains(svg, "rect") {
		t.Errorf("local mylogo content wrong: %q", svg)
	}

	// A name absent from the local dir falls through to the set.
	if svg := Get("circle-help"); svg == "" {
		t.Error("circle-help should fall through to lucide")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

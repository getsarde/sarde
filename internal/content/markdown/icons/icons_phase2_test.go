package icons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestLoadedSetLicenses(t *testing.T) {
	got := map[string]string{}
	for _, s := range LoadedSetLicenses() {
		got[s.Prefix] = s.SPDX
	}
	if got["lucide"] != "ISC" {
		t.Errorf("lucide license = %q, want ISC", got["lucide"])
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

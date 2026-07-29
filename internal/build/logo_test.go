package build

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

// writePublic creates projectDir/public/<rel> with the given content.
func writePublic(t *testing.T, projectDir, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(projectDir, consts.DirPublic, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// writePNG creates a w×h PNG under projectDir/public/<rel>.
func writePNG(t *testing.T, projectDir, rel string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	path := filepath.Join(projectDir, consts.DirPublic, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
}

func TestResolveLogo_Unconfigured(t *testing.T) {
	got := resolveLogo(config.Logo{}, t.TempDir(), &engine.URLResolver{BasePath: "/"})
	if got != (engine.LogoContext{}) {
		t.Errorf("resolveLogo with no logo = %+v, want zero value", got)
	}
}

func TestResolveLogo_SingleVariant(t *testing.T) {
	// The string form of `logo:` sets Light and Dark to the same path; it must
	// collapse to one image so the template emits a single <img>.
	dir := t.TempDir()
	writePublic(t, dir, "logo.svg", []byte("<svg/>"))

	got := resolveLogo(
		config.Logo{Light: "/logo.svg", Dark: "/logo.svg", Alt: "Sarde"},
		dir,
		&engine.URLResolver{BasePath: "/"},
	)
	if !got.Single {
		t.Error("Single = false, want true when both variants share a path")
	}
	if got.Light.URL != "/logo.svg" || got.Dark.URL != "/logo.svg" {
		t.Errorf("URLs = %q / %q, want /logo.svg for both", got.Light.URL, got.Dark.URL)
	}
	if got.Alt != "Sarde" {
		t.Errorf("Alt = %q, want Sarde", got.Alt)
	}
}

func TestResolveLogo_OneVariantFillsBoth(t *testing.T) {
	// A logo declaring only `dark:` must still render in light mode rather than
	// disappearing when the theme is toggled.
	dir := t.TempDir()
	writePublic(t, dir, "dark.svg", []byte("<svg/>"))

	got := resolveLogo(config.Logo{Dark: "/dark.svg"}, dir, &engine.URLResolver{BasePath: "/"})
	if !got.Single {
		t.Error("Single = false, want true when only one variant is configured")
	}
	if got.Light.URL != "/dark.svg" {
		t.Errorf("Light.URL = %q, want /dark.svg (fallback to the only variant)", got.Light.URL)
	}
}

func TestResolveLogo_DistinctVariants(t *testing.T) {
	dir := t.TempDir()
	writePublic(t, dir, "light.svg", []byte("<svg/>"))
	writePublic(t, dir, "dark.svg", []byte("<svg/>"))

	got := resolveLogo(
		config.Logo{Light: "/light.svg", Dark: "/dark.svg"},
		dir,
		&engine.URLResolver{BasePath: "/"},
	)
	if got.Single {
		t.Error("Single = true, want false for distinct light/dark paths")
	}
	if got.Light.URL != "/light.svg" || got.Dark.URL != "/dark.svg" {
		t.Errorf("URLs = %q / %q, want /light.svg / /dark.svg", got.Light.URL, got.Dark.URL)
	}
}

func TestResolveLogo_RasterDimensionsProbed(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, dir, "img/logo.png", 120, 40)

	got := resolveLogo(config.Logo{Light: "/img/logo.png"}, dir, &engine.URLResolver{BasePath: "/"})
	if got.Light.Width != 120 || got.Light.Height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", got.Light.Width, got.Light.Height)
	}
}

func TestResolveLogo_SVGDimensionsSkipped(t *testing.T) {
	dir := t.TempDir()
	writePublic(t, dir, "logo.SVG", []byte("<svg width='10' height='10'/>"))

	got := resolveLogo(config.Logo{Light: "/logo.SVG"}, dir, &engine.URLResolver{BasePath: "/"})
	if got.Light.Width != 0 || got.Light.Height != 0 {
		t.Errorf("dimensions = %dx%d, want 0x0 for SVG", got.Light.Width, got.Light.Height)
	}
}

func TestResolveLogo_MissingFileDegrades(t *testing.T) {
	// A bad path warns but must not fail the build or drop the <img>.
	got := resolveLogo(config.Logo{Light: "/nope.png"}, t.TempDir(), &engine.URLResolver{BasePath: "/"})
	if got.Light.URL != "/nope.png" {
		t.Errorf("Light.URL = %q, want /nope.png", got.Light.URL)
	}
	if got.Light.Width != 0 || got.Light.Height != 0 {
		t.Errorf("dimensions = %dx%d, want 0x0 for a missing file", got.Light.Width, got.Light.Height)
	}
}

func TestResolveLogo_BasePathPrefixed(t *testing.T) {
	dir := t.TempDir()
	writePublic(t, dir, "logo.svg", []byte("<svg/>"))

	got := resolveLogo(config.Logo{Light: "/logo.svg"}, dir, &engine.URLResolver{BasePath: "/sarde/"})
	if got.Light.URL != "/sarde/logo.svg" {
		t.Errorf("Light.URL = %q, want /sarde/logo.svg", got.Light.URL)
	}
}

func TestResolveLogo_ReplacesTitle(t *testing.T) {
	dir := t.TempDir()
	writePublic(t, dir, "logo.svg", []byte("<svg/>"))
	tr := true

	got := resolveLogo(
		config.Logo{Light: "/logo.svg", ReplacesTitle: &tr},
		dir,
		&engine.URLResolver{BasePath: "/"},
	)
	if !got.ReplacesTitle {
		t.Error("ReplacesTitle = false, want true")
	}
}

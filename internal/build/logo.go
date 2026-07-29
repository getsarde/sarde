package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
)

// resolveLogo turns the configured site logo into template-ready data: URLs
// prefixed with the site base path and intrinsic dimensions probed from the
// files under public/.
//
// A logo declaring only one variant is used for both themes, so a single-variant
// logo never disappears when the theme is toggled. When both variants resolve to
// the same path the result is marked Single and the template renders one <img>.
func resolveLogo(cfg config.Logo, projectDir string, resolver *engine.URLResolver) engine.LogoContext {
	light, dark := cfg.Light, cfg.Dark
	switch {
	case light == "" && dark == "":
		return engine.LogoContext{}
	case light == "":
		light = dark
	case dark == "":
		dark = light
	}

	out := engine.LogoContext{
		Alt:           cfg.Alt,
		ReplacesTitle: config.BoolVal(cfg.ReplacesTitle, false),
		Single:        light == dark,
		Light:         resolveLogoImage(light, projectDir, resolver),
	}
	if out.Single {
		out.Dark = out.Light
	} else {
		out.Dark = resolveLogoImage(dark, projectDir, resolver)
	}
	return out
}

// resolveLogoImage resolves one variant's URL and, for raster formats, its
// intrinsic dimensions. Missing files are reported but never fail the build.
func resolveLogoImage(path, projectDir string, resolver *engine.URLResolver) engine.LogoImage {
	img := engine.LogoImage{URL: path}
	if resolver != nil {
		img.URL = resolver.URL(path, "", "")
	}

	srcPath := filepath.Join(projectDir, consts.DirPublic, filepath.FromSlash(strings.TrimPrefix(path, "/")))
	if _, err := os.Stat(srcPath); err != nil {
		devlog.Warn("logo", "logo file not found: %s (expected at %s)", path, srcPath)
		return img
	}

	// SVGs have no reliable intrinsic pixel size; CSS sizes them instead.
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return img
	}
	if w, h, err := asset.ImageSize(srcPath); err == nil {
		img.Width, img.Height = w, h
	}
	return img
}

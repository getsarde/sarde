package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
)

// loadIconSources configures the icon engine from b.config.Icons: the default
// prefix, any extra Iconify sets (explicit files plus a sets_dir of *.json),
// and the local icons/ directory. Failures are warnings, never fatal — a
// missing or malformed icon source must not break the build.
func (b *SiteBuilder) loadIconSources() {
	ic := b.config.Icons
	icons.SetDefaultPrefix(ic.DefaultPrefix)
	icons.SetRenderMode(ic.Render)
	icons.DrainMissing() // misses are reported per build

	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(b.projectDir, p)
	}
	loadSet := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			devlog.Warn("icons", "read set %s: %v", path, err)
			return
		}
		if err := icons.LoadCollection(data); err != nil {
			devlog.Warn("icons", "load set %s: %v", path, err)
		}
	}

	for _, set := range ic.Sets {
		if set.File != "" {
			loadSet(resolve(set.File))
		}
	}
	// `sarde icons add` downloads into icon-sets/ unless icons.sets_dir
	// names another folder, so that folder is read by default too.
	setsDir := ic.SetsDir
	if setsDir == "" {
		setsDir = consts.DirIconSets
	}
	dir := resolve(setsDir)
	if entries, err := os.ReadDir(dir); err != nil {
		if !os.IsNotExist(err) {
			devlog.Warn("icons", "read sets_dir %s: %v", dir, err)
		}
	} else {
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".json") {
				loadSet(filepath.Join(dir, e.Name()))
			}
		}
	}

	localDir := ic.LocalDir
	if localDir == "" {
		localDir = "icons"
	}
	if err := icons.LoadIconDirectory(resolve(localDir)); err != nil {
		devlog.Warn("icons", "load local dir %s: %v", localDir, err)
	}
}

func buildIconLicenses() []engine.IconLicense {
	sets := icons.LoadedSetLicenses()
	out := make([]engine.IconLicense, 0, len(sets))
	for _, s := range sets {
		out = append(out, engine.IconLicense{Prefix: s.Prefix, Title: s.Title, SPDX: s.SPDX, URL: s.URL})
	}
	return out
}

// warnMissingIcons reports each icon name the pages rendered this build
// asked for but no loaded set has, once per name. Each renders the
// placeholder icon.
func (b *SiteBuilder) warnMissingIcons() {
	for _, name := range icons.DrainMissing() {
		devlog.Warn("icons", "unknown icon %q; the placeholder icon renders instead (check the name, or load its set with sarde icons add)", name)
	}
}

func (b *SiteBuilder) warnIconAttribution() {
	if strings.TrimSpace(b.config.Icons.Attribution) != "" {
		return
	}
	for _, s := range icons.UsedSetLicenses() {
		if !requiresAttribution(s.SPDX) {
			continue
		}
		label := s.Title
		if label == "" {
			label = s.SPDX
		}
		devlog.Warn("icons", "set %q (%s) requires attribution; set icons.attribution or render a credits page from .Site.IconLicenses", s.Prefix, label)
	}
}

func requiresAttribution(spdx string) bool {
	s := strings.ToUpper(strings.TrimSpace(spdx))
	switch {
	case strings.HasPrefix(s, "CC-BY"):
		return true
	case strings.HasPrefix(s, "OFL"):
		return true
	case strings.Contains(s, "GPL"):
		return true
	}
	return false
}

package icons

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/devlog"
)

// LoadIconDirectory scans dirPath for *.svg files and registers each into the
// local icon namespace, keyed by filename without extension (e.g. logo.svg →
// "logo"). Bare icon names resolve here BEFORE any Iconify set, so a project can
// supply custom/brand icons or override a set icon by dropping a file. Files
// that fail to parse are skipped with a warning. A missing directory is not an
// error (returns nil) — the local dir is an optional convention.
func LoadIconDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	loaded := make(map[string]localIcon)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".svg") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dirPath, name))
		if readErr != nil {
			devlog.Warn("icons", "read %s: %v", name, readErr)
			continue
		}
		body, viewBox, w, h, parseErr := parseSVGFile(data)
		if parseErr != nil {
			devlog.Warn("icons", "parse %s: %v", name, parseErr)
			continue
		}
		key := strings.TrimSuffix(name, filepath.Ext(name))
		loaded[key] = localIcon{Body: body, ViewBox: viewBox, Width: w, Height: h}
	}

	ensureLoaded()
	resolver.mu.Lock()
	for k, v := range loaded {
		resolver.localIcons[k] = v
	}
	resolver.mu.Unlock()
	return nil
}

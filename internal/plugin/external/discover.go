package external

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
)

// DiscoverSlugs returns the names of directories under {projectDir}/plugins
// that contain a plugin.yaml. It is deliberately cheap (no YAML parsing) so
// it can run before config resolution to extend the known plugin name list.
func DiscoverSlugs(projectDir string) []string {
	var slugs []string
	for _, dir := range DiscoverDirs(projectDir) {
		slugs = append(slugs, filepath.Base(dir))
	}
	return slugs
}

// DiscoverDirs returns the absolute paths of plugin directories containing a
// plugin.yaml, in sorted (directory listing) order. Dot-directories are
// skipped.
func DiscoverDirs(projectDir string) []string {
	root := filepath.Join(projectDir, consts.DirPlugins)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, consts.FilePluginManifest)); err != nil {
			continue
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

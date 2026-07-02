package plugin

import (
	"path"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

func feedEnabledCollections(cfg map[string]any, collections map[string]*engine.Collection) []string {
	names := cfgutil.StringSlice(cfg, "collections")
	if len(names) > 0 {
		return names
	}
	for name, col := range collections {
		if col.Config != nil && col.Config.Feed {
			names = append(names, name)
		}
	}
	return names
}

// shouldExclude checks if a URL path matches any exclude pattern.
// Uses path.Match (slash-separated glob semantics on every platform);
// filepath.Match would make patterns behave differently on Windows.
func shouldExclude(urlPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, urlPath); matched {
			return true
		}
		// Also check without trailing slash.
		trimmed := strings.TrimRight(urlPath, "/")
		if matched, _ := path.Match(pattern, trimmed); matched {
			return true
		}
	}
	return false
}

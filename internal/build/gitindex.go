package build

import (
	"fmt"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// resolveGitIndex builds the batched git-history index used by the "git"
// last_updated strategy, and turns any git problem into a single build warning
// rather than a silent per-file fallback to mtime.
//
// Returns (nil, nil) for every other strategy, so mtime and disabled sites pay
// nothing.
func (b *SiteBuilder) resolveGitIndex(files []content.ContentFile, contentDir string) (*content.GitLastModIndex, *engine.ValidationWarning) {
	if string(b.config.Build.LastUpdated) != "git" {
		return nil, nil
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.FilePath)
	}

	idx, err := content.BuildGitLastModIndex(contentDir, paths)
	if err != nil {
		return idx, &engine.ValidationWarning{
			File:  consts.FileSiteConfig,
			Field: "build.last_updated",
			Message: fmt.Sprintf(
				"git strategy unavailable (%v); page dates fall back to file modification time. "+
					"In CI that is the checkout time, so every page reports the same timestamp. "+
					"Build inside a git repository, or set build.last_updated: mtime to silence this.", err),
			Level: "warning",
		}
	}

	if idx.Shallow() {
		return idx, &engine.ValidationWarning{
			File:  consts.FileSiteConfig,
			Field: "build.last_updated",
			Message: "git strategy is running against a shallow clone; pages whose last change predates " +
				"the shallow boundary fall back to file modification time, which in CI is the checkout time. " +
				"Use a full clone (fetch-depth: 0 in GitHub Actions) for accurate dates.",
			Level: "warning",
		}
	}

	return idx, nil
}

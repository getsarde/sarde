package build

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

// slowestPagesCount is how many pages BuildResult.SlowestPages keeps.
const slowestPagesCount = 5

// slowestPages sums each page's markdown render time (md, indexed like pages)
// and template render time (from rendered), then returns the n slowest,
// slowest first. Pages rendered more than once from the same source (e.g.
// versioned copies) are merged under one path.
func slowestPages(projectDir string, pages []*engine.Page, md []time.Duration, rendered []RenderedPage, n int) []engine.PageTiming {
	total := make(map[string]time.Duration, len(pages))
	for i, p := range pages {
		if i < len(md) && md[i] > 0 {
			total[pageTimingPath(projectDir, p)] += md[i]
		}
	}
	for _, rp := range rendered {
		if rp.Page != nil && rp.renderTime > 0 {
			total[pageTimingPath(projectDir, rp.Page)] += rp.renderTime
		}
	}

	timings := make([]engine.PageTiming, 0, len(total))
	for path, d := range total {
		timings = append(timings, engine.PageTiming{Path: path, Duration: d})
	}
	slices.SortFunc(timings, func(a, b engine.PageTiming) int {
		return cmp.Or(cmp.Compare(b.Duration, a.Duration), cmp.Compare(a.Path, b.Path))
	})
	if len(timings) > n {
		timings = timings[:n]
	}
	return timings
}

// pageTimingPath names a page by its project-relative source path, falling
// back to the permalink for generated pages without a source file.
func pageTimingPath(projectDir string, p *engine.Page) string {
	if p.FilePath == "" {
		return p.RelPermalink
	}
	if filepath.IsAbs(p.FilePath) {
		if rel, err := filepath.Rel(projectDir, p.FilePath); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(p.FilePath)
}

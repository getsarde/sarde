package build

import (
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

// pruneOrphanedTaxonomyOutputs deletes output files for taxonomy terms that
// existed in the previous build but have no pages in the current one.
// Skipped on the body-only fast path (single-lang) since term membership
// is provably unchanged when the frontmatter digest gate passes.
func (b *SiteBuilder) pruneOrphanedTaxonomyOutputs(s *incrementalRebuildState) error {
	if s.oldTaxonomies == nil || b.urlResolver == nil || b.lastOutputDir == "" {
		s.recordTiming("Pruning orphaned taxonomy files")
		return nil
	}
	if s.bodyOnly && !s.isMultiLang {
		s.recordTiming("Pruning orphaned taxonomy files")
		return nil
	}

	oldByLang := s.oldTaxByLang
	if oldByLang == nil {
		oldByLang = map[string]map[string]*engine.Taxonomy{"": s.oldTaxonomies}
	}

	for lang, oldLangTax := range oldByLang {
		newLangTax := s.rebuildTaxByLang[lang]
		for taxName, oldTax := range oldLangTax {
			if oldTax == nil || !b.config.Taxonomies[taxName].ShouldRender() {
				continue
			}
			newTax := newLangTax[taxName]
			if newTax == nil {
				for _, term := range oldTax.Terms {
					b.removeOrphanedTermOutputs(taxName, term, oldTax.PaginateBy, lang)
				}
				b.removeOrphanedOutput("/"+taxName+"/", lang, "taxonomy index "+taxName)
				continue
			}
			for slug, term := range oldTax.Terms {
				if _, ok := newTax.Terms[slug]; ok {
					continue
				}
				b.removeOrphanedTermOutputs(taxName, term, oldTax.PaginateBy, lang)
			}
		}
	}

	s.recordTiming("Pruning orphaned taxonomy files")
	return nil
}

func (b *SiteBuilder) removeOrphanedTermOutputs(taxName string, term *engine.TaxonomyTerm, paginateBy int, lang string) {
	bare := "/" + taxName + "/" + term.Slug + "/"
	b.removeOrphanedOutput(bare, lang, "term "+taxName+"/"+term.Slug)

	if paginateBy <= 0 {
		paginateBy = consts.DefaultPaginateBy
	}
	total := (len(term.Pages) + paginateBy - 1) / paginateBy
	for n := 2; n <= total; n++ {
		b.removeOrphanedOutput(sardetemplate.PaginationURL(bare, n), lang,
			fmt.Sprintf("term %s/%s page %d", taxName, term.Slug, n))
	}
}

// collectPageOutputPaths builds a set of relative output paths from the
// current lastAllPages snapshot (including aliases). Used before a full
// rebuild to detect pages that disappear.
func (b *SiteBuilder) collectPageOutputPaths() map[string]struct{} {
	if b.lastAllPages == nil || b.urlResolver == nil {
		return nil
	}
	paths := make(map[string]struct{}, len(b.lastAllPages))
	for _, p := range b.lastAllPages {
		if p.RelPermalink == "" {
			continue
		}
		paths[PageOutputPath(b.urlResolver.OutputRelPath(p.RelPermalink, p.Lang, ""))] = struct{}{}
		for _, alias := range p.Aliases {
			paths[PageOutputPath(alias)] = struct{}{}
		}
	}
	return paths
}

// pruneRemovedPageOutputs deletes output files that existed in oldPaths but
// are no longer produced by the current build. Called after a dev-mode full
// rebuild fallback to clean up HTML for deleted/renamed content files.
func (b *SiteBuilder) pruneRemovedPageOutputs(oldPaths map[string]struct{}) {
	if oldPaths == nil || b.lastOutputDir == "" {
		return
	}
	current := b.collectPageOutputPaths()
	for relOut := range oldPaths {
		if _, ok := current[relOut]; ok {
			continue
		}
		target, err := outputpath.SafeJoin(b.lastOutputDir, relOut)
		if err != nil {
			continue
		}
		if err := os.Remove(target); err != nil {
			if !os.IsNotExist(err) {
				devlog.Warn("build", "rebuild fallback: failed to remove stale output %s: %v", relOut, err)
			}
			continue
		}
		devlog.Log("build", "rebuild fallback: removed stale output %s", relOut)
	}
}

func (b *SiteBuilder) removeOrphanedOutput(barePermalink, lang, desc string) {
	relOut := PageOutputPath(b.urlResolver.OutputRelPath(barePermalink, lang, ""))
	target, err := outputpath.SafeJoin(b.lastOutputDir, relOut)
	if err != nil {
		devlog.Warn("build", "ContentRebuild: could not resolve orphaned %s output path: %v", desc, err)
		return
	}
	if err := os.Remove(target); err != nil {
		if !os.IsNotExist(err) {
			devlog.Warn("build", "ContentRebuild: failed to remove orphaned %s output %s: %v", desc, relOut, err)
		}
		return
	}
	devlog.Log("build", "ContentRebuild: removed orphaned %s output %s", desc, relOut)
}

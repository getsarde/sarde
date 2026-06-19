package build

import (
	"fmt"
	"sort"
	"strings"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/workers"
)

// ---------------------------------------------------------------------------
// Page index helpers
// ---------------------------------------------------------------------------

func populatePageIndexHeadings(idx *content.PageIndex, pages []*engine.Page) {
	for _, page := range pages {
		setPageIndexHeadings(idx, page)
	}
}

func setPageIndexHeadings(idx *content.PageIndex, page *engine.Page) {
	if idx == nil || page == nil || len(page.Headings) == 0 {
		return
	}
	// First-match: if another page already registered headings for this
	// permalink, two distinct pages claim the same URL. Keep the first —
	// overwriting would make anchor validation depend on page order (the source
	// of the nondeterministic broken_anchor counts). Do NOT merge heading sets:
	// that would let anchors pass against headings from the page that does not
	// serve at this URL, silently hiding broken links. Silent: the same URL
	// collision is already recorded (and reported) via byPermalink in BuildPageIndex.
	if existing := idx.HeadingsFor(page.Permalink); existing != nil {
		return
	}
	ids := make([]string, len(page.Headings))
	for i, h := range page.Headings {
		ids[i] = h.ID
	}
	idx.SetHeadings(page.Permalink, ids)
}

func updateValidationEntry(data map[string]engine.ValidationEntry, page *engine.Page, links []engine.CollectedLink) {
	if len(links) == 0 {
		delete(data, page.Permalink)
		return
	}
	data[page.Permalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath}
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

func minWorkerLimit(workerCount, workItems int) int {
	if workerCount < 1 {
		workerCount = workers.Count()
	}
	if workItems > 0 && workItems < workerCount {
		return workItems
	}
	return workerCount
}

func countMarkdownPages(pages []*engine.Page) int {
	count := 0
	for _, page := range pages {
		if page.RawContent != "" {
			count++
		}
	}
	return count
}

func sortedCollectionNames(collections map[string]*engine.Collection) []string {
	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedTaxonomyNames(taxonomies map[string]*engine.Taxonomy) []string {
	names := make([]string, 0, len(taxonomies))
	for name := range taxonomies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func tabbedCollectionRedirect(page *engine.Page) string {
	col := page.Collection
	if col == nil || !col.IsTabbed || col.IndexPage != page || len(col.Tabs) == 0 {
		return ""
	}
	return col.Tabs[0].Permalink
}

func buildRedirectHTML(target string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head>`+
		`<meta http-equiv="refresh" content="0;url=%s">`+
		`<link rel="canonical" href="%s">`+
		`</head><body><p><a href="%s">Continue</a></p></body></html>`,
		target, target, target)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const warnCollapseThreshold = 3

// collisionWarnThreshold is intentionally higher than warnCollapseThreshold:
// collision lines carry the kept/dropped file paths needed to fix the dup, and a
// real site has only a handful of distinct colliding URLs. The summary (which
// drops the file paths) is reserved for pathological, systemic dup bugs.
const collisionWarnThreshold = 10

// emitCollisionWarnings reports duplicate-URL page collisions deduped by URL and
// capped per build. Up to collisionWarnThreshold distinct URLs each get a detail
// line (URL + kept + dropped files); past that they collapse to one count + URL
// summary. Order follows first-seen (deterministic after the fallback-sort fix).
func emitCollisionWarnings(collisions []content.Collision) {
	if len(collisions) == 0 {
		return
	}

	type agg struct {
		kept    string
		dropped []string
	}
	byKey := make(map[string]*agg)
	var order []string
	for _, c := range collisions {
		a, ok := byKey[c.Permalink]
		if !ok {
			a = &agg{kept: c.KeptFile}
			byKey[c.Permalink] = a
			order = append(order, c.Permalink)
		}
		a.dropped = append(a.dropped, c.DroppedFile)
	}

	if len(order) <= collisionWarnThreshold {
		for _, k := range order {
			a := byKey[k]
			devlog.Warn("pages", "URL collision: %q resolved by %d pages — keeping %q, ignoring %s",
				k, len(a.dropped)+1, a.kept, strings.Join(a.dropped, ", "))
		}
		return
	}
	devlog.Warn("pages", "%d duplicate-URL collisions (multiple pages resolve to one URL; keeping first match) — %s",
		len(order), strings.Join(order, ", "))
}

func emitTaxonomyWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}

	type group struct {
		taxName string
		terms   []string
	}
	groups := make(map[string]*group)
	var order []string

	for _, w := range warnings {
		taxPart, termPart, ok := strings.Cut(w, ": term ")
		if !ok {
			devlog.Warn("taxonomy", "%s", w)
			continue
		}
		taxName := strings.TrimPrefix(taxPart, "taxonomy ")
		taxName = strings.Trim(taxName, "\"")

		term := termPart
		if idx := strings.Index(term, "\" "); idx >= 0 {
			term = term[1:idx]
		} else {
			term = strings.Trim(term, "\"")
		}

		g, exists := groups[taxName]
		if !exists {
			g = &group{taxName: taxName}
			groups[taxName] = g
			order = append(order, taxName)
		}
		g.terms = append(g.terms, term)
	}

	for _, name := range order {
		g := groups[name]
		if len(g.terms) <= warnCollapseThreshold {
			for _, t := range g.terms {
				devlog.Warn("taxonomy", "%q: term %q is not defined in data/%s.yml", name, t, name)
			}
		} else {
			devlog.Warn("taxonomy", "%q: %d undefined terms (define in data/%s.yml) — %s", name, len(g.terms), name, strings.Join(g.terms, ", "))
		}
	}
}

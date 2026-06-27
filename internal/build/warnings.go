package build

import (
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/devlog"
)

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

func dedupStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

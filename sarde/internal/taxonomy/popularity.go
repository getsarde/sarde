package taxonomy

import (
	"sort"

	"github.com/frostybee/sarde/internal/engine"
)

// ComputeTermEntries builds a sorted slice of TermEntry with popularity tiers.
// Hidden terms are excluded. Sort order: Priority desc, Count desc, Name asc.
func ComputeTermEntries(tax *engine.Taxonomy) []*engine.TermEntry {
	entries := make([]*engine.TermEntry, 0, len(tax.Terms))
	for _, term := range tax.Terms {
		if term.Hidden {
			continue
		}
		entries = append(entries, &engine.TermEntry{
			TaxonomyTerm: term,
			Count:        len(term.Pages),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Priority != entries[j].Priority {
			return entries[i].Priority > entries[j].Priority
		}
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Name < entries[j].Name
	})

	assignPopTiers(entries)
	return entries
}

// assignPopTiers assigns a 1-5 popularity tier via quintile bucketing.
func assignPopTiers(entries []*engine.TermEntry) {
	n := len(entries)
	if n == 0 {
		return
	}

	counts := make([]int, n)
	for i, e := range entries {
		counts[i] = e.Count
	}
	sort.Ints(counts)

	min := counts[0]
	max := counts[n-1]

	for _, e := range entries {
		e.PopTier = countToTier(e.Count, min, max)
	}
}

// countToTier maps a count to a 1-5 tier based on its position in the range.
func countToTier(count, min, max int) int {
	if max == min {
		return 3
	}
	ratio := float64(count-min) / float64(max-min)
	switch {
	case ratio >= 0.8:
		return 5
	case ratio >= 0.6:
		return 4
	case ratio >= 0.4:
		return 3
	case ratio >= 0.2:
		return 2
	default:
		return 1
	}
}

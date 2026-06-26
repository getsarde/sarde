package links

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
)

// LaneSummary holds link-checking stats for a single rendered lane.
type LaneSummary struct {
	Dim        DimKey
	Pages      int
	Links      int
	Broken     int
	External   int
	IsFallback bool // true if this lane only contains fallback pages
}

// CoverageSummary reports which lanes were checked and aggregate stats.
type CoverageSummary struct {
	Lanes       []LaneSummary
	TotalLanes  int
	TotalPages  int
	TotalLinks  int
	TotalBroken int
	MissedLanes []DimKey // expected lanes with no links checked
}

// ComputeCoverage analyzes the link graph and the set of rendered pages to
// produce a coverage summary. expectedLanes is the full set of lanes that
// should produce output (from EnumerateLanes); the graph provides what was
// actually checked.
func ComputeCoverage(graph *LinkGraph, pages []*engine.Page, expectedLanes []DimKey) CoverageSummary {
	// Count pages per lane (only pages with content that went through the renderer).
	type laneStats struct {
		pages      int
		links      int
		broken     int
		external   int
		hasFallbackOnly bool
	}

	pagesByLane := make(map[DimKey]int)
	fallbackByLane := make(map[DimKey]bool)
	for _, p := range pages {
		if p.RawContent == "" {
			continue
		}
		collName := ""
		if p.Collection != nil {
			collName = p.Collection.Name
		}
		dim := DimKey{
			Collection: collName,
			Lang:       p.Lang,
			Version:    p.Version,
		}
		pagesByLane[dim]++
		if _, exists := fallbackByLane[dim]; !exists {
			fallbackByLane[dim] = p.IsFallback
		} else if !p.IsFallback {
			fallbackByLane[dim] = false
		}
	}

	// Tally links per lane from the graph.
	linksByLane := make(map[DimKey]*laneStats)
	refs := graph.Refs()
	for _, ref := range refs {
		ls, ok := linksByLane[ref.Dim]
		if !ok {
			ls = &laneStats{}
			linksByLane[ref.Dim] = ls
		}
		ls.links++
		switch ref.Status {
		case StatusBrokenTarget, StatusBrokenAnchor:
			ls.broken++
		case StatusExternal:
			ls.external++
		}
	}

	// Merge page counts and link stats into lane summaries.
	// Use the union of pagesByLane and linksByLane keys to capture all active lanes.
	allDims := make(map[DimKey]bool)
	for dim := range pagesByLane {
		allDims[dim] = true
	}
	for dim := range linksByLane {
		allDims[dim] = true
	}

	var lanes []LaneSummary
	totalPages := 0
	totalLinks := 0
	totalBroken := 0

	for dim := range allDims {
		ls := linksByLane[dim]
		pCount := pagesByLane[dim]
		summary := LaneSummary{
			Dim:        dim,
			Pages:      pCount,
			IsFallback: fallbackByLane[dim],
		}
		if ls != nil {
			summary.Links = ls.links
			summary.Broken = ls.broken
			summary.External = ls.external
		}
		lanes = append(lanes, summary)
		totalPages += pCount
		totalLinks += summary.Links
		totalBroken += summary.Broken
	}

	sortLaneSummaries(lanes)

	// Identify expected lanes that have no recorded links and no pages with content.
	checkedLanes := make(map[DimKey]bool, len(allDims))
	for dim := range allDims {
		checkedLanes[dim] = true
	}
	var missed []DimKey
	for _, expected := range expectedLanes {
		if !checkedLanes[expected] {
			missed = append(missed, expected)
		}
	}

	return CoverageSummary{
		Lanes:       lanes,
		TotalLanes:  len(lanes),
		TotalPages:  totalPages,
		TotalLinks:  totalLinks,
		TotalBroken: totalBroken,
		MissedLanes: missed,
	}
}

// EnumerateLanes computes the full set of expected lanes from collections and
// language configuration. Each collection × language × version combination
// that produces output is one lane.
func EnumerateLanes(collections map[string]*engine.Collection, languages []string) []DimKey {
	if len(languages) == 0 {
		languages = []string{""}
	}

	var lanes []DimKey
	for name, col := range collections {
		versions := []string{""}
		if col.Config != nil && col.Config.Versioning != nil && col.Config.Versioning.Enabled {
			versions = nil
			for _, vd := range col.Config.Versioning.Versions {
				versions = append(versions, vd.ID)
			}
		}
		for _, lang := range languages {
			for _, ver := range versions {
				lanes = append(lanes, DimKey{
					Collection: name,
					Lang:       lang,
					Version:    ver,
				})
			}
		}
	}

	sort.Slice(lanes, func(i, j int) bool {
		return dimLess(lanes[i], lanes[j])
	})
	return lanes
}

func sortLaneSummaries(lanes []LaneSummary) {
	sort.Slice(lanes, func(i, j int) bool {
		return dimLess(lanes[i].Dim, lanes[j].Dim)
	})
}

func dimLess(a, b DimKey) bool {
	if a.Collection != b.Collection {
		return a.Collection < b.Collection
	}
	if a.Lang != b.Lang {
		return a.Lang < b.Lang
	}
	return a.Version < b.Version
}

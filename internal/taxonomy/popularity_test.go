package taxonomy

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func makeTerm(name string, pageCount int, priority int, hidden bool) *engine.TaxonomyTerm {
	pages := make([]*engine.Page, pageCount)
	for i := range pages {
		pages[i] = &engine.Page{PageIdentity: engine.PageIdentity{Title: name}}
	}
	return &engine.TaxonomyTerm{
		Name:     name,
		Slug:     name,
		Pages:    pages,
		Priority: priority,
		Hidden:   hidden,
	}
}

func TestComputeTermEntries_Sort(t *testing.T) {
	tax := &engine.Taxonomy{
		Terms: map[string]*engine.TaxonomyTerm{
			"alpha": makeTerm("alpha", 3, 0, false),
			"beta":  makeTerm("beta", 5, 0, false),
			"gamma": makeTerm("gamma", 5, 10, false), // higher priority
		},
	}

	entries := ComputeTermEntries(tax)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// gamma first (priority 10), then beta (count 5, alpha < beta), then alpha (count 3)
	if entries[0].Name != "gamma" {
		t.Errorf("entries[0] = %q, want gamma (highest priority)", entries[0].Name)
	}
	if entries[1].Name != "beta" {
		t.Errorf("entries[1] = %q, want beta (count 5, lower priority)", entries[1].Name)
	}
	if entries[2].Name != "alpha" {
		t.Errorf("entries[2] = %q, want alpha (count 3)", entries[2].Name)
	}
}

func TestComputeTermEntries_HiddenExcluded(t *testing.T) {
	tax := &engine.Taxonomy{
		Terms: map[string]*engine.TaxonomyTerm{
			"visible": makeTerm("visible", 2, 0, false),
			"hidden":  makeTerm("hidden", 5, 0, true),
		},
	}

	entries := ComputeTermEntries(tax)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (hidden excluded), got %d", len(entries))
	}
	if entries[0].Name != "visible" {
		t.Errorf("expected visible, got %q", entries[0].Name)
	}
}

func TestComputeTermEntries_EmptyTaxonomy(t *testing.T) {
	tax := &engine.Taxonomy{
		Terms: map[string]*engine.TaxonomyTerm{},
	}

	entries := ComputeTermEntries(tax)

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestComputeTermEntries_SingleTerm(t *testing.T) {
	tax := &engine.Taxonomy{
		Terms: map[string]*engine.TaxonomyTerm{
			"only": makeTerm("only", 3, 0, false),
		},
	}

	entries := ComputeTermEntries(tax)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].PopTier != 3 {
		t.Errorf("PopTier = %d, want 3 (all-equal case)", entries[0].PopTier)
	}
}

func TestComputeTermEntries_EqualCounts(t *testing.T) {
	tax := &engine.Taxonomy{
		Terms: map[string]*engine.TaxonomyTerm{
			"a": makeTerm("a", 2, 0, false),
			"b": makeTerm("b", 2, 0, false),
			"c": makeTerm("c", 2, 0, false),
		},
	}

	entries := ComputeTermEntries(tax)

	for _, e := range entries {
		if e.PopTier != 3 {
			t.Errorf("%s PopTier = %d, want 3 (all equal)", e.Name, e.PopTier)
		}
	}
}

func TestCountToTier_Boundaries(t *testing.T) {
	tests := []struct {
		count, min, max int
		want            int
	}{
		{1, 1, 10, 1},
		{2, 1, 10, 1},
		{3, 1, 10, 2},
		{5, 1, 10, 3},
		{7, 1, 10, 4},
		{9, 1, 10, 5},
		{10, 1, 10, 5},
		{5, 5, 5, 3}, // equal min/max
	}

	for _, tt := range tests {
		got := countToTier(tt.count, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("countToTier(%d, %d, %d) = %d, want %d", tt.count, tt.min, tt.max, got, tt.want)
		}
	}
}

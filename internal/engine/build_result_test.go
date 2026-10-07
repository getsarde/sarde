package engine

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSortWarnings(t *testing.T) {
	ws := []ValidationWarning{
		{File: "b.md", Message: "z"},
		{File: "a.md", Line: 9, Message: "late"},
		{File: "a.md", Line: 2, Col: 5, Message: "col5"},
		{File: "a.md", Line: 2, Col: 1, Field: "title", Message: "t"},
		{File: "a.md", Line: 2, Col: 1, Field: "lint", Message: "l"},
	}
	SortWarnings(ws)
	want := []string{"l", "t", "col5", "late", "z"}
	for i, w := range ws {
		if w.Message != want[i] {
			t.Fatalf("position %d: got %q, want %q (order %v)", i, w.Message, want[i], ws)
		}
	}
}

// TestBuildResult_JSONKeysStable pins the JSON contract consumed by
// cmd/sarde-bench and Sarde Studio: existing keys keep their names and
// types; new fields are additive.
func TestBuildResult_JSONKeysStable(t *testing.T) {
	res := BuildResult{
		PageCount:    3,
		Duration:     1500 * time.Millisecond,
		Warnings:     []ValidationWarning{{File: "a.md", Field: "title", Message: "m", Level: "warning", Line: 4, Col: 2}},
		OutputDir:    "dist",
		LogMessages:  []BuildLogEntry{{Source: "sitemap", Message: "x"}},
		PhaseTimings: []PhaseTiming{{Phase: "Parsing content", Duration: time.Millisecond}},
		Links:        &LinkSummary{Links: 10, Lanes: 1},
		SlowestPages: []PageTiming{{Path: "content/a.md", Duration: time.Millisecond}},
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"PageCount", "Duration", "Warnings", "OutputDir", "PaginatorPages", "Collections",
		"BundleAssets", "PublicFiles", "ProcessedImages", "AliasCount", "SitemapCount",
		"LogMessages", "PhaseTimings", "Links", "SlowestPages",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in %s", key, raw)
		}
	}
	if d, ok := m["Duration"].(float64); !ok || d != float64(1500*time.Millisecond) {
		t.Errorf("Duration must stay an integer nanosecond count, got %v", m["Duration"])
	}
	w := m["Warnings"].([]any)[0].(map[string]any)
	for _, key := range []string{"File", "Field", "Message", "Level", "Line", "Col"} {
		if _, ok := w[key]; !ok {
			t.Errorf("warning missing key %q", key)
		}
	}
}

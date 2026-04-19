package collection

import (
	"testing"
	"time"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestSortPages_DateDesc(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Old", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "New", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Mid", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	SortPages(pages, "date", "desc")
	if pages[0].Title != "New" || pages[1].Title != "Mid" || pages[2].Title != "Old" {
		t.Errorf("got [%s, %s, %s], want [New, Mid, Old]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_DateAsc(t *testing.T) {
	pages := []*engine.Page{
		{Title: "New", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Old", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	SortPages(pages, "date", "asc")
	if pages[0].Title != "Old" {
		t.Errorf("first = %q, want %q", pages[0].Title, "Old")
	}
}

func TestSortPages_WeightAsc(t *testing.T) {
	pages := []*engine.Page{
		{Title: "C", Weight: 3},
		{Title: "A", Weight: 1},
		{Title: "B", Weight: 2},
	}
	SortPages(pages, "weight", "asc")
	if pages[0].Title != "A" || pages[1].Title != "B" || pages[2].Title != "C" {
		t.Errorf("got [%s, %s, %s], want [A, B, C]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_WeightTiebreaker(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Zebra", Weight: 1},
		{Title: "Alpha", Weight: 1},
	}
	SortPages(pages, "weight", "asc")
	if pages[0].Title != "Alpha" {
		t.Errorf("first = %q, want %q (title tiebreaker)", pages[0].Title, "Alpha")
	}
}

func TestSortPages_TitleAsc(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Zebra"},
		{Title: "alpha"},
		{Title: "Beta"},
	}
	SortPages(pages, "title", "asc")
	if pages[0].Title != "alpha" || pages[1].Title != "Beta" || pages[2].Title != "Zebra" {
		t.Errorf("got [%s, %s, %s], want [alpha, Beta, Zebra]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_SlugAsc(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "z-post"},
		{Slug: "a-post"},
	}
	SortPages(pages, "slug", "asc")
	if pages[0].Slug != "a-post" {
		t.Errorf("first slug = %q, want %q", pages[0].Slug, "a-post")
	}
}

func TestSortPages_Empty(t *testing.T) {
	// Should not panic
	SortPages(nil, "date", "asc")
	SortPages([]*engine.Page{}, "date", "asc")
}

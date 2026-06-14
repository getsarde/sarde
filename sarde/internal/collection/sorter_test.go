package collection

import (
	"testing"
	"time"

	"github.com/frostybee/sarde/internal/engine"
)

func TestSortPages_DateDesc(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Old", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{PageIdentity: engine.PageIdentity{Title: "New", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{PageIdentity: engine.PageIdentity{Title: "Mid", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}
	SortPages(pages, "date", "desc")
	if pages[0].Title != "New" || pages[1].Title != "Mid" || pages[2].Title != "Old" {
		t.Errorf("got [%s, %s, %s], want [New, Mid, Old]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_DateAsc(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "New", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{PageIdentity: engine.PageIdentity{Title: "Old", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}
	SortPages(pages, "date", "asc")
	if pages[0].Title != "Old" {
		t.Errorf("first = %q, want %q", pages[0].Title, "Old")
	}
}

func TestSortPages_WeightAsc(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "C"}, Sidebar: engine.PageSidebar{Order: 3}},
		{PageIdentity: engine.PageIdentity{Title: "A"}, Sidebar: engine.PageSidebar{Order: 1}},
		{PageIdentity: engine.PageIdentity{Title: "B"}, Sidebar: engine.PageSidebar{Order: 2}},
	}
	SortPages(pages, "order", "asc")
	if pages[0].Title != "A" || pages[1].Title != "B" || pages[2].Title != "C" {
		t.Errorf("got [%s, %s, %s], want [A, B, C]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_WeightTiebreaker(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Zebra"}, Sidebar: engine.PageSidebar{Order: 1}},
		{PageIdentity: engine.PageIdentity{Title: "Alpha"}, Sidebar: engine.PageSidebar{Order: 1}},
	}
	SortPages(pages, "order", "asc")
	if pages[0].Title != "Alpha" {
		t.Errorf("first = %q, want %q (title tiebreaker)", pages[0].Title, "Alpha")
	}
}

func TestSortPages_TitleAsc(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Zebra"}},
		{PageIdentity: engine.PageIdentity{Title: "alpha"}},
		{PageIdentity: engine.PageIdentity{Title: "Beta"}},
	}
	SortPages(pages, "title", "asc")
	if pages[0].Title != "alpha" || pages[1].Title != "Beta" || pages[2].Title != "Zebra" {
		t.Errorf("got [%s, %s, %s], want [alpha, Beta, Zebra]", pages[0].Title, pages[1].Title, pages[2].Title)
	}
}

func TestSortPages_SlugAsc(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "z-post"}},
		{PageIdentity: engine.PageIdentity{Slug: "a-post"}},
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

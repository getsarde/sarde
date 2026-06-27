package template

import (
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// buildPaginator computes the Paginator for a list page of a paginated collection.
// current is the 1-based index of the page being rendered.
func buildPaginator(col *engine.Collection, current int) *engine.Paginator {
	perPage := col.Config.Paginate
	// Only include rendered pages (exclude section index).
	var pages []*engine.Page
	for _, p := range col.Pages {
		if p.Kind != engine.KindSection {
			pages = append(pages, p)
		}
	}
	total := (len(pages) + perPage - 1) / perPage
	if total < 1 {
		total = 1
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	start := (current - 1) * perPage
	end := start + perPage
	if end > len(pages) {
		end = len(pages)
	}

	base := paginationBaseURL(col)
	p := &engine.Paginator{
		CurrentPages: pages[start:end],
		Current:      current,
		Total:        total,
		TotalItems:   len(pages),
		BaseURL:      base,
		FirstURL:     PaginationURL(base, 1),
		LastURL:      PaginationURL(base, total),
	}
	p.Pages = make([]engine.PaginationLink, 0, total)
	for i := 1; i <= total; i++ {
		p.Pages = append(p.Pages, engine.PaginationLink{
			URL:   PaginationURL(base, i),
			Title: fmt.Sprintf("%d", i),
		})
	}
	if current > 1 {
		p.HasPrev = true
		p.PrevURL = PaginationURL(base, current-1)
	}
	if current < total {
		p.HasNext = true
		p.NextURL = PaginationURL(base, current+1)
	}
	return p
}

// paginationBaseURL returns the index URL for a collection, e.g. "/blog/".
func paginationBaseURL(col *engine.Collection) string {
	if col == nil {
		return "/"
	}
	if col.IndexPage != nil && col.IndexPage.RelPermalink != "" {
		return col.IndexPage.RelPermalink
	}
	return "/" + col.Name + "/"
}

// buildTermPaginator builds the Paginator for a taxonomy term page.
func buildTermPaginator(term *engine.TaxonomyTerm, perPage, current int) *engine.Paginator {
	pages := term.Pages
	total := (len(pages) + perPage - 1) / perPage
	if total < 1 {
		total = 1
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	start := (current - 1) * perPage
	end := start + perPage
	if end > len(pages) {
		end = len(pages)
	}

	base := term.Permalink
	p := &engine.Paginator{
		CurrentPages: pages[start:end],
		Current:      current,
		Total:        total,
		TotalItems:   len(pages),
		BaseURL:      base,
		FirstURL:     PaginationURL(base, 1),
		LastURL:      PaginationURL(base, total),
	}
	p.Pages = make([]engine.PaginationLink, 0, total)
	for i := 1; i <= total; i++ {
		p.Pages = append(p.Pages, engine.PaginationLink{
			URL:   PaginationURL(base, i),
			Title: fmt.Sprintf("%d", i),
		})
	}
	if current > 1 {
		p.HasPrev = true
		p.PrevURL = PaginationURL(base, current-1)
	}
	if current < total {
		p.HasNext = true
		p.NextURL = PaginationURL(base, current+1)
	}
	return p
}

// PaginationURL returns the URL for the Nth pagination page of a collection.
// Page 1 maps to the collection's base URL; N>1 maps to "<base>page/N/".
func PaginationURL(base string, n int) string {
	if n <= 1 {
		return base
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return fmt.Sprintf("%spage/%d/", base, n)
}

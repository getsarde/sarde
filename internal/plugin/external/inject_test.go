package external

import (
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

func TestInjectMatches(t *testing.T) {
	docsPage := &engine.Page{}
	docsPage.Kind = engine.KindPage
	docsPage.Collection = &engine.Collection{Name: "docs"}
	docsPage.Headings = []engine.Heading{{Text: "Intro"}}
	docsRD := &engine.RouteData{Layout: engine.LayoutDocs}

	slidePage := &engine.Page{}
	slidePage.Kind = engine.KindPage
	slidePage.Collection = &engine.Collection{Name: "slides"}
	slideRD := &engine.RouteData{Layout: engine.LayoutPresentation}

	updatedPage := &engine.Page{}
	updatedPage.Kind = engine.KindPage
	updatedPage.Updated = time.Now()

	tests := []struct {
		name string
		inj  InjectConfig
		page *engine.Page
		rd   *engine.RouteData
		want bool
	}{
		{"empty when matches", InjectConfig{}, docsPage, docsRD, true},
		{"always", InjectConfig{When: "always"}, docsPage, docsRD, true},
		{"layout match", InjectConfig{When: "layout", Layout: "presentation"}, slidePage, slideRD, true},
		{"layout mismatch", InjectConfig{When: "layout", Layout: "presentation"}, docsPage, docsRD, false},
		{"collection match", InjectConfig{When: "collection", Collection: "docs"}, docsPage, docsRD, true},
		{"collection mismatch", InjectConfig{When: "collection", Collection: "blog"}, docsPage, docsRD, false},
		{"collection nil on page", InjectConfig{When: "collection", Collection: "docs"}, updatedPage, docsRD, false},
		{"has_sidebar true", InjectConfig{When: "has_sidebar"}, docsPage, docsRD, true},
		{"has_sidebar false", InjectConfig{When: "has_sidebar"}, slidePage, slideRD, false},
		{"has_toc true", InjectConfig{When: "has_toc"}, docsPage, docsRD, true},
		{"has_toc no headings", InjectConfig{When: "has_toc"}, updatedPage, docsRD, false},
		{"has_headings", InjectConfig{When: "has_headings"}, docsPage, docsRD, true},
		{"has_updated true", InjectConfig{When: "has_updated"}, updatedPage, docsRD, true},
		{"has_updated false", InjectConfig{When: "has_updated"}, docsPage, docsRD, false},
		{"is_content_page", InjectConfig{When: "is_content_page"}, docsPage, docsRD, true},
		{"unknown rule never matches", InjectConfig{When: "bogus"}, docsPage, docsRD, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := injectMatches(&tt.inj, tt.page, tt.rd); got != tt.want {
				t.Errorf("injectMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

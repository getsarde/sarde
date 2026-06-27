package template

import (
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/engine"
)

func buildNavFuncs(sitePtr **engine.SiteContext) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"navFor": func(colName string) *engine.NavTree {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			return col.NavTree
		},
		"breadcrumbs": func(data any) []engine.BreadcrumbItem {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return nil
			}
			return rd.Breadcrumbs
		},
		"siblings": func(page *engine.Page) []*engine.Page {
			if page == nil || page.Section == nil {
				return nil
			}
			return page.Section.Pages
		},
		"translations": func(data any) []engine.TranslationLink {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return nil
			}
			return rd.Translations
		},
	}
}

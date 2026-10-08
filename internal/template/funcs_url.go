package template

import (
	htmltemplate "html/template"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// relURLFor is the one template-side wrapper around URLResolver.URL: full
// URLs pass through, and without a resolver the path is returned as written.
// relURL, its per-language override in funcMapForLang, and rootURL all call it.
func relURLFor(r *engine.URLResolver, relPath, lang string) string {
	if r == nil || strings.Contains(relPath, "://") {
		return relPath
	}
	return r.URL(relPath, lang, "")
}

func buildURLFuncs(urlResolverPtr **engine.URLResolver, sitePtr **engine.SiteContext) htmltemplate.FuncMap {
	// rootURL is relURL without the page language: funcMapForLang overrides
	// relURL per language but never rootURL, so files written once at the
	// site root (sitemap.xml, theme fonts) keep their URL on translated pages.
	rootURL := func(relPath string) string {
		return relURLFor(*urlResolverPtr, relPath, "")
	}
	return htmltemplate.FuncMap{
		"absURL": func(relPath string) string {
			if r := *urlResolverPtr; r != nil {
				return r.AbsURL(relPath, "", "")
			}
			s := *sitePtr
			if s == nil || s.BaseURL == "" {
				return relPath
			}
			base := strings.TrimRight(s.BaseURL, "/")
			if strings.HasPrefix(relPath, "/") {
				return base + relPath
			}
			return base + "/" + relPath
		},
		"relURL":  rootURL,
		"rootURL": rootURL,
		"editURL": func(base, relPath string) string {
			if base == "" || relPath == "" {
				return ""
			}
			return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(filepath.ToSlash(relPath), "/")
		},
	}
}

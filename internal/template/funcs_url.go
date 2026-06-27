package template

import (
	htmltemplate "html/template"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

func buildURLFuncs(urlResolverPtr **engine.URLResolver, sitePtr **engine.SiteContext) htmltemplate.FuncMap {
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
		"relURL": func(relPath string) string {
			if strings.Contains(relPath, "://") {
				return relPath
			}
			if r := *urlResolverPtr; r != nil {
				return r.URL(relPath, "", "")
			}
			return relPath
		},
		"editURL": func(base, relPath string) string {
			if base == "" || relPath == "" {
				return ""
			}
			return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(filepath.ToSlash(relPath), "/")
		},
	}
}

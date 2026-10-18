package build

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkrender"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// Frontmatter fields the link check covers: the ones a template writes into
// the page as a URL. The theme writes `image` and `prev.link` / `next.link`
// as they are (no Markdown link resolution), so their values are URLs, not
// .md paths. `prev` / `next` may also name a page of the collection by slug.
const (
	fieldImage = "image"
	fieldPrev  = "prev"
	fieldNext  = "next"
)

// recordFrontmatterRefs adds a LinkRef to graph for every link-like
// frontmatter value of every page, so the link report covers them beside
// body links. It runs once the page index is complete; the values come from
// frontmatter alone, so nothing is cached with the rendered body. Line is 0
// because the frontmatter parser keeps no line per key; Field names the key.
func recordFrontmatterRefs(graph *links.LinkGraph, pages []*engine.Page, idx *content.PageIndex, resolver *engine.URLResolver) {
	for _, page := range pages {
		// Fallback copies of untranslated pages repeat their source's
		// frontmatter; the source page reports it once.
		if page.FilePath == "" || page.IsFallback {
			continue
		}
		if img := strings.TrimSpace(page.Image); img != "" {
			graph.Record(checkFrontmatterImage(page, img, idx, resolver))
		}
		for _, field := range []string{fieldPrev, fieldNext} {
			nav := navOverride(page.Params[field])
			if nav == nil || nav.Disabled {
				continue
			}
			switch {
			case nav.Link != "":
				graph.Record(checkFrontmatterLink(page, field+".link", strings.TrimSpace(nav.Link), idx, resolver))
			case nav.Slug != "":
				graph.Record(checkFrontmatterSlug(page, field, nav.Slug))
			}
		}
	}
}

func navOverride(v any) *engine.NavOverride {
	switch val := v.(type) {
	case *engine.NavOverride:
		return val
	case string:
		if val != "" {
			return &engine.NavOverride{Slug: val}
		}
	}
	return nil
}

// frontmatterRef is the LinkRef shared by every frontmatter check.
func frontmatterRef(page *engine.Page, field, raw string) links.LinkRef {
	collName := ""
	if page.Collection != nil {
		collName = page.Collection.Name
	}
	return links.LinkRef{
		FromPage: page,
		FromFile: page.FilePath,
		Field:    field,
		RawDest:  raw,
		Dim:      links.DimKey{Collection: collName, Lang: page.Lang, Version: page.Version},
		Kind:     links.KindContentRoot,
	}
}

// isExternalURL reports a value with a scheme ("https:", "data:") or a
// protocol-relative "//host" prefix.
func isExternalURL(raw string) bool {
	if strings.HasPrefix(raw, "//") {
		return true
	}
	return linkrender.ClassifyDest(raw).Kind == linkrender.LinkExternal && !strings.HasPrefix(raw, "/")
}

// sitePath turns a value into the site path a browser would request from
// the page: a site-absolute value as it is, a relative one joined to the
// page's URL directory. Query and fragment are split off.
func sitePath(page *engine.Page, raw string) (p, fragment string) {
	p = raw
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p, fragment = p[:i], p[i+1:]
	}
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	if !strings.HasPrefix(p, "/") {
		base := page.RelPermalink
		if !strings.HasSuffix(base, "/") {
			base = path.Dir(base) + "/"
		}
		trailing := strings.HasSuffix(p, "/")
		p = path.Join(base, p)
		if trailing && p != "/" {
			p += "/"
		}
	}
	return p, fragment
}

// checkFrontmatterImage looks for the file an image URL names: a file under
// public/, or a file published beside a page bundle's index.md. A missing
// file is reported as unverified (a warning by default): the image may be
// written by something the build does not index, such as a plugin.
func checkFrontmatterImage(page *engine.Page, raw string, idx *content.PageIndex, resolver *engine.URLResolver) links.LinkRef {
	ref := frontmatterRef(page, fieldImage, raw)
	if isExternalURL(raw) {
		ref.Kind = links.KindExternal
		ref.Status = links.StatusExternal
		return ref
	}
	p, _ := sitePath(page, raw)
	ref.Resolved = p
	if idx.HasAsset(p) || bundleHasFile(p, page, idx, resolver) {
		ref.Status = links.StatusOK
	} else {
		ref.Status = links.StatusUnverified
	}
	return ref
}

// bundleHasFile reports whether a page bundle publishes the file at site
// path p: the page at p's directory is an index.md bundle and the file sits
// beside it.
func bundleHasFile(p string, page *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver) bool {
	dir := path.Dir(p) + "/"
	target := linkrender.LookupInLaneWithDefaultFallback(idx, dir, page, resolver)
	if target == nil || target.FilePath == "" || filepath.Base(target.FilePath) != "index.md" {
		return false
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(target.FilePath), filepath.FromSlash(path.Base(p))))
	return err == nil && !info.IsDir()
}

// checkFrontmatterLink checks a prev / next link the way a browser follows
// it: a page of the site (anchor included), or a static file. A .md path is
// broken, since the theme writes the value into the href unchanged.
func checkFrontmatterLink(page *engine.Page, field, raw string, idx *content.PageIndex, resolver *engine.URLResolver) links.LinkRef {
	ref := frontmatterRef(page, field, raw)
	if isExternalURL(raw) {
		ref.Kind = links.KindExternal
		ref.Status = links.StatusExternal
		return ref
	}
	p, fragment := sitePath(page, raw)
	ref.Fragment = fragment
	if hasMarkdownExt(p) {
		ref.Status = links.StatusBrokenTarget
		return ref
	}
	target := linkrender.LookupInLaneWithDefaultFallback(idx, content.NormalizePermalink(p), page, resolver)
	switch {
	case target != nil:
		ref.TargetPage = target
		ref.Resolved = target.RelPermalink
		ref.Status = links.StatusOK
		if fragment != "" && !idx.HasHeading(target.Permalink, fragment) {
			ref.Status = links.StatusBrokenAnchor
		}
	case p == "/":
		ref.Status = links.StatusOK
	case path.Ext(path.Base(p)) != "":
		// A static file: published from public/ or beside a bundle.
		if idx.HasAsset(p) || bundleHasFile(p, page, idx, resolver) {
			ref.Status = links.StatusOK
		} else {
			ref.Status = links.StatusUnverified
		}
	default:
		ref.Status = links.StatusUnverified
	}
	return ref
}

func hasMarkdownExt(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	return ext == ".md" || ext == ".markdown"
}

// checkFrontmatterSlug checks `prev: some-slug`: the slug must name a page
// of the same collection, which is where navigation looks it up; otherwise
// the override is ignored and the automatic neighbour stays.
func checkFrontmatterSlug(page *engine.Page, field, slug string) links.LinkRef {
	ref := frontmatterRef(page, field, slug)
	ref.Status = links.StatusBrokenTarget
	if page.Collection == nil {
		return ref
	}
	for _, p := range page.Collection.Pages {
		if p.Slug == slug {
			ref.TargetPage = p
			ref.Resolved = p.RelPermalink
			ref.Status = links.StatusOK
			break
		}
	}
	return ref
}

// frontmatterLinkWarnings checks one page's frontmatter links for an
// incremental dev rebuild, which skips the full link report: each problem
// becomes a warning worded like the report's, unless its policy is ignore.
func frontmatterLinkWarnings(page *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver, lvc config.LinkValidationSettings) []engine.ValidationWarning {
	if !config.BoolVal(lvc.Enabled, true) {
		return nil
	}
	graph := links.NewLinkGraph()
	recordFrontmatterRefs(graph, []*engine.Page{page}, idx, resolver)
	var warnings []engine.ValidationWarning
	for _, ref := range graph.Refs() {
		var label, policy string
		switch ref.Status {
		case links.StatusBrokenTarget:
			label, policy = links.FindingBrokenTarget.Label(), lvc.EffectiveOnBroken()
		case links.StatusBrokenAnchor:
			label, policy = links.FindingBrokenAnchor.Label(), lvc.EffectiveOnBrokenAnchor()
		case links.StatusUnverified:
			label, policy = links.FindingUnverifiedInternal.Label(), lvc.EffectiveOnUnverifiedInternal()
		default:
			continue
		}
		if policy == "ignore" || links.ShouldExcludeRef(ref.RawDest, lvc.Exclude) {
			continue
		}
		warnings = append(warnings, engine.ValidationWarning{
			File:    ref.FromFile,
			Field:   "link",
			Message: fmt.Sprintf("%s: %s (frontmatter %s)", label, ref.RawDest, ref.Field),
			Level:   "warn",
		})
	}
	return warnings
}

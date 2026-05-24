package collection

import (
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/navigation"
)

// AnnotateVersions sets Version and VersionRelPath on every page in a versioned
// collection. If last_version is configured, pages matching that version get
// their URLs rewritten to serve at the collection root (no version prefix).
func AnnotateVersions(col *engine.Collection) {
	vc := col.Config.Versioning
	if vc == nil || !vc.Enabled {
		return
	}

	versionIDs := make(map[string]bool, len(vc.Versions))
	for _, v := range vc.Versions {
		versionIDs[v.ID] = true
	}

	for _, page := range col.Pages {
		vid, relPath := extractVersionFromPermalink(page.RelPermalink, col.Name, versionIDs)
		page.Version = vid
		page.VersionRelPath = relPath

		// Rewrite URLs for last_version pages to serve at root (no version prefix).
		if vc.LastVersion != "" && vid == vc.LastVersion {
			page.Permalink = removeVersionSegment(page.Permalink, vid)
			page.RelPermalink = removeVersionSegment(page.RelPermalink, vid)
		}
	}
}

// extractVersionFromPermalink checks whether the first path segment after the
// collection name matches a known version ID.
//
//	"/docs/v1/getting-started/" → ("v1", "getting-started")
//	"/docs/getting-started/"    → ("", "getting-started")
//	"/docs/v1/"                 → ("v1", "")
//	"/docs/"                    → ("", "")
func extractVersionFromPermalink(permalink, collectionName string, versionIDs map[string]bool) (versionID, versionRelPath string) {
	// Strip leading/trailing slashes and collection prefix.
	p := strings.Trim(permalink, "/")
	p = strings.TrimPrefix(p, collectionName)
	p = strings.TrimPrefix(p, "/")

	if p == "" {
		return "", ""
	}

	parts := strings.SplitN(p, "/", 2)
	if versionIDs[parts[0]] {
		versionID = parts[0]
		if len(parts) > 1 {
			versionRelPath = parts[1]
		}
		return versionID, versionRelPath
	}

	return "", p
}

// removeVersionSegment strips a version segment from a permalink.
// "/docs/v2/getting-started/" → "/docs/getting-started/"
// "/docs/v2/"                 → "/docs/"
func removeVersionSegment(permalink, versionID string) string {
	return strings.Replace(permalink, "/"+versionID+"/", "/", 1)
}

// BuildVersionedNavTrees builds a separate NavTree for each version in the
// collection. Returns a map keyed by version ID ("" for unversioned pages).
func BuildVersionedNavTrees(col *engine.Collection) map[string]*engine.NavTree {
	vc := col.Config.Versioning
	if vc == nil || !vc.Enabled {
		return nil
	}

	// Collect distinct version IDs present in pages.
	versionPages := make(map[string][]*engine.Page)
	for _, p := range col.Pages {
		versionPages[p.Version] = append(versionPages[p.Version], p)
	}

	trees := make(map[string]*engine.NavTree, len(versionPages))
	for vid, pages := range versionPages {
		sections := BuildSectionTree(pages, col.Name)
		vCol := &engine.Collection{
			Name:     col.Name,
			Title:    col.Title,
			Config:   col.Config,
			Pages:    pages,
			Sections: sections,
		}

		// Find index page for this version's sub-collection.
		for _, p := range pages {
			if p.Kind == engine.KindSection {
				relDir := sectionDir(p.RelPermalink, col.Name)
				if relDir == "" || relDir == vid {
					vCol.IndexPage = p
					break
				}
			}
		}

		tree := navigation.BuildNavTree(vCol)
		navigation.WirePrevNextFromTree(tree)
		trees[vid] = tree
	}

	return trees
}

// LinkVersions groups pages across versions by their VersionRelPath and
// populates each page's VersionPeers slice, mirroring i18n.LinkTranslations.
func LinkVersions(pages []*engine.Page) {
	// Key: collectionName + ":" + versionRelPath + ":" + lang
	groups := make(map[string][]*engine.Page)
	for _, p := range pages {
		if p.Collection == nil || p.Collection.Config == nil ||
			p.Collection.Config.Versioning == nil || !p.Collection.Config.Versioning.Enabled {
			continue
		}
		if p.VersionRelPath == "" && p.Kind == engine.KindSection {
			continue
		}
		key := p.Collection.Name + ":" + p.VersionRelPath + ":" + p.Lang
		groups[key] = append(groups[key], p)
	}

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, p := range group {
			var peers []*engine.Page
			for _, other := range group {
				if other != p {
					peers = append(peers, other)
				}
			}
			p.VersionPeers = peers
		}
	}
}

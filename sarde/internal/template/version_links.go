package template

import (
	"github.com/getsarde/sarde/internal/engine"
)

func isLastVersion(versionID string, vc *engine.VersionConfig) bool {
	if vc.LastVersion != "" {
		return versionID == vc.LastVersion
	}
	return versionID == ""
}

func versionLabelAndBanner(versionID string, vc *engine.VersionConfig) (label, banner string) {
	for _, vd := range vc.Versions {
		if vd.ID == versionID {
			label = vd.Label
			if label == "" {
				label = vd.ID
			}
			if vd.Banner != "" && vd.Banner != "none" {
				banner = vd.Banner
			}
			return
		}
	}
	// Unversioned pages (versionID == "").
	if versionID == "" {
		// If last_version is set, use that version's label.
		if vc.LastVersion != "" {
			for _, vd := range vc.Versions {
				if vd.ID == vc.LastVersion {
					label = vd.Label
					if label == "" {
						label = vd.ID
					}
					return
				}
			}
		}
		// No last_version — unversioned content is implicitly "Latest".
		label = "Latest"
	}
	return
}

func buildVersionLinks(page *engine.Page, vc *engine.VersionConfig, colName string) []engine.VersionLink {
	links := make([]engine.VersionLink, 0, len(vc.Versions)+1)

	// Build a map of version ID → peer page for quick lookup.
	peerURLs := make(map[string]*engine.Page, len(page.VersionPeers))
	for _, peer := range page.VersionPeers {
		peerURLs[peer.Version] = peer
	}

	// Check if unversioned content exists but isn't in the configured versions list.
	// This happens when LastVersion is empty — unversioned docs are implicitly "Latest".
	hasUnversionedEntry := vc.LastVersion != ""
	for _, vd := range vc.Versions {
		if vd.ID == "" {
			hasUnversionedEntry = true
			break
		}
	}

	// Add "Latest" entry for unversioned content when not covered by last_version.
	if !hasUnversionedEntry {
		link := engine.VersionLink{
			ID:        "",
			Label:     "Latest",
			IsCurrent: page.Version == "",
			IsLatest:  true,
			Banner:    "none",
			Redirect:  "same-page",
		}
		if page.Version == "" {
			link.URL = page.RelPermalink
			link.Title = page.Title
		} else if peer, ok := peerURLs[""]; ok {
			link.URL = peer.RelPermalink
			link.Title = peer.Title
		} else {
			link.URL = "/" + colName + "/"
		}
		links = append(links, link)
	}

	for _, vd := range vc.Versions {
		link := engine.VersionLink{
			ID:        vd.ID,
			Label:     vd.Label,
			IsCurrent: vd.ID == page.Version,
			IsLatest:  isLastVersion(vd.ID, vc),
			Banner:    vd.Banner,
			Redirect:  vd.Redirect,
		}
		if link.Label == "" {
			link.Label = vd.ID
		}

		if vd.ID == page.Version {
			link.URL = page.RelPermalink
			link.Title = page.Title
		} else if vd.Redirect == "root" {
			link.URL = versionRootURL(colName, vd.ID, vc.LastVersion)
		} else if peer, ok := peerURLs[vd.ID]; ok {
			link.URL = peer.RelPermalink
			link.Title = peer.Title
		} else {
			link.URL = versionRootURL(colName, vd.ID, vc.LastVersion)
		}

		links = append(links, link)
	}

	return links
}

func versionRootURL(colName, versionID, lastVersion string) string {
	if versionID == lastVersion || versionID == "" {
		return "/" + colName + "/"
	}
	return "/" + colName + "/" + versionID + "/"
}

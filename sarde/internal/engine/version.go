package engine

// ResolvePageVersion returns the version string to pass to the URL resolver.
// Returns "" for unversioned pages or latest-version pages (produces the
// unprefixed alias URL). Returns the version ID for older versions.
func ResolvePageVersion(page *Page) string {
	if page.Collection == nil || page.Collection.Config == nil {
		return ""
	}
	vc := page.Collection.Config.Versioning
	if vc == nil || !vc.Enabled {
		return ""
	}
	if page.Version == vc.LastVersion {
		return ""
	}
	return page.Version
}

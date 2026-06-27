package content

// ClassifyVersion sets the Version and VersionRelPath on a ContentFile
// based on whether the path segment after the collection name matches a
// configured version ID. Must be called after ClassifyLang.
//
// For single-version sites (versionIDs is nil/empty), this is a no-op.
func ClassifyVersion(cf *ContentFile, versionIDs map[string]map[string]bool) {
	if len(versionIDs) == 0 || cf.CollectionName == "" {
		return
	}

	colVersions, ok := versionIDs[cf.CollectionName]
	if !ok || len(colVersions) == 0 {
		return
	}

	// LangRelPath is already lang-stripped: "docs/v1/guides/auth.md"
	// First segment = collection name, second = potential version.
	rest := cf.LangRelPath
	colSeg, afterCol := splitFirstSegment(rest)
	if colSeg != cf.CollectionName || afterCol == "" {
		return
	}

	verSeg, afterVer := splitFirstSegment(afterCol)
	if !colVersions[verSeg] {
		return
	}

	cf.Version = verSeg
	cf.VersionRelPath = afterVer
}

// VersionFreeRelPath returns a LangRelPath with the version segment removed.
// Used for computing a version-free RelPermalink.
//
//	"docs/v1/guides/auth.md" → "docs/guides/auth.md"
//	"docs/intro.md"          → "docs/intro.md" (unversioned, unchanged)
func VersionFreeRelPath(cf *ContentFile) string {
	if cf.Version == "" {
		return cf.LangRelPath
	}
	colSeg, _ := splitFirstSegment(cf.LangRelPath)
	if cf.VersionRelPath == "" {
		return colSeg
	}
	return colSeg + "/" + cf.VersionRelPath
}

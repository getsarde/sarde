package content

import "strings"

// ClassifyLang sets the Lang, LangRelPath, and adjusts CollectionName on a ContentFile
// based on whether its first path segment matches a configured language code.
//
// For single-language sites (languages is nil/empty), this is a no-op.
func ClassifyLang(cf *ContentFile, languages map[string]bool, defaultLang string) {
	if len(languages) == 0 {
		return
	}

	// Check if first path segment is a language code
	firstSeg, rest := splitFirstSegment(cf.RelPath)

	if languages[firstSeg] && firstSeg != defaultLang {
		// This file is in a non-default language directory
		cf.Lang = firstSeg
		cf.LangRelPath = rest

		// Recompute collection name from the path after the language prefix
		cf.CollectionName = ""
		if rest != "" {
			seg, _ := splitFirstSegment(rest)
			if strings.Contains(rest, "/") {
				cf.CollectionName = seg
			}
		}
	} else {
		// Default language — no language prefix in path
		cf.Lang = defaultLang
		cf.LangRelPath = cf.RelPath
	}
}

// splitFirstSegment splits a forward-slash path into first segment and the rest.
// "fr/docs/getting-started.md" → ("fr", "docs/getting-started.md")
func splitFirstSegment(path string) (string, string) {
	i := strings.IndexByte(path, '/')
	if i < 0 {
		return path, ""
	}
	return path[:i], path[i+1:]
}

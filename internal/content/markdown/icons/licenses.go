package icons

// ---------------------------------------------------------------------------
// License reporting
// ---------------------------------------------------------------------------

// SetLicense is a loaded icon set's license metadata, for attribution.
type SetLicense struct {
	Prefix string
	Title  string
	SPDX   string
	URL    string
}

// LoadedSetLicenses returns license metadata for every loaded collection.
func LoadedSetLicenses() []SetLicense {
	collectionsMu.RLock()
	defer collectionsMu.RUnlock()
	out := make([]SetLicense, 0, len(collections))
	for prefix, col := range collections {
		sl := SetLicense{Prefix: prefix}
		if col.Info != nil && col.Info.License != nil {
			sl.Title = col.Info.License.Title
			sl.SPDX = col.Info.License.SPDX
			sl.URL = col.Info.License.URL
		}
		out = append(out, sl)
	}
	return sortLicenses(out)
}

// UsedSetLicenses returns license metadata for the collections actually
// referenced during the build.
func UsedSetLicenses() []SetLicense {
	collectionsMu.RLock()
	defer collectionsMu.RUnlock()
	var out []SetLicense
	usedSets.Range(func(k, _ any) bool {
		prefix, _ := k.(string)
		if col, ok := collections[prefix]; ok {
			sl := SetLicense{Prefix: prefix}
			if col.Info != nil && col.Info.License != nil {
				sl.Title = col.Info.License.Title
				sl.SPDX = col.Info.License.SPDX
				sl.URL = col.Info.License.URL
			}
			out = append(out, sl)
		}
		return true
	})
	return sortLicenses(out)
}

func sortLicenses(s []SetLicense) []SetLicense {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Prefix < s[j-1].Prefix; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s
}

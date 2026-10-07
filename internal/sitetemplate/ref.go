package sitetemplate

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/getsarde/sarde/internal/download"
)

// Resolved is the Git ref a fetch will ask GitHub for.
type Resolved struct {
	Ref  string
	Kind download.RefKind
	// Pinned marks an engine-derived tag: its cache entry is authoritative,
	// and a missing tag falls back to main.
	Pinned bool
}

// releasePrerelease matches the prerelease labels a real release can carry.
// Anything else (git describe suffixes, snapshot builds) is a dev build.
var releasePrerelease = regexp.MustCompile(`^(alpha|beta|rc)(\.\d+)?$`)

// ResolveRef picks the ref for spec. An explicit "#ref" wins. Third-party
// templates default to main. Official templates follow the engine: release
// builds use the tag "v<major>.<minor>" of the templates repository, so a
// template never relies on engine features the installed version lacks; dev
// builds use main.
func ResolveRef(spec Spec, engineVersion string) Resolved {
	if spec.Ref != "" {
		return Resolved{Ref: spec.Ref, Kind: download.RefAny}
	}
	if !spec.Official {
		return Resolved{Ref: "main", Kind: download.RefBranch}
	}
	v, err := semver.NewVersion(engineVersion)
	if err != nil || (v.Prerelease() != "" && !releasePrerelease.MatchString(v.Prerelease())) {
		return Resolved{Ref: "main", Kind: download.RefBranch}
	}
	return Resolved{Ref: fmt.Sprintf("v%d.%d", v.Major(), v.Minor()), Kind: download.RefTag, Pinned: true}
}

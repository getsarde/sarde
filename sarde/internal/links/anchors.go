package links

import (
	"github.com/getsarde/sarde/internal/engine"
)

// PendingAnchorCheck records a deferred anchor validation. It carries the full
// LinkRef fields needed to write a definitive graph entry after all page
// headings are populated.
type PendingAnchorCheck struct {
	SourceFile      string
	TargetPermalink string
	Fragment        string
	RawHref         string
	FromPage        *engine.Page
	TargetPage      *engine.Page
	Dim             DimKey
	Kind            LinkKind
	Resolved        string
}

// AnchorLookup is the interface ValidateAnchors needs from PageIndex.
// Satisfied by *content.PageIndex without importing the content package.
type AnchorLookup interface {
	HasHeading(permalink, headingID string) bool
}

// ValidateAnchors records a definitive LinkRef for each pending anchor check
// after all page headings are populated. It writes StatusOK or StatusBrokenAnchor
// into graph and returns the failed checks for the builder's warning machinery.
func ValidateAnchors(
	graph *LinkGraph,
	pending []PendingAnchorCheck,
	index AnchorLookup,
) []PendingAnchorCheck {
	var broken []PendingAnchorCheck
	for _, check := range pending {
		status := StatusOK
		if !index.HasHeading(check.TargetPermalink, check.Fragment) {
			status = StatusBrokenAnchor
			broken = append(broken, check)
		}
		graph.Record(LinkRef{
			FromPage:   check.FromPage,
			FromFile:   check.SourceFile,
			RawDest:    check.RawHref,
			Dim:        check.Dim,
			Kind:       check.Kind,
			Resolved:   check.Resolved,
			TargetPage: check.TargetPage,
			Fragment:   check.Fragment,
			Status:     status,
		})
	}
	return broken
}

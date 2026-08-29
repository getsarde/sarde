package linkrender

import (
	"testing"

	"github.com/getsarde/sarde/internal/links"
)

func TestResolveHref_WithoutPositionRecordsZero(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)
	r.ResolveHref("auth.md")
	ref := lastRef(t, graph)
	if ref.Status != links.StatusAmbiguous {
		t.Errorf("bare name.md: status %d, want StatusAmbiguous", ref.Status)
	}
	if ref.Line != 0 || ref.Col != 0 {
		t.Errorf("no-position caller recorded line=%d col=%d, want 0/0", ref.Line, ref.Col)
	}
	r.ResolveHref("./auth.md")
	if got := lastRef(t, graph); got.Status != links.StatusOK {
		t.Errorf("./auth.md: status %d, want StatusOK", got.Status)
	}
}

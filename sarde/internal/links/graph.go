package links

import (
	"sync"

	"github.com/frostybee/sarde/internal/engine"
)

// LinkStatus classifies the outcome of a link resolution attempt.
type LinkStatus int

const (
	StatusOK             LinkStatus = iota // resolved successfully
	StatusBrokenTarget                     // target page not found in index
	StatusBrokenAnchor                     // target page found but heading ID missing
	StatusExternal                         // external URL, unchecked by internal resolver
	StatusExternalBroken                   // external URL probed and returned non-2xx/3xx
)

// DimKey identifies the content dimension a link was resolved in.
type DimKey struct {
	Collection string
	Lang       string
	Version    string
}

// LinkRef records a single link resolution attempt.
type LinkRef struct {
	FromPage   *engine.Page
	FromFile   string
	Line, Col  int
	RawDest    string
	Dim        DimKey
	Kind       LinkKind
	Resolved   string
	TargetPage *engine.Page
	Fragment   string
	Status     LinkStatus
}

// LinkKind classifies a link destination for the link graph.
type LinkKind int

const (
	KindRelative    LinkKind = iota // ./ or ../
	KindContentRoot                // leading / within collection
	KindAnchorOnly                 // #fragment with no path
	KindExternal                   // http(s)://, mailto:, etc.
	KindAmbiguous                  // bare name — rejected
)

// LinkGraph records every link resolution attempt during a build for
// queryable post-build validation. Safe for concurrent use.
type LinkGraph struct {
	mu   sync.Mutex
	refs []LinkRef
}

// NewLinkGraph creates an empty link graph.
func NewLinkGraph() *LinkGraph {
	return &LinkGraph{}
}

// Record appends a link resolution record to the graph.
func (g *LinkGraph) Record(ref LinkRef) {
	g.mu.Lock()
	g.refs = append(g.refs, ref)
	g.mu.Unlock()
}

// Refs returns a copy of all recorded link references.
func (g *LinkGraph) Refs() []LinkRef {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]LinkRef, len(g.refs))
	copy(out, g.refs)
	return out
}

// Len returns the number of recorded link references.
func (g *LinkGraph) Len() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.refs)
}

// BrokenRefs returns only the refs with BrokenTarget or BrokenAnchor status.
func (g *LinkGraph) BrokenRefs() []LinkRef {
	g.mu.Lock()
	defer g.mu.Unlock()
	var broken []LinkRef
	for _, ref := range g.refs {
		if ref.Status == StatusBrokenTarget || ref.Status == StatusBrokenAnchor {
			broken = append(broken, ref)
		}
	}
	return broken
}

// ExternalRefs returns only the refs with External status.
func (g *LinkGraph) ExternalRefs() []LinkRef {
	g.mu.Lock()
	defer g.mu.Unlock()
	var ext []LinkRef
	for _, ref := range g.refs {
		if ref.Status == StatusExternal {
			ext = append(ext, ref)
		}
	}
	return ext
}

// MarkExternalBroken transitions all refs whose RawDest equals url
// from StatusExternal to StatusExternalBroken.
func (g *LinkGraph) MarkExternalBroken(url string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.refs {
		if g.refs[i].RawDest == url && g.refs[i].Status == StatusExternal {
			g.refs[i].Status = StatusExternalBroken
		}
	}
}

package engine

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

// NavTree represents a complete sidebar navigation tree for a collection.
type NavTree struct {
	Root       *NavNode
	Flat       []*NavNode
	TotalPages int
	MaxDepth   int
	Hash       string
}

// NavNode is a single entry in the sidebar navigation tree.
type NavNode struct {
	Label       string
	URL         string
	Slug        string
	Order       int
	Position    int
	Children    []*NavNode
	Parent      *NavNode
	Depth       int
	IsActive    bool
	IsOpen      bool
	HasActive   bool
	DefaultOpen bool
	GroupIndex  int
	Page        *Page
	Attrs       map[string]string
	Icon        string
	Badge       Badge
	Description string
}

// GlobalNav represents the top-level site navigation bar.
type GlobalNav struct {
	Items []GlobalNavItem
}

// GlobalNavItem is a single entry in the global navigation bar.
type GlobalNavItem struct {
	Label      string
	URL        string
	Collection string
	IsActive   bool
	External   bool
}

// BreadcrumbItem is a single entry in a breadcrumb trail.
type BreadcrumbItem struct {
	Label   string
	URL     string
	Current bool
}

// PaginationLinks holds prev/next page references.
type PaginationLinks struct {
	Prev *PaginationLink
	Next *PaginationLink
}

// PaginationLink is a reference to a prev or next page.
type PaginationLink struct {
	URL   string
	Title string
}

// Paginator holds numbered list-page pagination state for collection index pages.
type Paginator struct {
	Pages        []PaginationLink // Numbered links (one per page of results)
	CurrentPages []*Page          // Slice of content pages visible on this pagination page
	Current      int              // 1-based index of the current page
	Total        int              // Total number of pagination pages
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
	TotalItems   int    // Total content items across all pagers
	BaseURL      string // Collection base URL for constructing custom pagination links
	FirstURL     string // Permalink to the first pagination page
	LastURL      string // Permalink to the last pagination page
}

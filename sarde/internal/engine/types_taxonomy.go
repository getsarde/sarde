package engine

// ---------------------------------------------------------------------------
// Taxonomy
// ---------------------------------------------------------------------------

// Taxonomy represents a grouping dimension (tags, categories, authors, etc.).
type Taxonomy struct {
	Name       string
	Singular   string
	Terms      map[string]*TaxonomyTerm
	Permalink  string
	PaginateBy int // 0 = no pagination for term listing pages
}

// TaxonomyTerm is a single term within a taxonomy with its associated pages.
type TaxonomyTerm struct {
	Name        string
	Slug        string
	CustomSlug  string // from permalink field in data/*.yml; overrides Slugify(Name)
	Permalink   string
	Pages       []*Page
	Label       string
	Description string
	Color       string
	Icon        string
	Hidden      bool
	Priority    int
	Difficulty  string // beginner, intermediate, advanced
	ContentType string // lecture, lab, assignment, project, reference, tutorial, assessment
}

// TermEntry wraps a TaxonomyTerm with computed tag-cloud data.
type TermEntry struct {
	*TaxonomyTerm
	Count   int
	PopTier int // 1-5 popularity quintile
}

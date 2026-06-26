package build

import (
	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/links"
	"github.com/getsarde/sarde/internal/shortcode"
)

// buildState carries all intermediate results between pipeline phases for a
// single Build() invocation. Allocated at the top of Build() and passed by
// pointer to each phase method.
type buildState struct {
	recordTiming func(string)

	// Phase 1 — Initialize
	contentDir  string
	outputDir   string
	parallel    bool
	workerCount int
	isMultiLang bool
	defaultLang string
	stringTable *i18n.StringTable

	// Phase 2/3 — Discover + Parse
	files       []content.ContentFile
	collections map[string]*engine.Collection
	standalones []*engine.Page
	warnings    []engine.ValidationWarning

	// Phase 4 — Assemble
	allPages   []*engine.Page
	taxonomies map[string]*engine.Taxonomy
	taxByLang  map[string]map[string]*engine.Taxonomy
	siteCtx    *engine.SiteContext

	// Phase 4.5 — Assets + Markdown render
	assetPipeline  *asset.Pipeline
	pageIndex      *content.PageIndex
	scProcessor    *shortcode.Processor
	shortcodesHash string
	iconRenderKey  string
	pageCache      *PageCache
	pendingAnchors []links.PendingAnchorCheck
	validationData map[string]engine.ValidationEntry

	// Set by phaseAssets when b.checkOnly is true; Build() returns it directly.
	checkResult *engine.BuildResult

	// Phase 5 — Render
	rendered       []RenderedPage
	aliases        map[string]string
	paginatorPages int
	syntheticPages []*engine.Page
	taxonomyPages  []*engine.Page
}

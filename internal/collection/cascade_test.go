package collection

import (
	"testing"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

func TestApplyCascade_Basic(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs", "toc": map[string]any{"enabled": true}}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs, got %v", child.Params["layout"])
	}
	if child.TOC.Enabled == nil || *child.TOC.Enabled != true {
		t.Errorf("expected TOC.Enabled=true, got %v", child.TOC.Enabled)
	}
}

func TestApplyCascade_ChildOverrides(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs"}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{"layout": "wide"},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "wide" {
		t.Errorf("child's own layout should win, got %v", child.Params["layout"])
	}
}

func TestApplyCascade_MultiLevel(t *testing.T) {
	grandparentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs", "toc": map[string]any{"enabled": false}}},
	}
	parentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"toc": map[string]any{"enabled": true}}},
	}
	grandparent := &engine.Section{IndexPage: grandparentIdx}
	parent := &engine.Section{IndexPage: parentIdx, Parent: grandparent}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: parent},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs from grandparent, got %v", child.Params["layout"])
	}
	if child.TOC.Enabled == nil || *child.TOC.Enabled != true {
		t.Errorf("expected TOC.Enabled=true from parent (overrides grandparent), got %v", child.TOC.Enabled)
	}
}

func TestApplyCascade_ParamsMerge(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params: map[string]any{
			consts.CascadeKey: map[string]any{
				"params": map[string]any{"author": "Team", "color": "blue"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{"color": "red"},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["author"] != "Team" {
		t.Errorf("expected author=Team from cascade params, got %v", child.Params["author"])
	}
	if child.Params["color"] != "red" {
		t.Errorf("child's own color should win, got %v", child.Params["color"])
	}
}

func TestApplyCascade_SectionPageFromParent(t *testing.T) {
	parentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"toc": map[string]any{"enabled": true}}},
	}
	parent := &engine.Section{IndexPage: parentIdx}
	childIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{},
	}
	childSec := &engine.Section{IndexPage: childIdx, Parent: parent}
	childIdx.PageRelationships.Section = childSec

	ApplyCascade([]*engine.Page{childIdx})

	if childIdx.TOC.Enabled == nil || *childIdx.TOC.Enabled != true {
		t.Errorf("section _index.md should receive TOC.Enabled=true from parent, got %v", childIdx.TOC.Enabled)
	}
}

func TestApplyCascade_NoCascade(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if len(child.Params) != 0 {
		t.Errorf("expected no params change, got %v", child.Params)
	}
}

func TestApplyCascade_SidebarLabel(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"sidebar": map[string]any{"label": "Guides"}}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Sidebar.Label != "Guides" {
		t.Errorf("expected Sidebar.Label=Guides, got %q", child.Sidebar.Label)
	}
}

func TestApplyCascade_Banner(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params: map[string]any{
			consts.CascadeKey: map[string]any{
				"banner": map[string]any{"content": "Under review", "variant": "caution"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	b, ok := child.Params["banner"].(*engine.PageBanner)
	if !ok || b == nil {
		t.Fatalf("expected *PageBanner, got %T", child.Params["banner"])
	}
	if b.Content != "Under review" {
		t.Errorf("expected content='Under review', got %q", b.Content)
	}
	if b.Variant != "caution" {
		t.Errorf("expected variant=caution, got %q", b.Variant)
	}
}

// lanePage builds a page with a file in the given lane, for the per-lane
// cascade tests.
func lanePage(rel, lang, version string, kind engine.NodeKind, cascade map[string]any) *engine.Page {
	p := &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: rel, Kind: kind, FilePath: lang + version + rel + "index.md"},
		PageI18n:       engine.PageI18n{Lang: lang},
		PageVersioning: engine.PageVersioning{Version: version},
		Params:         map[string]any{},
	}
	if cascade != nil {
		p.Params[consts.CascadeKey] = cascade
	}
	return p
}

func authorCascade(name string) map[string]any {
	return map[string]any{"params": map[string]any{"author": name}}
}

// Each language inherits from its own _index.md, whichever of the two would
// win the folder's slot in the shared tree.
func TestApplyCascadeByLane_PerLanguage(t *testing.T) {
	enIdx := lanePage("/docs/guide/", "en", "", engine.KindSection, authorCascade("Velox Team"))
	frIdx := lanePage("/docs/guide/", "fr", "", engine.KindSection, authorCascade("Equipe Velox"))
	enPage := lanePage("/docs/guide/intro/", "en", "", engine.KindPage, nil)
	frPage := lanePage("/docs/guide/intro/", "fr", "", engine.KindPage, nil)
	pages := []*engine.Page{enIdx, frIdx, enPage, frPage}
	BuildSectionTree(pages, "docs")

	ApplyCascadeByLane(pages, pages, "docs", "en")

	if got := enPage.Params["author"]; got != "Velox Team" {
		t.Errorf("en page author = %v, want Velox Team", got)
	}
	if got := frPage.Params["author"]; got != "Equipe Velox" {
		t.Errorf("fr page author = %v, want Equipe Velox", got)
	}
}

// A French page in a folder with no French _index.md inherits the default
// language's cascade, as the i18n fallback would show it.
func TestApplyCascadeByLane_BorrowsDefaultLanguageIndex(t *testing.T) {
	enRoot := lanePage("/docs/", "en", "", engine.KindSection, map[string]any{"banner": map[string]any{"content": "Beta"}})
	enIdx := lanePage("/docs/guide/", "en", "", engine.KindSection, authorCascade("Velox Team"))
	frRoot := lanePage("/docs/", "fr", "", engine.KindSection, map[string]any{"banner": map[string]any{"content": "Bêta"}})
	frPage := lanePage("/docs/guide/intro/", "fr", "", engine.KindPage, nil)
	pages := []*engine.Page{enRoot, enIdx, frRoot, frPage}
	BuildSectionTree(pages, "docs")

	ApplyCascadeByLane(pages, pages, "docs", "en")

	if got := frPage.Params["author"]; got != "Velox Team" {
		t.Errorf("fr page author = %v, want the borrowed Velox Team", got)
	}
	banner, _ := frPage.Params["banner"].(*engine.PageBanner)
	if banner == nil || banner.Content != "Bêta" {
		t.Errorf("fr page banner = %+v, want the French root's Bêta", banner)
	}
}

// Version lanes never mix and never borrow from each other.
func TestApplyCascadeByLane_VersionLanesDoNotMix(t *testing.T) {
	v2Idx := lanePage("/docs/guide/", "", "v2", engine.KindSection, authorCascade("v2 team"))
	v1Idx := lanePage("/docs/guide/", "", "v1", engine.KindSection, authorCascade("v1 team"))
	v2Page := lanePage("/docs/guide/intro/", "", "v2", engine.KindPage, nil)
	v1Page := lanePage("/docs/guide/intro/", "", "v1", engine.KindPage, nil)
	v2Root := lanePage("/docs/", "", "v2", engine.KindSection, map[string]any{"layout": "wide"})
	pages := []*engine.Page{v2Root, v2Idx, v1Idx, v2Page, v1Page}
	BuildSectionTree(pages, "docs")

	ApplyCascadeByLane(pages, pages, "docs", "")

	if got := v2Page.Params["author"]; got != "v2 team" {
		t.Errorf("v2 page author = %v, want v2 team", got)
	}
	if got := v1Page.Params["author"]; got != "v1 team" {
		t.Errorf("v1 page author = %v, want v1 team", got)
	}
	if _, ok := v1Page.Params["layout"]; ok {
		t.Errorf("v1 page took the v2 root's cascade: layout = %v", v1Page.Params["layout"])
	}
}

// The per-lane trees are scratch: every page keeps the Section it had.
func TestApplyCascadeByLane_RestoresSections(t *testing.T) {
	enIdx := lanePage("/docs/guide/", "en", "", engine.KindSection, authorCascade("Velox Team"))
	frPage := lanePage("/docs/guide/intro/", "fr", "", engine.KindPage, nil)
	enPage := lanePage("/docs/guide/intro/", "en", "", engine.KindPage, nil)
	pages := []*engine.Page{enIdx, frPage, enPage}
	BuildSectionTree(pages, "docs")
	before := map[*engine.Page]*engine.Section{}
	for _, p := range pages {
		before[p] = p.Section
	}

	ApplyCascadeByLane(pages, pages, "docs", "en")

	for _, p := range pages {
		if p.Section != before[p] {
			t.Errorf("%s (%s): Section changed", p.RelPermalink, p.Lang)
		}
	}
}

// Pages without a file (a synthetic index) neither give nor take cascade.
func TestApplyCascadeByLane_IgnoresPagesWithoutFile(t *testing.T) {
	synthetic := lanePage("/docs/", "fr", "", engine.KindSection, authorCascade("ghost"))
	synthetic.FilePath = ""
	enRoot := lanePage("/docs/", "en", "", engine.KindSection, authorCascade("Velox Team"))
	frPage := lanePage("/docs/intro/", "fr", "", engine.KindPage, nil)
	pages := []*engine.Page{synthetic, enRoot, frPage}
	BuildSectionTree(pages, "docs")

	ApplyCascadeByLane(pages, pages, "docs", "en")

	if got := frPage.Params["author"]; got != "Velox Team" {
		t.Errorf("fr page author = %v, want the borrowed Velox Team (the synthetic index has no file)", got)
	}
}

// Only targets are filled; the rest of the collection only supplies the
// section indexes.
func TestApplyCascadeByLane_OnlyTargetsChange(t *testing.T) {
	idx := lanePage("/docs/", "", "", engine.KindSection, authorCascade("Velox Team"))
	target := lanePage("/docs/a/", "", "", engine.KindPage, nil)
	other := lanePage("/docs/b/", "", "", engine.KindPage, nil)
	pages := []*engine.Page{idx, target, other}
	BuildSectionTree(pages, "docs")

	ApplyCascadeByLane(pages, []*engine.Page{target}, "docs", "")

	if got := target.Params["author"]; got != "Velox Team" {
		t.Errorf("target author = %v, want Velox Team", got)
	}
	if _, ok := other.Params["author"]; ok {
		t.Errorf("non-target page was filled: author = %v", other.Params["author"])
	}
}

// ApplyPageDefaults detects a multi-course labs collection from every page,
// so a course-level page passed alone keeps the default layout.
func TestApplyPageDefaults_LabsMultiCourseFromAllPages(t *testing.T) {
	course := lanePage("/labs/web/", "", "", engine.KindSection, nil)
	lab := lanePage("/labs/web/build/", "", "", engine.KindSection, nil)
	step := lanePage("/labs/web/build/scaffold/", "", "", engine.KindPage, nil)
	courseNote := lanePage("/labs/web/syllabus/", "", "", engine.KindPage, nil)
	pages := []*engine.Page{course, lab, step, courseNote}
	BuildSectionTree(pages, "labs")

	ApplyPageDefaults("labs", pages, []*engine.Page{courseNote}, "")
	if _, ok := courseNote.Params["layout"]; ok {
		t.Errorf("course-level page got layout %v; multi-course was misdetected", courseNote.Params["layout"])
	}
	ApplyPageDefaults("labs", pages, []*engine.Page{step}, "")
	if got := step.Params["layout"]; got != string(engine.LayoutLabs) {
		t.Errorf("lab step layout = %v, want labs", got)
	}
}

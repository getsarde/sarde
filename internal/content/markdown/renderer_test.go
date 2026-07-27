package markdown

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestRender_BasicMarkdown(t *testing.T) {
	r := NewRenderer()
	result, err := r.Render("Hello **world**!")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, "<strong>world</strong>") {
		t.Errorf("expected <strong>, got: %s", result.HTML)
	}
}

func TestRender_GFM_Table(t *testing.T) {
	r := NewRenderer()
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Errorf("expected <table>, got: %s", result.HTML)
	}
}

func TestRender_GFM_TaskList(t *testing.T) {
	r := NewRenderer()
	md := "- [x] Done\n- [ ] Todo\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "checked") {
		t.Errorf("expected checked checkbox, got: %s", result.HTML)
	}
}

func TestRender_GFM_Strikethrough(t *testing.T) {
	r := NewRenderer()
	result, err := r.Render("~~deleted~~")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<del>deleted</del>") {
		t.Errorf("expected <del>, got: %s", result.HTML)
	}
}

func TestRender_Footnote(t *testing.T) {
	r := NewRenderer()
	md := "Text[^1]\n\n[^1]: Footnote content\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "footnote") {
		t.Errorf("expected footnote, got: %s", result.HTML)
	}
}

func TestRender_HeadingExtraction(t *testing.T) {
	r := NewRenderer()
	md := "## First Heading\n\nSome text.\n\n### Sub Heading\n\nMore text.\n\n## Another H2\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Headings) != 3 {
		t.Fatalf("headings = %d, want 3", len(result.Headings))
	}

	if result.Headings[0].Level != 2 || result.Headings[0].Text != "First Heading" {
		t.Errorf("headings[0] = %+v, want Level=2 Text='First Heading'", result.Headings[0])
	}
	if result.Headings[1].Level != 3 || result.Headings[1].Text != "Sub Heading" {
		t.Errorf("headings[1] = %+v, want Level=3 Text='Sub Heading'", result.Headings[1])
	}
	if result.Headings[2].Level != 2 || result.Headings[2].Text != "Another H2" {
		t.Errorf("headings[2] = %+v, want Level=2 Text='Another H2'", result.Headings[2])
	}

	// Verify IDs are injected
	if !strings.Contains(result.HTML, `id="first-heading"`) {
		t.Errorf("expected id='first-heading' in HTML")
	}
	if !strings.Contains(result.HTML, `id="sub-heading"`) {
		t.Errorf("expected id='sub-heading' in HTML")
	}
}

func TestRender_HeadingIDCollision(t *testing.T) {
	r := NewRenderer()
	md := "## Same Title\n\n## Same Title\n\n## Same Title\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 3 {
		t.Fatalf("headings = %d, want 3", len(result.Headings))
	}
	ids := map[string]bool{}
	for _, h := range result.Headings {
		if ids[h.ID] {
			t.Errorf("duplicate heading ID: %q", h.ID)
		}
		ids[h.ID] = true
	}
}

func TestRender_HeadingAnchorLink(t *testing.T) {
	r := NewRenderer()
	md := "## My Section\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `class="sarde-heading-anchor"`) {
		t.Errorf("expected heading-anchor, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `href="#my-section"`) {
		t.Errorf("expected anchor href, got: %s", result.HTML)
	}
}

func TestRender_CustomHeadingID(t *testing.T) {
	r := NewRenderer()
	md := "## My Section {#custom-id}\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `id="custom-id"`) {
		t.Errorf("expected custom id preserved, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, `id="my-section"`) {
		t.Errorf("slugified id should not replace the custom id, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `href="#custom-id"`) {
		t.Errorf("anchor link should use the custom id, got: %s", result.HTML)
	}
	if len(result.Headings) != 1 || result.Headings[0].ID != "custom-id" {
		t.Errorf("Headings should carry the custom id, got: %+v", result.Headings)
	}
}

func TestRender_CustomHeadingIDCollision(t *testing.T) {
	r := NewRenderer()
	// The custom id occupies "shared"; the later heading's slug collides
	// with it and must be bumped by the counter.
	md := "## Alpha {#shared}\n\n## Shared\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(result.Headings))
	}
	if result.Headings[0].ID != "shared" {
		t.Errorf("first heading id = %q, want %q", result.Headings[0].ID, "shared")
	}
	if result.Headings[1].ID != "shared-1" {
		t.Errorf("second heading id = %q, want %q", result.Headings[1].ID, "shared-1")
	}
}

func TestRender_HeadingLinksDisabled(t *testing.T) {
	r := NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: defaultBlockedHrefSchemes,
		HeadingLinks:       false,
	})
	md := "## My Section\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.HTML, "sarde-heading-anchor") {
		t.Errorf("heading_links: false must not inject anchor links, got: %s", result.HTML)
	}
	// IDs are still assigned: the TOC, search index, and anchor validation
	// depend on them regardless of the anchor-link setting.
	if !strings.Contains(result.HTML, `id="my-section"`) {
		t.Errorf("heading id must still be assigned, got: %s", result.HTML)
	}
	if len(result.Headings) != 1 || result.Headings[0].ID != "my-section" {
		t.Errorf("Headings must still be extracted, got: %+v", result.Headings)
	}
}

func TestRender_H1NotInTOC(t *testing.T) {
	r := NewRenderer()
	md := "# Title\n\n## Section\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range result.Headings {
		if h.Level == 1 {
			t.Error("h1 should not be in headings with default config (min=2, max=4)")
		}
	}
}

func TestRender_HeadingLevelRange_H5H6(t *testing.T) {
	r := NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: defaultBlockedHrefSchemes,
		HeadingLinks:       true,
		HeadingMinLevel:    2,
		HeadingMaxLevel:    6,
	})
	md := "## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 5 {
		t.Fatalf("headings = %d, want 5 (h2-h6)", len(result.Headings))
	}
	for i, want := range []int{2, 3, 4, 5, 6} {
		if result.Headings[i].Level != want {
			t.Errorf("headings[%d].Level = %d, want %d", i, result.Headings[i].Level, want)
		}
	}
	if !strings.Contains(result.HTML, `id="h5"`) {
		t.Error("h5 heading should have an injected id")
	}
	if !strings.Contains(result.HTML, `id="h6"`) {
		t.Error("h6 heading should have an injected id")
	}
}

func TestRender_HeadingLevelRange_H1Included(t *testing.T) {
	r := NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: defaultBlockedHrefSchemes,
		HeadingLinks:       true,
		HeadingMinLevel:    1,
		HeadingMaxLevel:    4,
	})
	md := "# Title\n\n## Section\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 2 {
		t.Fatalf("headings = %d, want 2 (h1+h2)", len(result.Headings))
	}
	if result.Headings[0].Level != 1 {
		t.Errorf("headings[0].Level = %d, want 1", result.Headings[0].Level)
	}
}

func TestRender_HeadingLevelRange_NarrowRange(t *testing.T) {
	r := NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: defaultBlockedHrefSchemes,
		HeadingLinks:       true,
		HeadingMinLevel:    3,
		HeadingMaxLevel:    5,
	})
	md := "## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 3 {
		t.Fatalf("headings = %d, want 3 (h3, h4, h5)", len(result.Headings))
	}
	for i, want := range []int{3, 4, 5} {
		if result.Headings[i].Level != want {
			t.Errorf("headings[%d].Level = %d, want %d", i, result.Headings[i].Level, want)
		}
	}
	if strings.Contains(result.HTML, `id="h2"`) {
		t.Error("h2 should NOT have an id when minLevel=3")
	}
	if strings.Contains(result.HTML, `id="h6"`) {
		t.Error("h6 should NOT have an id when maxLevel=5")
	}
}

func TestRender_HeadingLevelRange_DefaultMatchesOldBehavior(t *testing.T) {
	rDefault := NewRenderer()
	rExplicit := NewRendererFromConfig(RendererConfig{
		BlockedHrefSchemes: defaultBlockedHrefSchemes,
		HeadingLinks:       true,
		HeadingMinLevel:    2,
		HeadingMaxLevel:    4,
	})
	md := "# H1\n\n## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6\n"

	res1, _ := rDefault.Render(md)
	res2, _ := rExplicit.Render(md)

	if len(res1.Headings) != len(res2.Headings) {
		t.Fatalf("default headings=%d, explicit headings=%d", len(res1.Headings), len(res2.Headings))
	}
	for i := range res1.Headings {
		if res1.Headings[i].Level != res2.Headings[i].Level || res1.Headings[i].ID != res2.Headings[i].ID {
			t.Errorf("heading[%d] mismatch: default=%+v explicit=%+v", i, res1.Headings[i], res2.Headings[i])
		}
	}
}

func TestRender_Aside(t *testing.T) {
	r := NewRenderer()
	md := ":::note\nThis is a note.\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "sarde-aside") {
		t.Errorf("expected aside element, got: %s", result.HTML)
	}
}

func TestRender_CodeBlockSyntaxHighlighting(t *testing.T) {
	r := NewRenderer()
	md := "```go\nfunc main() {}\n```\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	// Should have syntax highlighting classes
	if !strings.Contains(result.HTML, "kz-") && !strings.Contains(result.HTML, "<pre") {
		if !strings.Contains(result.HTML, "<code") {
			t.Errorf("expected code block, got: %s", result.HTML)
		}
	}
}

func TestRender_Badge(t *testing.T) {
	r := NewRenderer()
	md := ":::badge(type=\"success\")\nNew\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "sarde-badge") {
		t.Errorf("expected badge, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "sarde-badge-icon") {
		t.Errorf("expected inline Lucide SVG icon in badge, got: %s", result.HTML)
	}
}

func TestRender_Kbd(t *testing.T) {
	r := NewRenderer()
	md := "Press ::kbd[Ctrl+S] to save.\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<kbd") {
		t.Errorf("expected <kbd>, got: %s", result.HTML)
	}
}

func TestRender_Highlight(t *testing.T) {
	r := NewRenderer()
	md := "This is ==highlighted== text.\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<mark") {
		t.Errorf("expected <mark>, got: %s", result.HTML)
	}
}

func TestRender_Tabs(t *testing.T) {
	r := NewRenderer()
	md := ":::tabs\n== Tab 1\nContent 1\n== Tab 2\nContent 2\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "tablist") || !strings.Contains(result.HTML, "tabpanel") {
		t.Errorf("expected tablist/tabpanel, got: %s", result.HTML)
	}
	// The markdown above has no blank lines between markers, so both labels
	// and both bodies must survive the paragraph split.
	for _, want := range []string{`data-tab-label="Tab 1"`, `data-tab-label="Tab 2"`, "Content 1", "Content 2"} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("expected %s, got: %s", want, result.HTML)
		}
	}
	if strings.Contains(result.HTML, "== Tab 2") {
		t.Errorf("literal marker leaked into output: %s", result.HTML)
	}
}

func TestRender_Steps(t *testing.T) {
	r := NewRenderer()
	md := ":::steps\n### Step 1\nDo this.\n### Step 2\nDo that.\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "sarde-steps") {
		t.Errorf("expected steps, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `<h3 class="sarde-step-title"`) {
		t.Errorf("expected h3 step title, got: %s", result.HTML)
	}
}

func TestRender_Steps_H2(t *testing.T) {
	r := NewRenderer()
	md := ":::steps\n## Step 1\nDo this.\n## Step 2\nDo that.\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "sarde-steps") {
		t.Errorf("expected steps container, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `<h2 class="sarde-step-title"`) {
		t.Errorf("expected h2 step title, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `data-step="2"`) {
		t.Errorf("expected two steps, got: %s", result.HTML)
	}
}

func TestRender_Details(t *testing.T) {
	r := NewRenderer()
	md := ":::details[Click to expand]\nHidden content.\n:::\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<details") {
		t.Errorf("expected <details>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<summary") {
		t.Errorf("expected <summary>, got: %s", result.HTML)
	}
}

func TestRender_MathInline(t *testing.T) {
	r := NewRenderer()
	md := "Inline math $E = mc^2$ here.\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "math") {
		t.Errorf("expected math class, got: %s", result.HTML)
	}
}

func TestRender_EmptyInput(t *testing.T) {
	r := NewRenderer()
	result, err := r.Render("")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headings) != 0 {
		t.Errorf("headings = %d, want 0", len(result.Headings))
	}
}

func TestRender_ImplementsInterface(t *testing.T) {
	var _ engine.MarkdownRenderer = NewRenderer()
	r := NewRenderer()
	_, err := r.Render("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRender_HasCodeBlocks(t *testing.T) {
	r := NewRenderer()

	result, err := r.Render("```go\nfunc main() {}\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasCodeBlocks {
		t.Error("expected HasCodeBlocks=true for fenced code block")
	}

	result, err = r.Render("Hello world\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.HasCodeBlocks {
		t.Error("expected HasCodeBlocks=false for plain text")
	}
}

func TestRender_HasImages(t *testing.T) {
	r := NewRenderer()

	result, err := r.Render("![alt](image.png)\n")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasImages {
		t.Error("expected HasImages=true for image")
	}

	result, err = r.Render("Hello world\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.HasImages {
		t.Error("expected HasImages=false for plain text")
	}
}

func TestRenderer_Fingerprint_Stable(t *testing.T) {
	cfg := RendererConfig{HeadingLinks: true}
	r1 := NewRendererFromConfig(cfg)
	r2 := NewRendererFromConfig(cfg)
	if r1.Fingerprint() != r2.Fingerprint() {
		t.Errorf("same config produced different fingerprints:\n  %s\n  %s", r1.Fingerprint(), r2.Fingerprint())
	}
	if len(r1.Fingerprint()) != 64 {
		t.Errorf("fingerprint length = %d, want 64 (hex SHA-256)", len(r1.Fingerprint()))
	}
}

func TestRenderer_Fingerprint_ChangesWithHeadingLinks(t *testing.T) {
	r1 := NewRendererFromConfig(RendererConfig{HeadingLinks: true})
	r2 := NewRendererFromConfig(RendererConfig{HeadingLinks: false})
	if r1.Fingerprint() == r2.Fingerprint() {
		t.Error("different HeadingLinks values should produce different fingerprints")
	}
}

func TestRenderer_Fingerprint_ChangesWithBlockedSchemes(t *testing.T) {
	r1 := NewRendererFromConfig(RendererConfig{BlockedHrefSchemes: []string{"javascript:"}})
	r2 := NewRendererFromConfig(RendererConfig{BlockedHrefSchemes: []string{"javascript:", "data:"}})
	if r1.Fingerprint() == r2.Fingerprint() {
		t.Error("different BlockedHrefSchemes should produce different fingerprints")
	}
}

func TestRender_SoftLineBreakIsSpaceByDefault(t *testing.T) {
	r := NewRenderer()
	result, err := r.Render("First line\nsecond line.")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.HTML, "<br") {
		t.Errorf("a soft line break must not render a break tag, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "First line\nsecond line.") {
		t.Errorf("expected both lines in one paragraph, got: %s", result.HTML)
	}
}

func TestRender_HardWrapsEnabled(t *testing.T) {
	r := NewRendererFromConfig(RendererConfig{HardWraps: true})
	result, err := r.Render("First line\nsecond line.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<br") {
		t.Errorf("HardWraps should render a break tag, got: %s", result.HTML)
	}
}

// Explicit breaks are independent of HardWraps: two trailing spaces or a
// trailing backslash must still break the line when soft wraps are in effect.
func TestRender_ExplicitLineBreaksSurviveSoftWraps(t *testing.T) {
	for name, md := range map[string]string{
		"trailing spaces":    "First line  \nsecond line.",
		"trailing backslash": "First line\\\nsecond line.",
	} {
		t.Run(name, func(t *testing.T) {
			r := NewRenderer()
			result, err := r.Render(md)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(result.HTML, "<br") {
				t.Errorf("expected an explicit break tag, got: %s", result.HTML)
			}
		})
	}
}

func TestRenderer_Fingerprint_ChangesWithHardWraps(t *testing.T) {
	r1 := NewRendererFromConfig(RendererConfig{HardWraps: true})
	r2 := NewRendererFromConfig(RendererConfig{HardWraps: false})
	if r1.Fingerprint() == r2.Fingerprint() {
		t.Error("different HardWraps values should produce different fingerprints")
	}
}

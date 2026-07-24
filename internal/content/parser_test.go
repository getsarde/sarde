package content

import (
	"testing"
)

func TestParse_YAML(t *testing.T) {
	raw := []byte("---\ntitle: \"Hello World\"\ntags:\n  - go\n  - ssg\n---\nThis is the body.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "Hello World" {
		t.Errorf("title = %v, want %q", fm["title"], "Hello World")
	}
	tags, ok := fm["tags"].([]interface{})
	if !ok || len(tags) != 2 {
		t.Errorf("tags = %v, want [go, ssg]", fm["tags"])
	}
	if body != "This is the body.\n" {
		t.Errorf("body = %q, want %q", body, "This is the body.\n")
	}
}

func TestParse_TOML(t *testing.T) {
	raw := []byte("+++\ntitle = \"Hello TOML\"\norder = 5\n+++\nTOML body.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "Hello TOML" {
		t.Errorf("title = %v, want %q", fm["title"], "Hello TOML")
	}
	// TOML numbers come as int64
	if w, ok := fm["order"].(int64); !ok || w != 5 {
		t.Errorf("weight = %v (%T), want 5", fm["order"], fm["order"])
	}
	if body != "TOML body.\n" {
		t.Errorf("body = %q, want %q", body, "TOML body.\n")
	}
}

func TestParse_JSON(t *testing.T) {
	raw := []byte("{\"title\": \"Hello JSON\", \"draft\": true}\nJSON body.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "Hello JSON" {
		t.Errorf("title = %v, want %q", fm["title"], "Hello JSON")
	}
	if fm["draft"] != true {
		t.Errorf("draft = %v, want true", fm["draft"])
	}
	if body != "JSON body.\n" {
		t.Errorf("body = %q, want %q", body, "JSON body.\n")
	}
}

// Braces inside JSON string values must not confuse the end-of-frontmatter
// detection (a naive brace counter truncates at the } inside the string).
func TestParse_JSON_BracesInStrings(t *testing.T) {
	raw := []byte("{\"title\": \"Hello\", \"description\": \"use {} in code blocks\"}\nJSON body.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "Hello" {
		t.Errorf("title = %v, want %q", fm["title"], "Hello")
	}
	if fm["description"] != "use {} in code blocks" {
		t.Errorf("description = %v, want %q", fm["description"], "use {} in code blocks")
	}
	if body != "JSON body.\n" {
		t.Errorf("body = %q, want %q", body, "JSON body.\n")
	}
}

func TestParse_JSON_NestedObjectsWithBraces(t *testing.T) {
	raw := []byte("{\"title\": \"T\", \"params\": {\"snippet\": \"if (x) { y() }\"}}\nBody.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	params, ok := fm["params"].(map[string]interface{})
	if !ok {
		t.Fatalf("params = %T, want map", fm["params"])
	}
	if params["snippet"] != "if (x) { y() }" {
		t.Errorf("snippet = %v, want %q", params["snippet"], "if (x) { y() }")
	}
	if body != "Body.\n" {
		t.Errorf("body = %q, want %q", body, "Body.\n")
	}
}

func TestParse_JSON_EscapedQuotesInStrings(t *testing.T) {
	raw := []byte("{\"title\": \"say \\\"hi\\\" {}\"}\nBody.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != `say "hi" {}` {
		t.Errorf("title = %v, want %q", fm["title"], `say "hi" {}`)
	}
	if body != "Body.\n" {
		t.Errorf("body = %q, want %q", body, "Body.\n")
	}
}

// An unterminated opening brace is not JSON frontmatter — the whole content
// is the body, mirroring the YAML/TOML missing-closing-delimiter behavior.
func TestParse_JSON_UnclosedBrace(t *testing.T) {
	raw := []byte("{\"title\": \"never closed\"\nBody text.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("fm should be empty, got %v", fm)
	}
	if body != string(raw) {
		t.Errorf("body = %q, want full content", body)
	}
}

func TestParse_JSON_MalformedReturnsError(t *testing.T) {
	raw := []byte("{\"title\": bad}\nBody.\n")
	p := &Parser{}
	_, _, err := p.Parse(raw)
	if err == nil {
		t.Fatal("expected error for malformed JSON frontmatter, got nil")
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	raw := []byte("Just some content\nwith no frontmatter.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("fm should be empty, got %v", fm)
	}
	if body != "Just some content\nwith no frontmatter.\n" {
		t.Errorf("body = %q, want full content", body)
	}
}

func TestParse_EmptyFile(t *testing.T) {
	p := &Parser{}
	fm, body, err := p.Parse([]byte(""))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("fm should be empty, got %v", fm)
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestParse_BOM(t *testing.T) {
	raw := []byte("\xef\xbb\xbf---\ntitle: \"BOM Test\"\n---\nBody.\n")
	p := &Parser{}
	fm, _, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "BOM Test" {
		t.Errorf("title = %v, want %q", fm["title"], "BOM Test")
	}
}

func TestParseFrontmatter_TypedYAML(t *testing.T) {
	raw := []byte("---\ntitle: \"Typed Test\"\ndraft: true\nsidebar:\n  order: 10\ntags:\n  - go\n---\nBody content.\n")
	fm, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Title != "Typed Test" {
		t.Errorf("Title = %q, want %q", fm.Title, "Typed Test")
	}
	if !fm.Draft {
		t.Error("Draft should be true")
	}
	if fm.Sidebar.Order != 10 {
		t.Errorf("Weight = %d, want 10", fm.Sidebar.Order)
	}
	if len(fm.Tags) != 1 || fm.Tags[0] != "go" {
		t.Errorf("Tags = %v, want [go]", fm.Tags)
	}
	if body != "Body content.\n" {
		t.Errorf("body = %q, want %q", body, "Body content.\n")
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	raw := []byte("Just plain markdown.\n")
	fm, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Title != "" {
		t.Errorf("Title = %q, want empty", fm.Title)
	}
	if body != "Just plain markdown.\n" {
		t.Errorf("body = %q, want full content", body)
	}
}

func TestParseFrontmatter_Image(t *testing.T) {
	raw := []byte("---\ntitle: Test\nimage: /img/hero.png\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Image != "/img/hero.png" {
		t.Errorf("Image = %q, want %q", fm.Image, "/img/hero.png")
	}
}

func TestParseFrontmatter_RenderFalse(t *testing.T) {
	raw := []byte("---\ntitle: Test\nrender: false\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Render == nil {
		t.Fatal("Render should not be nil")
	}
	if *fm.Render != false {
		t.Errorf("Render = %v, want false", *fm.Render)
	}
}

func TestParseFrontmatter_Transparent(t *testing.T) {
	raw := []byte("---\ntitle: Section\ntransparent: true\n---\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if !fm.Transparent {
		t.Error("Transparent should be true")
	}
}

func TestParseFrontmatter_Summary(t *testing.T) {
	raw := []byte("---\ntitle: Test\nsummary: \"A hand-written summary.\"\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Summary != "A hand-written summary." {
		t.Errorf("Summary = %q, want %q", fm.Summary, "A hand-written summary.")
	}
}

func TestParseFrontmatter_PrevNext(t *testing.T) {
	raw := []byte("---\ntitle: Test\nprev: intro\nnext: advanced\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Prev == nil || fm.Prev.Slug != "intro" {
		t.Errorf("Prev.Slug = %v, want %q", fm.Prev, "intro")
	}
	if fm.Next == nil || fm.Next.Slug != "advanced" {
		t.Errorf("Next.Slug = %v, want %q", fm.Next, "advanced")
	}
}

func TestParseFrontmatter_SidebarAttrs(t *testing.T) {
	raw := []byte("---\ntitle: Test\nsidebar:\n  attrs:\n    icon: star\n    badge: new\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if len(fm.Sidebar.Attrs) != 2 {
		t.Fatalf("Sidebar.Attrs len = %d, want 2", len(fm.Sidebar.Attrs))
	}
	if fm.Sidebar.Attrs["icon"] != "star" {
		t.Errorf("Sidebar.Attrs[icon] = %q, want %q", fm.Sidebar.Attrs["icon"], "star")
	}
	if fm.Sidebar.Attrs["badge"] != "new" {
		t.Errorf("Sidebar.Attrs[badge] = %q, want %q", fm.Sidebar.Attrs["badge"], "new")
	}
}

func TestParseFrontmatter_TOC(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntoc:\n  enabled: false\n  min_level: 2\n  max_level: 4\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.TOC.Enabled == nil {
		t.Fatal("TOC.Enabled should not be nil")
	}
	if *fm.TOC.Enabled != false {
		t.Errorf("TOC.Enabled = %v, want false", *fm.TOC.Enabled)
	}
	if fm.TOC.MinLevel != 2 {
		t.Errorf("TOC.MinLevel = %d, want 2", fm.TOC.MinLevel)
	}
	if fm.TOC.MaxLevel != 4 {
		t.Errorf("TOC.MaxLevel = %d, want 4", fm.TOC.MaxLevel)
	}
}

func TestParseFrontmatter_Pagefind(t *testing.T) {
	raw := []byte("---\ntitle: Test\npagefind: false\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Pagefind == nil {
		t.Fatal("Pagefind should not be nil")
	}
	if *fm.Pagefind != false {
		t.Errorf("Pagefind = %v, want false", *fm.Pagefind)
	}
}

func TestParseFrontmatter_Layout(t *testing.T) {
	raw := []byte("---\ntitle: Test\nlayout: splash\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Layout != "splash" {
		t.Errorf("Layout = %q, want %q", fm.Layout, "splash")
	}
}

func TestParseFrontmatter_Type(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntype: tutorial\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Type != "tutorial" {
		t.Errorf("Type = %q, want %q", fm.Type, "tutorial")
	}
}

func TestParseFrontmatter_Hero(t *testing.T) {
	raw := []byte("---\ntitle: Home\nhero:\n  title: Welcome\n  tagline: Build fast sites\n  image:\n    src: /img/hero.png\n    alt: Hero image\n  actions:\n    - text: Get Started\n      link: /docs/\n      variant: primary\n      icon: \"lucide:arrow-right\"\n    - text: Learn More\n      link: /about/\n      variant: secondary\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Hero == nil {
		t.Fatal("Hero should not be nil")
	}
	if fm.Hero.Title != "Welcome" {
		t.Errorf("Hero.Title = %q, want %q", fm.Hero.Title, "Welcome")
	}
	if fm.Hero.Tagline != "Build fast sites" {
		t.Errorf("Hero.Tagline = %q, want %q", fm.Hero.Tagline, "Build fast sites")
	}
	if fm.Hero.Image == nil {
		t.Fatal("Hero.Image should not be nil")
	}
	if fm.Hero.Image.Src != "/img/hero.png" {
		t.Errorf("Hero.Image.Src = %q, want %q", fm.Hero.Image.Src, "/img/hero.png")
	}
	if fm.Hero.Image.Alt != "Hero image" {
		t.Errorf("Hero.Image.Alt = %q, want %q", fm.Hero.Image.Alt, "Hero image")
	}
	if len(fm.Hero.Actions) != 2 {
		t.Fatalf("Hero.Actions len = %d, want 2", len(fm.Hero.Actions))
	}
	a := fm.Hero.Actions[0]
	if a.Text != "Get Started" {
		t.Errorf("Actions[0].Text = %q, want %q", a.Text, "Get Started")
	}
	if a.Link != "/docs/" {
		t.Errorf("Actions[0].Link = %q, want %q", a.Link, "/docs/")
	}
	if a.Variant != "primary" {
		t.Errorf("Actions[0].Variant = %q, want %q", a.Variant, "primary")
	}
	if a.Icon != "lucide:arrow-right" {
		t.Errorf("Actions[0].Icon = %q, want %q", a.Icon, "lucide:arrow-right")
	}
}

func TestParseFrontmatter_EmptyDateFields(t *testing.T) {
	// Sarde Studio writes an empty string when a date field is cleared. Such a
	// page must parse as "date unset", not abort the whole build.
	raw := []byte("---\ntitle: Cleared Dates\npublish_date: ''\nexpiry_date: ''\ndate: ''\n---\n\nBody.\n")
	fm, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.Title != "Cleared Dates" {
		t.Errorf("Title = %q, want %q", fm.Title, "Cleared Dates")
	}
	if !fm.Date.IsZero() || !fm.PublishDate.IsZero() || !fm.ExpiryDate.IsZero() {
		t.Errorf("empty date strings should decode as zero, got date=%v publish=%v expiry=%v",
			fm.Date.Time, fm.PublishDate.Time, fm.ExpiryDate.Time)
	}
	if body != "\nBody.\n" {
		t.Errorf("body = %q, want %q", body, "\nBody.\n")
	}
}

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
	raw := []byte("+++\ntitle = \"Hello TOML\"\nweight = 5\n+++\nTOML body.\n")
	p := &Parser{}
	fm, body, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if fm["title"] != "Hello TOML" {
		t.Errorf("title = %v, want %q", fm["title"], "Hello TOML")
	}
	// TOML numbers come as int64
	if w, ok := fm["weight"].(int64); !ok || w != 5 {
		t.Errorf("weight = %v (%T), want 5", fm["weight"], fm["weight"])
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
	raw := []byte("---\ntitle: \"Typed Test\"\ndraft: true\nweight: 10\ntags:\n  - go\n---\nBody content.\n")
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
	if fm.Weight != 10 {
		t.Errorf("Weight = %d, want 10", fm.Weight)
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
	raw := []byte("---\ntitle: Test\nsidebar_attrs:\n  icon: star\n  badge: new\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if len(fm.SidebarAttrs) != 2 {
		t.Fatalf("SidebarAttrs len = %d, want 2", len(fm.SidebarAttrs))
	}
	if fm.SidebarAttrs["icon"] != "star" {
		t.Errorf("SidebarAttrs[icon] = %q, want %q", fm.SidebarAttrs["icon"], "star")
	}
	if fm.SidebarAttrs["badge"] != "new" {
		t.Errorf("SidebarAttrs[badge] = %q, want %q", fm.SidebarAttrs["badge"], "new")
	}
}

func TestParseFrontmatter_SidebarGroup(t *testing.T) {
	raw := []byte("---\ntitle: Test\nsidebar_group: \"API Reference\"\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.SidebarGroup != "API Reference" {
		t.Errorf("SidebarGroup = %q, want %q", fm.SidebarGroup, "API Reference")
	}
}

func TestParseFrontmatter_TOC(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntoc: false\ntoc_min_level: 2\ntoc_max_level: 4\n---\nBody.\n")
	fm, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter error: %v", err)
	}
	if fm.TOC == nil {
		t.Fatal("TOC should not be nil")
	}
	if *fm.TOC != false {
		t.Errorf("TOC = %v, want false", *fm.TOC)
	}
	if fm.TOCMinLevel != 2 {
		t.Errorf("TOCMinLevel = %d, want 2", fm.TOCMinLevel)
	}
	if fm.TOCMaxLevel != 4 {
		t.Errorf("TOCMaxLevel = %d, want 4", fm.TOCMaxLevel)
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

package content

import (
	"path/filepath"
	"testing"
)

func TestComputePermalink(t *testing.T) {
	content := filepath.Join("project", "content")

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			"home",
			filepath.Join(content, "_index.md"),
			"/",
		},
		{
			"standalone",
			filepath.Join(content, "about.md"),
			"/about/",
		},
		{
			"section index",
			filepath.Join(content, "docs", "_index.md"),
			"/docs/",
		},
		{
			"page in collection",
			filepath.Join(content, "docs", "getting-started.md"),
			"/docs/getting-started/",
		},
		{
			"nested section index",
			filepath.Join(content, "docs", "guides", "_index.md"),
			"/docs/guides/",
		},
		{
			"nested page",
			filepath.Join(content, "docs", "guides", "authentication.md"),
			"/docs/guides/authentication/",
		},
		{
			"bundle index.md",
			filepath.Join(content, "docs", "my-bundle", "index.md"),
			"/docs/my-bundle/",
		},
		{
			"numeric prefix page",
			filepath.Join(content, "courses", "01-basics.md"),
			"/courses/basics/",
		},
		{
			"nested numeric prefix",
			filepath.Join(content, "courses", "go-fundamentals", "01-variables.md"),
			"/courses/go-fundamentals/variables/",
		},
		{
			"blog post",
			filepath.Join(content, "blog", "hello-world.md"),
			"/blog/hello-world/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePermalink(content, tt.filePath)
			if got != tt.want {
				t.Errorf("ComputePermalink(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestComputePatternPermalink(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		vars    PermalinkVars
		want    string
	}{
		{
			"blog pattern",
			"/:collection/:year/:month/:slug/",
			PermalinkVars{Slug: "hello-world", Year: "2026", Month: "04", Day: "06", Collection: "blog"},
			"/blog/2026/04/hello-world/",
		},
		{
			"docs with section",
			"/:collection/:section/:slug/",
			PermalinkVars{Slug: "install", Section: "guides", Collection: "docs"},
			"/docs/guides/install/",
		},
		{
			"missing vars collapse slashes",
			"/:collection/:section/:slug/",
			PermalinkVars{Slug: "about", Section: "", Collection: "pages"},
			"/pages/about/",
		},
		{
			"no leading slash in pattern",
			":collection/:slug/",
			PermalinkVars{Slug: "intro", Collection: "docs"},
			"/docs/intro/",
		},
		{
			"no trailing slash in pattern",
			"/:collection/:slug",
			PermalinkVars{Slug: "intro", Collection: "docs"},
			"/docs/intro/",
		},
		{
			"title variable",
			"/:collection/:title/",
			PermalinkVars{Title: "my-page", Collection: "blog"},
			"/blog/my-page/",
		},
		{
			"full date pattern",
			"/:collection/:year/:month/:day/:slug/",
			PermalinkVars{Slug: "post", Year: "2026", Month: "01", Day: "15", Collection: "blog"},
			"/blog/2026/01/15/post/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePatternPermalink(tt.pattern, tt.vars)
			if got != tt.want {
				t.Errorf("ComputePatternPermalink(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestPrefixPermalink(t *testing.T) {
	tests := []struct {
		permalink   string
		lang        string
		defaultLang string
		want        string
	}{
		{"/docs/getting-started/", "en", "en", "/docs/getting-started/"},
		{"/docs/getting-started/", "fr", "en", "/fr/docs/getting-started/"},
		{"/blog/welcome/", "ar", "en", "/ar/blog/welcome/"},
		{"/", "fr", "en", "/fr/"},
		{"/about/", "", "en", "/about/"},
	}

	for _, tt := range tests {
		got := PrefixPermalink(tt.permalink, tt.lang, tt.defaultLang)
		if got != tt.want {
			t.Errorf("PrefixPermalink(%q, %q, %q) = %q, want %q",
				tt.permalink, tt.lang, tt.defaultLang, got, tt.want)
		}
	}
}

func TestComputePermalinkFromRelPath(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		want    string
	}{
		{"regular page", "docs/guide.md", "/docs/guide/"},
		{"nested page", "docs/guides/auth.md", "/docs/guides/auth/"},
		{"section index", "docs/_index.md", "/docs/"},
		{"page bundle", "docs/guide/index.md", "/docs/guide/"},
		{"root index", "_index.md", "/"},
		{"root page", "about.md", "/about/"},
		{"numeric prefix stripped", "docs/01-getting-started.md", "/docs/getting-started/"},
		{"deeply nested", "docs/reference/api/rest.md", "/docs/reference/api/rest/"},
		{"nested section index", "docs/guides/_index.md", "/docs/guides/"},
		{"root page bundle", "about/index.md", "/about/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ComputePermalinkFromRelPath(c.relPath); got != c.want {
				t.Errorf("ComputePermalinkFromRelPath(%q) = %q, want %q", c.relPath, got, c.want)
			}
		})
	}
}

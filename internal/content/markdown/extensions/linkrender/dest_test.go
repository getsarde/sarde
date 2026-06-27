package linkrender

import (
	"testing"
)

func TestClassifyDest(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		wantKind LinkKind
		wantPath string
		wantFrag string
		wantQ    string
	}{
		// External links — pass through untouched.
		{"https", "https://example.com/page", LinkExternal, "", "", ""},
		{"http", "http://example.com", LinkExternal, "", "", ""},
		{"protocol-relative", "//cdn.example.com/lib.js", LinkExternal, "", "", ""},
		{"mailto", "mailto:user@example.com", LinkExternal, "", "", ""},
		{"tel", "tel:+1234567890", LinkExternal, "", "", ""},
		{"data uri", "data:text/html,<h1>Hi</h1>", LinkExternal, "", "", ""},
		{"javascript", "javascript:void(0)", LinkExternal, "", "", ""},
		{"empty", "", LinkExternal, "", "", ""},

		// Anchor-only.
		{"anchor only", "#setup", LinkAnchorOnly, "", "setup", ""},
		{"anchor with query", "#setup?foo=bar", LinkAnchorOnly, "", "setup?foo=bar", ""},

		// Relative links (recommended form) — always treated as source-file refs.
		{"relative same dir", "./auth.md", LinkRelative, "auth", "", ""},
		{"relative same dir no ext", "./auth", LinkRelative, "auth", "", ""},
		{"relative parent", "../guides/auth.md", LinkRelative, "../guides/auth", "", ""},
		{"relative nested", "./sub/deep/page.md", LinkRelative, "sub/deep/page", "", ""},
		{"relative with fragment", "./auth.md#setup", LinkRelative, "auth", "setup", ""},
		{"relative with query", "./auth.md?version=v1", LinkRelative, "auth", "", "version=v1"},
		{"relative with both", "./auth.md?v=1#top", LinkRelative, "auth", "top", "v=1"},
		{"relative mdx", "./component.mdx", LinkRelative, "component", "", ""},
		{"relative index", "./_index.md", LinkRelative, "_index", "", ""},
		{"relative dir trailing slash", "./guides/", LinkRelative, "guides", "", ""},

		// Content-root-relative (requires .md/.mdx extension to be treated as source-file ref).
		{"content root with ext", "/guides/auth.md", LinkContentRoot, "/guides/auth", "", ""},
		{"content root no ext passthrough", "/guides/auth", LinkExternal, "", "", ""},
		{"content root with fragment", "/guides/auth.md#api", LinkContentRoot, "/guides/auth", "api", ""},
		{"content root deep", "/a/b/c/page.md", LinkContentRoot, "/a/b/c/page", "", ""},
		{"content root url passthrough", "/about/", LinkExternal, "", "", ""},

		// Bare name with .md — ambiguous (Hugo #4727 lesson).
		{"bare name with ext", "auth.md", LinkAmbiguous, "auth", "", ""},
		{"bare with fragment", "auth.md#setup", LinkAmbiguous, "auth", "setup", ""},
		{"bare path with ext", "guides/auth.md", LinkAmbiguous, "guides/auth", "", ""},
		// Bare name without .md — pass through as already-resolved URL.
		{"bare name no ext passthrough", "auth", LinkExternal, "", "", ""},
		{"bare path no ext passthrough", "guides/auth", LinkExternal, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyDest(tt.href)

			if got.Kind != tt.wantKind {
				t.Errorf("Kind = %d, want %d", got.Kind, tt.wantKind)
			}
			if got.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.Fragment != tt.wantFrag {
				t.Errorf("Fragment = %q, want %q", got.Fragment, tt.wantFrag)
			}
			if got.Query != tt.wantQ {
				t.Errorf("Query = %q, want %q", got.Query, tt.wantQ)
			}
			if got.Raw != tt.href {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.href)
			}
		})
	}
}

func TestStripMarkdownExt(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"auth.md", "auth"},
		{"auth.mdx", "auth"},
		{"auth.txt", "auth.txt"},
		{"path/to/file.md", "path/to/file"},
		{"no-extension", "no-extension"},
		{".md", ""},
		{"readme.markdown", "readme.markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stripMarkdownExt(tt.input); got != tt.want {
				t.Errorf("stripMarkdownExt(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsExternal(t *testing.T) {
	externals := []string{
		"https://example.com",
		"http://example.com",
		"//cdn.example.com",
		"mailto:x@y.com",
		"tel:+123",
		"data:text/html,hi",
		"javascript:void(0)",
	}
	for _, href := range externals {
		if !isExternal(href) {
			t.Errorf("isExternal(%q) = false, want true", href)
		}
	}

	internals := []string{
		"./auth.md",
		"../page.md",
		"/guides/auth.md",
		"auth.md",
		"#anchor",
		"",
	}
	for _, href := range internals {
		if isExternal(href) {
			t.Errorf("isExternal(%q) = true, want false", href)
		}
	}
}

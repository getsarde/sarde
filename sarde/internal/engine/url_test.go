package engine

import "testing"

func newTestResolver(baseURL, basePath string) *URLResolver {
	return &URLResolver{BaseURL: baseURL, BasePath: basePath}
}

func TestURL_BasePath(t *testing.T) {
	cases := []struct {
		name, basePath, relPath, want string
	}{
		{"subdir", "/docs/", "/guides/auth/", "/docs/guides/auth/"},
		{"root", "/", "/guides/auth/", "/guides/auth/"},
		{"subdir no trailing rel", "/docs/", "/guides/auth", "/docs/guides/auth"},
		{"rel is root", "/docs/", "/", "/docs/"},
		{"double slashes collapse", "/docs/", "//guides//auth//", "/docs/guides/auth/"},
		{"nested basepath", "/a/b/", "/x/", "/a/b/x/"},
		{"empty rel", "/docs/", "", "/docs/"},
	}
	for _, c := range cases {
		r := newTestResolver("https://x.com", c.basePath)
		if got := r.URL(c.relPath, "", ""); got != c.want {
			t.Errorf("%s: URL(%q) = %q; want %q", c.name, c.relPath, got, c.want)
		}
	}
}

// The resolver always prefixes — no idempotency guard. Callers must
// pass prefix-free input and call the resolver exactly once per URL.
func TestURL_AlwaysPrefixes(t *testing.T) {
	cases := []struct {
		name, basePath, relPath, want string
	}{
		{"collection matches basepath", "/docs/", "/docs/", "/docs/docs/"},
		{"collection matches basepath nested", "/docs/", "/docs/guides/auth/", "/docs/docs/guides/auth/"},
		{"no collision", "/docs/", "/blog/", "/docs/blog/"},
		{"boundary safe", "/docs/", "/docsfoo/x/", "/docs/docsfoo/x/"},
	}
	for _, c := range cases {
		r := newTestResolver("https://x.com", c.basePath)
		if got := r.URL(c.relPath, "", ""); got != c.want {
			t.Errorf("%s: URL(%q) = %q; want %q", c.name, c.relPath, got, c.want)
		}
	}
}

func TestAbsURL_BasePath(t *testing.T) {
	r := newTestResolver("https://frostybee.dev/", "/docs/")
	got := r.AbsURL("/guides/auth/", "", "")
	want := "https://frostybee.dev/docs/guides/auth/"
	if got != want {
		t.Errorf("AbsURL = %q; want %q", got, want)
	}
}

func TestURL_IgnoresLangVersionInPhaseA(t *testing.T) {
	r := newTestResolver("https://x.com", "/docs/")
	base := r.URL("/guides/", "", "")
	if got := r.URL("/guides/", "fr", "v1"); got != base {
		t.Errorf("lang/version changed output in Phase A: got %q, base %q", got, base)
	}
}

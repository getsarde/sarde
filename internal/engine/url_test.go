package engine

import "testing"

func newTestResolver(baseURL, basePath string) *URLResolver {
	return &URLResolver{BaseURL: baseURL, BasePath: basePath}
}

func newI18nResolver(baseURL, basePath, defaultLang string, langs []string) *URLResolver {
	langSet := make(map[string]bool, len(langs))
	for _, l := range langs {
		langSet[l] = true
	}
	return &URLResolver{
		BaseURL:     baseURL,
		BasePath:    basePath,
		I18nEnabled: true,
		DefaultLang: defaultLang,
		Strategy:    "prefix-except-default",
		Languages:   langSet,
	}
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

func TestURL_BasePathIdempotent(t *testing.T) {
	r := newTestResolver("https://x.com", "/docs/")
	cases := []struct {
		name, relPath, want string
	}{
		{"prefix-free input", "/guides/auth/", "/docs/guides/auth/"},
		{"already prefixed (idempotent)", "/docs/guides/auth/", "/docs/guides/auth/"},
		{"already prefixed exact", "/docs/", "/docs/"},
		{"no collision", "/blog/", "/docs/blog/"},
		{"boundary safe", "/docsfoo/x/", "/docs/docsfoo/x/"},
	}
	for _, c := range cases {
		if got := r.URL(c.relPath, "", ""); got != c.want {
			t.Errorf("%s: URL(%q) = %q; want %q", c.name, c.relPath, got, c.want)
		}
		// Double application must equal single application
		once := r.URL(c.relPath, "", "")
		twice := r.URL(once, "", "")
		if once != twice {
			t.Errorf("%s: not idempotent: URL(URL(x))=%q != URL(x)=%q", c.name, twice, once)
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

// --- i18n lang segment tests ---

func TestURL_LangSegment(t *testing.T) {
	r := newI18nResolver("https://x.com", "/docs/", "en", []string{"en", "fr", "ar"})
	cases := []struct {
		name, lang, rel, want string
	}{
		{"default unprefixed", "en", "/guides/auth/", "/docs/guides/auth/"},
		{"default via empty", "", "/guides/auth/", "/docs/guides/auth/"},
		{"french prefixed", "fr", "/guides/auth/", "/docs/fr/guides/auth/"},
		{"arabic prefixed", "ar", "/guides/auth/", "/docs/ar/guides/auth/"},
		{"root path french", "fr", "/", "/docs/fr/"},
		{"root basepath french", "fr", "/blog/post/", "/docs/fr/blog/post/"},
	}
	for _, c := range cases {
		if got := r.URL(c.rel, c.lang, ""); got != c.want {
			t.Errorf("%s: URL(%q, %q) = %q; want %q", c.name, c.rel, c.lang, got, c.want)
		}
	}
}

func TestURL_LangWithRootBasePath(t *testing.T) {
	r := newI18nResolver("https://x.com", "/", "en", []string{"en", "fr"})
	cases := []struct {
		name, lang, rel, want string
	}{
		{"default at root", "en", "/guides/", "/guides/"},
		{"french at root", "fr", "/guides/", "/fr/guides/"},
		{"french root", "fr", "/", "/fr/"},
	}
	for _, c := range cases {
		if got := r.URL(c.rel, c.lang, ""); got != c.want {
			t.Errorf("%s: URL(%q, %q) = %q; want %q", c.name, c.rel, c.lang, got, c.want)
		}
	}
}

func TestURL_LangIdempotent(t *testing.T) {
	r := newI18nResolver("https://x.com", "/docs/", "en", []string{"en", "fr"})
	once := r.URL("/guides/", "fr", "")
	twice := r.URL(once, "fr", "")
	if once != twice {
		t.Errorf("not idempotent with lang: once=%q twice=%q", once, twice)
	}
	if once != "/docs/fr/guides/" {
		t.Errorf("got %q want /docs/fr/guides/", once)
	}
}

func TestURL_NoI18n_LangIgnored(t *testing.T) {
	r := newTestResolver("https://x.com", "/docs/")
	base := r.URL("/guides/", "", "")
	withLang := r.URL("/guides/", "fr", "")
	if base != withLang {
		t.Errorf("lang should be ignored when i18n disabled: base=%q withLang=%q", base, withLang)
	}
}

func TestAbsURL_WithLang(t *testing.T) {
	r := newI18nResolver("https://frostybee.dev/", "/docs/", "en", []string{"en", "fr"})
	got := r.AbsURL("/guides/auth/", "fr", "")
	want := "https://frostybee.dev/docs/fr/guides/auth/"
	if got != want {
		t.Errorf("AbsURL = %q; want %q", got, want)
	}
}

func TestOutputRelPath(t *testing.T) {
	r := newI18nResolver("https://x.com", "/docs/", "en", []string{"en", "fr"})
	cases := []struct {
		name, lang, rel, want string
	}{
		{"default no prefix", "en", "/guides/", "/guides/"},
		{"french prefixed", "fr", "/guides/", "/fr/guides/"},
		{"empty lang default", "", "/guides/", "/guides/"},
		{"root french", "fr", "/", "/fr/"},
	}
	for _, c := range cases {
		if got := r.OutputRelPath(c.rel, c.lang, ""); got != c.want {
			t.Errorf("%s: OutputRelPath(%q, %q) = %q; want %q", c.name, c.rel, c.lang, got, c.want)
		}
	}

	// OutputRelPath must NOT include basePath
	noI18n := newTestResolver("https://x.com", "/docs/")
	if got := noI18n.OutputRelPath("/guides/", "", ""); got != "/guides/" {
		t.Errorf("OutputRelPath without i18n: got %q, want /guides/", got)
	}
}

func TestFirstSegment(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"/fr/guides/", "fr"},
		{"/guides/", "guides"},
		{"/", ""},
		{"", ""},
		{"fr/guides", "fr"},
	}
	for _, c := range cases {
		if got := firstSegment(c.input); got != c.want {
			t.Errorf("firstSegment(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

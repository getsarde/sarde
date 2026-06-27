package engine

import "testing"

func versionedResolver(basePath, defaultLang string, versionIDs []string, mounts []string) *URLResolver {
	vids := make(map[string]bool, len(versionIDs))
	for _, id := range versionIDs {
		vids[id] = true
	}
	langs := map[string]bool{"en": true, "fr": true}
	return &URLResolver{
		BasePath:         basePath,
		BaseURL:          "https://example.com",
		I18nEnabled:      true,
		DefaultLang:      defaultLang,
		Strategy:         "prefix-except-default",
		Languages:        langs,
		CollectionMounts: mounts,
		VersionIDs:       vids,
	}
}

func TestURL_VersionInsideCollection(t *testing.T) {
	r := versionedResolver("/site/", "en", []string{"v1", "v2"}, []string{"/docs", "/blog"})

	cases := []struct {
		name, lang, ver, rel, want string
	}{
		{"latest en (alias)", "en", "", "/docs/guides/auth/", "/site/docs/guides/auth/"},
		{"latest fr", "fr", "", "/docs/guides/auth/", "/site/fr/docs/guides/auth/"},
		{"v1 en", "en", "v1", "/docs/guides/auth/", "/site/docs/v1/guides/auth/"},
		{"v1 fr", "fr", "v1", "/docs/guides/auth/", "/site/fr/docs/v1/guides/auth/"},
		{"lang outer of coll", "fr", "v1", "/docs/x/", "/site/fr/docs/v1/x/"},
		{"unversioned blog", "fr", "", "/blog/my-post/", "/site/fr/blog/my-post/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.URL(c.rel, c.lang, c.ver)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestURL_VersionIdempotent(t *testing.T) {
	r := versionedResolver("/site/", "en", []string{"v1", "v2"}, []string{"/docs"})

	once := r.URL("/docs/guides/", "fr", "v1")
	twice := r.URL(once, "fr", "v1")
	want := "/site/fr/docs/v1/guides/"
	if once != want {
		t.Errorf("first pass: got %q, want %q", once, want)
	}
	if once != twice {
		t.Errorf("not idempotent: once=%q twice=%q", once, twice)
	}
}

func TestOutputRelPath_Version(t *testing.T) {
	r := versionedResolver("/site/", "en", []string{"v1", "v2"}, []string{"/docs"})

	cases := []struct {
		name, lang, ver, rel, want string
	}{
		{"latest en", "en", "", "/docs/guides/auth/", "/docs/guides/auth/"},
		{"latest fr", "fr", "", "/docs/guides/auth/", "/fr/docs/guides/auth/"},
		{"v1 en", "en", "v1", "/docs/guides/auth/", "/docs/v1/guides/auth/"},
		{"v1 fr", "fr", "v1", "/docs/guides/auth/", "/fr/docs/v1/guides/auth/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.OutputRelPath(c.rel, c.lang, c.ver)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestCollectionMountFor(t *testing.T) {
	r := &URLResolver{
		CollectionMounts: []string{"/docs", "/blog", "/docs/api"},
	}
	cases := []struct {
		rel, want string
	}{
		{"/docs/guides/auth/", "/docs"},
		{"/docs/api/reference/", "/docs/api"},
		{"/blog/my-post/", "/blog"},
		{"/standalone/", ""},
	}
	for _, c := range cases {
		got := r.collectionMountFor(c.rel)
		if got != c.want {
			t.Errorf("collectionMountFor(%q) = %q, want %q", c.rel, got, c.want)
		}
	}
}

func TestIsVersionID(t *testing.T) {
	r := &URLResolver{VersionIDs: map[string]bool{"v1": true, "v2": true}}
	if !r.IsVersionID("v1") {
		t.Error("expected v1 to be a version ID")
	}
	if r.IsVersionID("guides") {
		t.Error("expected guides to not be a version ID")
	}
}

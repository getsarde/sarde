package i18n

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestBlogKeys(t *testing.T) {
	yamlContent := `nav:
  previous: "Previous"
  newer: "Newer"
  older: "Older"
  newer_posts: "Newer posts"
  older_posts: "Older posts"
blog:
  featured: "Featured"
  page: "Page"
  of: "of"
`
	fsys := fstest.MapFS{
		"en.yaml": &fstest.MapFile{Data: []byte(yamlContent)},
	}
	st, err := LoadStrings(fs.FS(fsys), t.TempDir(), "", "en")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"nav.previous":    "Previous",
		"nav.newer":       "Newer",
		"nav.older_posts": "Older posts",
		"blog.featured":   "Featured",
		"blog.page":       "Page",
		"blog.of":         "of",
	}
	for key, want := range tests {
		got := st.Resolve("en", key)
		if got != want {
			t.Errorf("Resolve(%q) = %q, want %q", key, got, want)
		}
	}
}

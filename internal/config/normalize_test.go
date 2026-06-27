package config

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "/"},
		{"/", "/"},
		{"docs", "/docs/"},
		{"/docs", "/docs/"},
		{"docs/", "/docs/"},
		{"/docs/", "/docs/"},
		{"  /docs/  ", "/docs/"},
		{"//docs//", "/docs/"},
		{"a/b", "/a/b/"},
		{"/a/b/", "/a/b/"},
		{"a//b", "/a/b/"},
		{"///", "/"},
	}
	for _, c := range cases {
		if got := NormalizeBasePath(c.in); got != c.want {
			t.Errorf("NormalizeBasePath(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

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

func TestNormalizeDateFormat(t *testing.T) {
	cases := []struct{ in, want string }{
		// Empty must match what the theme hardcoded before the setting existed,
		// so adding the key changes no existing site's output.
		{"", "Jan 2, 2006"},
		{"short", "Jan 2, 2006"},
		{"long", "January 2, 2006"},
		{"iso", "2006-01-02"},
		{"ISO", "2006-01-02"},
		{"  short  ", "Jan 2, 2006"},
		// Anything unrecognized is a raw Go layout and passes through.
		{"2006/01/02", "2006/01/02"},
		{"Mon Jan 2", "Mon Jan 2"},
	}
	for _, c := range cases {
		if got := NormalizeDateFormat(c.in); got != c.want {
			t.Errorf("NormalizeDateFormat(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

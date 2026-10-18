package suggest

import "testing"

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"bg-sidebar", "bg-sidbar", 1},
		{"font-sans", "fontsans", 1},
	}
	for _, tt := range tests {
		if got := Distance(tt.a, tt.b); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestClosest(t *testing.T) {
	plugins := []string{"sitemap", "search", "seo", "rss", "atom", "robots"}
	tests := []struct {
		name       string
		unknown    string
		candidates []string
		want       string
	}{
		{"one typo", "sitemp", plugins, "sitemap"},
		{"case differs", "SiteMap", plugins, "sitemap"},
		{"too far", "analytics", plugins, ""},
		{"exact match needs no hint", "seo", plugins, ""},
		{"empty", "", plugins, ""},
		// "rsx" is one edit from both; the first in sorted order wins,
		// whatever order the candidates arrive in.
		{"tie, sorted first", "rsx", []string{"rsz", "rss"}, "rss"},
		{"tie, other order", "rsx", []string{"rss", "rsz"}, "rss"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Closest(tt.unknown, tt.candidates); got != tt.want {
				t.Errorf("Closest(%q) = %q, want %q", tt.unknown, got, tt.want)
			}
		})
	}
}

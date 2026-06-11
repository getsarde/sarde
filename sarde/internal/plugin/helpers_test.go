package plugin

import "testing"

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		urlPath  string
		patterns []string
		want     bool
	}{
		// Single-component glob matches with and without trailing slash.
		{"/blog/post/", []string{"/blog/*"}, true},
		{"/blog/post", []string{"/blog/*"}, true},
		// path.Match: * never crosses a slash (on every platform).
		{"/blog/a/b", []string{"/blog/*"}, false},
		// Exact match.
		{"/drafts/wip/", []string{"/drafts/wip"}, true},
		// No patterns.
		{"/blog/post/", nil, false},
		// Non-matching pattern.
		{"/docs/guide/", []string{"/blog/*"}, false},
	}
	for _, tt := range tests {
		if got := shouldExclude(tt.urlPath, tt.patterns); got != tt.want {
			t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.urlPath, tt.patterns, got, tt.want)
		}
	}
}

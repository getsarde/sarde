package build

import "testing"

func TestRequiresAttribution(t *testing.T) {
	tests := []struct {
		spdx string
		want bool
	}{
		{"MIT", false},
		{"ISC", false},
		{"Apache-2.0", false},
		{"CC-BY-4.0", true},
		{"CC-BY-SA-4.0", true},
		{"cc-by-4.0", true}, // case-insensitive
		{"OFL-1.1", true},
		{"GPL-2.0", true},
		{"GPL-3.0-only", true},
		{"", false},
	}
	for _, tt := range tests {
		if got := requiresAttribution(tt.spdx); got != tt.want {
			t.Errorf("requiresAttribution(%q) = %v, want %v", tt.spdx, got, tt.want)
		}
	}
}

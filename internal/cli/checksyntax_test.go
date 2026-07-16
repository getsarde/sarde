package cli

import (
	"testing"
)

func TestStdinModeEligible(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		contentFlag string
		want        bool
	}{
		{"no args, no content flag", nil, "", true},
		{"project dir arg", []string{"/some/project"}, "", false},
		{"content flag only", nil, "/some/project/content", false},
		{"both arg and content flag", []string{"/some/project"}, "/some/project/content", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stdinModeEligible(tt.args, tt.contentFlag); got != tt.want {
				t.Errorf("stdinModeEligible(%v, %q) = %v, want %v", tt.args, tt.contentFlag, got, tt.want)
			}
		})
	}
}

package plugin

import (
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

func TestMatchesInjectRule_HasUpdated(t *testing.T) {
	stamp := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		page    *engine.Page
		want    bool
		comment string
	}{
		{
			name: "resolved timestamp, no override",
			page: &engine.Page{PageIdentity: engine.PageIdentity{Updated: stamp}},
			want: true,
		},
		{
			name: "no timestamp",
			page: &engine.Page{},
			want: false,
		},
		{
			name: "explicit updated date suppressed by show_updated",
			page: &engine.Page{
				PageIdentity: engine.PageIdentity{Updated: stamp},
				Params:       map[string]any{"show_updated": false},
			},
			want:    false,
			comment: "the badge must stay hidden even though the date is set explicitly",
		},
		{
			name: "show_updated true is the same as unset",
			page: &engine.Page{
				PageIdentity: engine.PageIdentity{Updated: stamp},
				Params:       map[string]any{"show_updated": true},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchesInjectRule("has_updated", tc.page, &engine.RouteData{})
			if got != tc.want {
				t.Errorf("has_updated = %v, want %v. %s", got, tc.want, tc.comment)
			}
		})
	}
}

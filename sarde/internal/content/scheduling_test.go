package content

import (
	"testing"
	"time"
)

var now = time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

func TestIsScheduled(t *testing.T) {
	tests := []struct {
		name        string
		publishDate time.Time
		want        bool
	}{
		{"zero date", time.Time{}, false},
		{"past date", now.Add(-24 * time.Hour), false},
		{"future date", now.Add(24 * time.Hour), true},
		{"exact now", now, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsScheduled(tt.publishDate, now); got != tt.want {
				t.Errorf("IsScheduled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	future := now.Add(48 * time.Hour)
	past := now.Add(-48 * time.Hour)

	tests := []struct {
		name          string
		draft         bool
		publishDate   time.Time
		includeDrafts bool
		includeFuture bool
		want          bool
	}{
		{"published, not draft", false, past, false, false, false},
		{"draft excluded", true, past, false, false, true},
		{"draft included", true, past, true, false, false},
		{"future excluded", false, future, false, false, true},
		{"future included", false, future, false, true, false},
		{"draft+future, both excluded", true, future, false, false, true},
		{"draft+future, both included", true, future, true, true, false},
		{"no publish date, not draft", false, time.Time{}, false, false, false},
		{"no publish date, draft", true, time.Time{}, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldExclude(tt.draft, tt.publishDate, tt.includeDrafts, tt.includeFuture, now)
			if got != tt.want {
				t.Errorf("ShouldExclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

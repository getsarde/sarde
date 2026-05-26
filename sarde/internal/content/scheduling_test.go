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

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name       string
		expiryDate time.Time
		want       bool
	}{
		{"zero date", time.Time{}, false},
		{"future date", now.Add(24 * time.Hour), false},
		{"past date", now.Add(-24 * time.Hour), true},
		{"exact now", now, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExpired(tt.expiryDate, now); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	future := now.Add(48 * time.Hour)
	past := now.Add(-48 * time.Hour)
	zero := time.Time{}

	tests := []struct {
		name           string
		draft          bool
		publishDate    time.Time
		expiryDate     time.Time
		includeDrafts  bool
		includeFuture  bool
		includeExpired bool
		want           bool
	}{
		{"published, not draft", false, past, zero, false, false, false, false},
		{"draft excluded", true, past, zero, false, false, false, true},
		{"draft included", true, past, zero, true, false, false, false},
		{"future excluded", false, future, zero, false, false, false, true},
		{"future included", false, future, zero, false, true, false, false},
		{"draft+future, both excluded", true, future, zero, false, false, false, true},
		{"draft+future, both included", true, future, zero, true, true, false, false},
		{"no publish date, not draft", false, zero, zero, false, false, false, false},
		{"no publish date, draft", true, zero, zero, false, false, false, true},
		{"expired excluded", false, past, past, false, false, false, true},
		{"expired included", false, past, past, false, false, true, false},
		{"not yet expired", false, past, future, false, false, false, false},
		{"expired + draft, both excluded", true, past, past, false, false, false, true},
		{"expired + draft, both included", true, past, past, true, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldExclude(tt.draft, tt.publishDate, tt.expiryDate, tt.includeDrafts, tt.includeFuture, tt.includeExpired, now)
			if got != tt.want {
				t.Errorf("ShouldExclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

package engine

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestFlexDate_UnmarshalYAML_Unset(t *testing.T) {
	// An editor clearing a date field writes one of these. None may error.
	for _, input := range []string{`""`, `''`, `"   "`, `null`, `~`, ``} {
		var fm FrontmatterIdentity
		if err := yaml.Unmarshal([]byte("publish_date: "+input), &fm); err != nil {
			t.Fatalf("publish_date: %s returned error: %v", input, err)
		}
		if !fm.PublishDate.IsZero() {
			t.Errorf("publish_date: %s = %v, want zero", input, fm.PublishDate.Time)
		}
	}
}

func TestFlexDate_UnmarshalYAML_Formats(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2026-07-24", time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)},
		{"2026-07-24T10:30:00Z", time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)},
		{`"2026-07-24"`, time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)},
		{`"2026-07-24 10:30:00"`, time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		var fm FrontmatterIdentity
		if err := yaml.Unmarshal([]byte("date: "+tc.input), &fm); err != nil {
			t.Fatalf("date: %s returned error: %v", tc.input, err)
		}
		if !fm.Date.Equal(tc.want) {
			t.Errorf("date: %s = %v, want %v", tc.input, fm.Date.Time, tc.want)
		}
	}
}

func TestFlexDate_UnmarshalYAML_Invalid(t *testing.T) {
	var fm FrontmatterIdentity
	err := yaml.Unmarshal([]byte(`date: "not a date"`), &fm)
	if err == nil {
		t.Fatal("expected an error for an unparseable date, got nil")
	}
}

package engine

import "testing"

func TestValidateLayout(t *testing.T) {
	valid := []LayoutType{LayoutDefault, LayoutDocs, LayoutSplash, LayoutWide, LayoutFull, LayoutCentered}
	for _, lt := range valid {
		if !ValidateLayout(lt) {
			t.Errorf("ValidateLayout(%q) = false, want true", lt)
		}
	}
	if ValidateLayout("nonexistent") {
		t.Error("ValidateLayout(nonexistent) = true, want false")
	}
}

func TestResolveLayout(t *testing.T) {
	tests := []struct {
		input string
		want  LayoutType
	}{
		{"docs", LayoutDocs},
		{"wide", LayoutWide},
		{"full", LayoutFull},
		{"centered", LayoutCentered},
		{"splash", LayoutSplash},
		{"default", LayoutDefault},
		{"", LayoutDefault},
		{"invalid", LayoutDefault},
	}
	for _, tt := range tests {
		got := ResolveLayout(tt.input)
		if got != tt.want {
			t.Errorf("ResolveLayout(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLayoutHasSidebar(t *testing.T) {
	tests := []struct {
		layout LayoutType
		want   bool
	}{
		{LayoutDocs, true},
		{LayoutWide, true},
		{LayoutDefault, false},
		{LayoutSplash, false},
		{LayoutFull, false},
		{LayoutCentered, false},
	}
	for _, tt := range tests {
		if got := LayoutHasSidebar(tt.layout); got != tt.want {
			t.Errorf("LayoutHasSidebar(%q) = %v, want %v", tt.layout, got, tt.want)
		}
	}
}

func TestLayoutHasTOC(t *testing.T) {
	tests := []struct {
		layout LayoutType
		want   bool
	}{
		{LayoutDocs, true},
		{LayoutWide, false},
		{LayoutDefault, false},
		{LayoutFull, false},
	}
	for _, tt := range tests {
		if got := LayoutHasTOC(tt.layout); got != tt.want {
			t.Errorf("LayoutHasTOC(%q) = %v, want %v", tt.layout, got, tt.want)
		}
	}
}

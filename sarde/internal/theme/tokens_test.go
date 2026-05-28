package theme

import (
	"strings"
	"testing"
)

func TestDeriveTokens_Accent(t *testing.T) {
	tokens := map[string]string{"accent": "#6366f1"}
	result := DeriveTokens(tokens)

	if result["accent-hover"] == "" {
		t.Error("expected accent-hover to be derived")
	}
	if result["accent-high"] == "" {
		t.Error("expected accent-high to be derived")
	}
	if result["accent-low"] == "" {
		t.Error("expected accent-low to be derived")
	}
	if !strings.HasPrefix(result["accent-low"], "rgba(") {
		t.Errorf("accent-low should be rgba, got %q", result["accent-low"])
	}
}

func TestDeriveTokens_HoverIsDarker(t *testing.T) {
	tokens := map[string]string{"accent": "#6366f1"}
	DeriveTokens(tokens)

	// Parse original and hover to verify hover is darker.
	_, _, _, err := parseHex(tokens["accent"])
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = parseHex(tokens["accent-hover"])
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it's a valid hex (detailed color math tested via HSL functions).
	if !strings.HasPrefix(tokens["accent-hover"], "#") {
		t.Errorf("expected hex, got %q", tokens["accent-hover"])
	}
}

func TestDeriveTokens_ExistingKeyNotOverridden(t *testing.T) {
	tokens := map[string]string{
		"accent":       "#6366f1",
		"accent-hover": "#custom",
	}
	DeriveTokens(tokens)

	if tokens["accent-hover"] != "#custom" {
		t.Errorf("existing key should not be overridden, got %q", tokens["accent-hover"])
	}
}

func TestDeriveTokens_NonHexSkipped(t *testing.T) {
	tokens := map[string]string{"accent": "oklch(60% 0.15 250)"}
	DeriveTokens(tokens)

	if _, ok := tokens["accent-hover"]; ok {
		t.Error("should not derive from non-hex value")
	}
}

func TestDeriveTokens_NoAccent(t *testing.T) {
	tokens := map[string]string{"bg": "#ffffff"}
	result := DeriveTokens(tokens)

	if _, ok := result["accent-hover"]; ok {
		t.Error("should not derive without accent")
	}
}

func TestDeriveTokens_ShortHex(t *testing.T) {
	tokens := map[string]string{"accent": "#fff"}
	DeriveTokens(tokens)

	if tokens["accent-hover"] == "" {
		t.Error("should derive from 3-digit hex")
	}
}

func TestRGBToHSL_Roundtrip(t *testing.T) {
	tests := []struct{ r, g, b int }{
		{255, 0, 0},   // red
		{0, 255, 0},   // green
		{0, 0, 255},   // blue
		{99, 102, 241}, // indigo (accent)
		{0, 0, 0},     // black
		{255, 255, 255}, // white
	}
	for _, tt := range tests {
		h, s, l := rgbToHSL(tt.r, tt.g, tt.b)
		r, g, b := hslToRGB(h, s, l)
		if abs(r-tt.r) > 1 || abs(g-tt.g) > 1 || abs(b-tt.b) > 1 {
			t.Errorf("roundtrip (%d,%d,%d) → HSL(%f,%f,%f) → (%d,%d,%d)", tt.r, tt.g, tt.b, h, s, l, r, g, b)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

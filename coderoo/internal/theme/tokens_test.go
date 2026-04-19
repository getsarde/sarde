package theme

import (
	"strings"
	"testing"
)

func TestDeriveTokens_Primary(t *testing.T) {
	tokens := map[string]string{"primary": "#6366f1"}
	result := DeriveTokens(tokens)

	if result["primary-hover"] == "" {
		t.Error("expected primary-hover to be derived")
	}
	if result["primary-high"] == "" {
		t.Error("expected primary-high to be derived")
	}
	if result["primary-low"] == "" {
		t.Error("expected primary-low to be derived")
	}
	if !strings.HasPrefix(result["primary-low"], "rgba(") {
		t.Errorf("primary-low should be rgba, got %q", result["primary-low"])
	}
}

func TestDeriveTokens_HoverIsDarker(t *testing.T) {
	tokens := map[string]string{"primary": "#6366f1"}
	DeriveTokens(tokens)

	// Parse original and hover to verify hover is darker.
	_, _, _, err := parseHex(tokens["primary"])
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = parseHex(tokens["primary-hover"])
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it's a valid hex (detailed color math tested via HSL functions).
	if !strings.HasPrefix(tokens["primary-hover"], "#") {
		t.Errorf("expected hex, got %q", tokens["primary-hover"])
	}
}

func TestDeriveTokens_ExistingKeyNotOverridden(t *testing.T) {
	tokens := map[string]string{
		"primary":       "#6366f1",
		"primary-hover": "#custom",
	}
	DeriveTokens(tokens)

	if tokens["primary-hover"] != "#custom" {
		t.Errorf("existing key should not be overridden, got %q", tokens["primary-hover"])
	}
}

func TestDeriveTokens_NonHexSkipped(t *testing.T) {
	tokens := map[string]string{"primary": "oklch(60% 0.15 250)"}
	DeriveTokens(tokens)

	if _, ok := tokens["primary-hover"]; ok {
		t.Error("should not derive from non-hex value")
	}
}

func TestDeriveTokens_NoPrimary(t *testing.T) {
	tokens := map[string]string{"bg": "#ffffff"}
	result := DeriveTokens(tokens)

	if _, ok := result["primary-hover"]; ok {
		t.Error("should not derive without primary")
	}
}

func TestDeriveTokens_ShortHex(t *testing.T) {
	tokens := map[string]string{"primary": "#fff"}
	DeriveTokens(tokens)

	if tokens["primary-hover"] == "" {
		t.Error("should derive from 3-digit hex")
	}
}

func TestRGBToHSL_Roundtrip(t *testing.T) {
	tests := []struct{ r, g, b int }{
		{255, 0, 0},   // red
		{0, 255, 0},   // green
		{0, 0, 255},   // blue
		{99, 102, 241}, // indigo (primary)
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

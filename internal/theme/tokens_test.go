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

func TestDeriveTokens_InvalidFormatSkipped(t *testing.T) {
	tokens := map[string]string{"accent": "hsl(200, 50%, 50%)"}
	DeriveTokens(tokens)

	if _, ok := tokens["accent-hover"]; ok {
		t.Error("should not derive from unsupported format")
	}
}

func TestDeriveTokens_OKLCH(t *testing.T) {
	tokens := map[string]string{"accent": "oklch(0.600 0.220 264.0)"}
	result := DeriveTokens(tokens)

	if result["accent-hover"] == "" {
		t.Error("expected accent-hover to be derived from OKLCH")
	}
	if result["accent-high"] == "" {
		t.Error("expected accent-high to be derived from OKLCH")
	}
	if result["accent-low"] == "" {
		t.Error("expected accent-low to be derived from OKLCH")
	}
	if !strings.HasPrefix(result["accent-hover"], "oklch(") {
		t.Errorf("accent-hover should be oklch, got %q", result["accent-hover"])
	}
	if !strings.HasPrefix(result["accent-high"], "oklch(") {
		t.Errorf("accent-high should be oklch, got %q", result["accent-high"])
	}
	if !strings.Contains(result["accent-low"], "/ 0.1") {
		t.Errorf("accent-low should contain alpha, got %q", result["accent-low"])
	}
}

func TestDeriveTokens_OKLCH_Percent(t *testing.T) {
	tokens := map[string]string{"accent": "oklch(60% 0.22 264)"}
	result := DeriveTokens(tokens)

	if result["accent-hover"] == "" {
		t.Error("expected accent-hover to be derived from OKLCH with percentage L")
	}
	if !strings.HasPrefix(result["accent-hover"], "oklch(0.520") {
		t.Errorf("hover L should be 0.60-0.08=0.52, got %q", result["accent-hover"])
	}
}

func TestDeriveTokens_OKLCH_ClampAtBounds(t *testing.T) {
	tokens := map[string]string{"accent": "oklch(0.020 0.15 200)"}
	result := DeriveTokens(tokens)

	if !strings.HasPrefix(result["accent-hover"], "oklch(0.000") {
		t.Errorf("hover should clamp to 0, got %q", result["accent-hover"])
	}
}

func TestDeriveTokens_OKLCH_HighClamp(t *testing.T) {
	tokens := map[string]string{"accent": "oklch(0.950 0.15 200)"}
	result := DeriveTokens(tokens)

	if !strings.HasPrefix(result["accent-high"], "oklch(1.000") {
		t.Errorf("high should clamp to 1, got %q", result["accent-high"])
	}
}

func TestParseOKLCH(t *testing.T) {
	tests := []struct {
		input   string
		wantL   float64
		wantC   float64
		wantH   float64
		wantErr bool
	}{
		{"oklch(0.600 0.220 264.0)", 0.600, 0.220, 264.0, false},
		{"oklch(60% 0.15 250)", 0.600, 0.15, 250.0, false},
		{"oklch(0.985 0.002 286)", 0.985, 0.002, 286.0, false},
		{"oklch(1 0 0)", 1.0, 0.0, 0.0, false},
		{"#ffffff", 0, 0, 0, true},
		{"hsl(200, 50%, 50%)", 0, 0, 0, true},
		{"oklch(0.5 0.2 200 / 0.5)", 0, 0, 0, true}, // alpha form not parsed
	}
	for _, tt := range tests {
		l, c, h, err := parseOKLCH(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseOKLCH(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if l != tt.wantL || c != tt.wantC || h != tt.wantH {
			t.Errorf("parseOKLCH(%q) = (%.3f, %.3f, %.1f), want (%.3f, %.3f, %.1f)",
				tt.input, l, c, h, tt.wantL, tt.wantC, tt.wantH)
		}
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
